package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bidshard/ad-event-processor/internal/trialregistry"
)

const (
	envBotToken      = trialregistry.EnvVendorTrialBotToken
	envTrialRegistry = trialregistry.EnvRegistryPath
	defaultPollSec   = 30
)

type botConfig struct {
	Token        string
	RegistryPath string
	PollTimeout  time.Duration
	DryRun       bool
}

type tgUpdateResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int     `json:"message_id"`
	Text      string  `json:"text"`
	From      *tgUser `json:"from"`
	Chat      tgChat  `json:"chat"`
}

type tgUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

func runBot(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registryPath := fs.String("trial-registry", "", "trial registry file path override")
	pollSec := fs.Int("poll-timeout", defaultPollSec, "Telegram long-poll timeout seconds")
	dryRun := fs.Bool("dry-run", false, "log updates without calling Telegram or writing registry")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := botConfig{
		Token:       strings.TrimSpace(os.Getenv(envBotToken)),
		PollTimeout: time.Duration(*pollSec) * time.Second,
		DryRun:      *dryRun,
	}
	trialCfg := trialregistry.ConfigFromEnv()
	if path := strings.TrimSpace(*registryPath); path != "" {
		trialCfg.RegistryPath = path
	}
	cfg.RegistryPath = trialCfg.RegistryPath

	if cfg.Token == "" && !cfg.DryRun {
		fmt.Fprintf(os.Stderr, "vendor-trial-bot: %s is required (or use --dry-run)\n", envBotToken)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	reg := trialregistry.NewFromConfig(trialCfg)
	client := &http.Client{Timeout: cfg.PollTimeout + 10*time.Second}

	var offset int64
	slog.Info("vendor-trial-bot started", "registry", cfg.RegistryPath, "dry_run", cfg.DryRun)

	for {
		select {
		case <-ctx.Done():
			slog.Info("vendor-trial-bot stopped")
			return 0
		default:
		}

		updates, err := fetchUpdates(ctx, client, cfg, offset)
		if err != nil {
			if ctx.Err() != nil {
				return 0
			}
			slog.Error("getUpdates failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}
			if upd.Message == nil || upd.Message.From == nil {
				continue
			}
			handleMessage(ctx, client, cfg, reg, *upd.Message)
		}
	}
}

func handleMessage(ctx context.Context, client *http.Client, cfg botConfig, reg *trialregistry.Registry, msg tgMessage) {
	cmd, _ := parseCommand(msg.Text)
	userID := fmt.Sprintf("%d", msg.From.ID)
	chatID := fmt.Sprintf("%d", msg.Chat.ID)

	switch cmd {
	case "start", "help":
		reply := "BidShard pilot signup.\n\nSend /trial to request a pilot license. A vendor operator will review and send your JWT."
		sendBotMessage(ctx, client, cfg, chatID, reply)
	case "trial":
		if cfg.DryRun {
			slog.Info("dry-run trial request", "telegram_id", userID, "username", msg.From.Username)
			return
		}
		req, err := reg.EnqueuePending(trialregistry.EnqueuePendingInput{
			TelegramID:       userID,
			TelegramUsername: msg.From.Username,
		})
		if err != nil {
			slog.Error("enqueue pending failed", "telegram_id", userID, "error", err)
			text := "Could not queue your request. Contact vendor support."
			if err == trialregistry.ErrTrialTelegramUsed {
				text = "A pilot was already issued for this Telegram account."
			}
			sendBotMessage(ctx, client, cfg, chatID, text)
			return
		}
		slog.Info("pending trial queued", "id", req.ID, "telegram_id", userID)
		sendBotMessage(ctx, client, cfg, chatID, fmt.Sprintf(
			"Request queued (id=%s). You will receive a pilot JWT after vendor approval.",
			req.ID,
		))
	default:
		if strings.TrimSpace(msg.Text) != "" {
			sendBotMessage(ctx, client, cfg, chatID, "Unknown command. Send /trial to request a pilot.")
		}
	}
}

func parseCommand(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}
	cmd := strings.TrimPrefix(fields[0], "/")
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i]
	}
	args := ""
	if len(fields) > 1 {
		args = strings.Join(fields[1:], " ")
	}
	return strings.ToLower(cmd), args
}

func fetchUpdates(ctx context.Context, client *http.Client, cfg botConfig, offset int64) ([]tgUpdate, error) {
	if cfg.DryRun {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil, nil
		}
	}

	q := url.Values{}
	q.Set("timeout", fmt.Sprintf("%d", int(cfg.PollTimeout.Seconds())))
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?%s", cfg.Token, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram getUpdates: status=%d body=%s", resp.StatusCode, string(body))
	}

	var payload tgUpdateResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if !payload.OK {
		return nil, fmt.Errorf("telegram getUpdates: ok=false")
	}
	return payload.Result, nil
}

func sendBotMessage(ctx context.Context, client *http.Client, cfg botConfig, chatID, text string) {
	if cfg.DryRun || cfg.Token == "" {
		slog.Info("dry-run send", "chat_id", chatID, "text", text)
		return
	}

	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("sendMessage build failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("sendMessage failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("sendMessage bad status", "status", resp.StatusCode, "body", string(body))
	}
}
