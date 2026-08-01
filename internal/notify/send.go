package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"espx/internal/notify/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var defaultHTTPClient = &http.Client{Timeout: 10 * time.Second}

func SendTelegram(ctx context.Context, cfg Config, breaker *CircuitBreaker, recipient, title, body string) error {
	if breaker != nil && !breaker.Allow() {
		return ErrCircuitOpen
	}
	if cfg.FailTelegram || strings.Contains(body, "trigger_failure") {
		recordBreakerFailure(breaker)
		return fmt.Errorf("send failure triggered")
	}

	chatID := recipient
	if chatID == "" {
		chatID = cfg.TelegramChatID
	}
	if cfg.TelegramBotToken == "" || chatID == "" {
		if cfg.RequireCredentials {
			return fmt.Errorf("telegram credentials not configured")
		}
		slog.Info("telegram notification dry-run", "title", title, "body", body)
		recordBreakerSuccess(breaker)
		return nil
	}

	var htmlMessage string
	if title != "" {
		htmlMessage = "<b>" + title + "</b>\n\n" + body
	} else {
		htmlMessage = body
	}

	payload := telegramPayload{
		ChatID:    chatID,
		Text:      htmlMessage,
		ParseMode: "HTML",
	}

	notificationID, _ := NotificationIDFromContext(ctx)
	actions := BuildInteractiveActions(notificationID, title, body)
	if actions.AcknowledgeURL != "" || actions.BlockIPURL != "" {
		var rows []telegramButtonRow
		if actions.AcknowledgeURL != "" {
			rows = append(rows, telegramButtonRow{{Text: "✅ Acknowledge Incident", URL: actions.AcknowledgeURL}})
		}
		if actions.BlockIPURL != "" {
			rows = append(rows, telegramButtonRow{{Text: "🚫 Block IP " + actions.BlockIP, URL: actions.BlockIPURL}})
		}
		payload.ReplyMarkup = &telegramReplyMarkup{InlineKeyboard: rows}
	}

	apiURL := "https://api.telegram.org/bot" + cfg.TelegramBotToken + "/sendMessage"
	respBody, status, err := postJSON(ctx, defaultHTTPClient, apiURL, payload)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if status != http.StatusOK {
		if status == http.StatusTooManyRequests {
			retryAfter := parseTelegramRetryAfter(respBody)
			slog.Warn("telegram api rate limited", "retry_after", retryAfter)
			return &ProviderRateLimitedError{Provider: "TELEGRAM", RetryAfter: retryAfter}
		}
		recordBreakerFailure(breaker)
		return fmt.Errorf("telegram api returned status %d: %s", status, string(respBody))
	}
	recordBreakerSuccess(breaker)
	return nil
}

func SendSlack(ctx context.Context, cfg Config, breaker *CircuitBreaker, recipient, title, body string) error {
	if breaker != nil && !breaker.Allow() {
		return ErrCircuitOpen
	}
	if cfg.FailSlack || strings.Contains(body, "trigger_failure") {
		recordBreakerFailure(breaker)
		return fmt.Errorf("send failure triggered")
	}

	webhookURL := recipient
	if webhookURL == "" {
		webhookURL = cfg.SlackWebhookURL
	}
	if webhookURL == "" {
		if cfg.RequireCredentials {
			return fmt.Errorf("slack webhook not configured")
		}
		slog.Info("slack notification dry-run", "title", title, "body", body)
		recordBreakerSuccess(breaker)
		return nil
	}

	var text string
	if title != "" {
		text = "*" + title + "*\n" + body
	} else {
		text = body
	}

	notificationID, _ := NotificationIDFromContext(ctx)
	actions := BuildInteractiveActions(notificationID, title, body)

	var payload any
	if actions.AcknowledgeURL != "" || actions.BlockIPURL != "" {
		blocks := []slackBlock{
			{Type: "section", Text: &slackText{Type: "mrkdwn", Text: text}},
		}
		var buttons []slackButton
		if actions.AcknowledgeURL != "" {
			buttons = append(buttons, slackButton{Type: "button", Text: slackText{Type: "plain_text", Text: "✅ Acknowledge"}, URL: actions.AcknowledgeURL})
		}
		if actions.BlockIPURL != "" {
			buttons = append(buttons, slackButton{Type: "button", Style: "danger", Text: slackText{Type: "plain_text", Text: "🚫 Block IP " + actions.BlockIP}, URL: actions.BlockIPURL})
		}
		blocks = append(blocks, slackBlock{Type: "actions", Elements: buttons})
		payload = slackBlocksPayload{Blocks: blocks}
	} else {
		payload = slackTextPayload{Text: text}
	}

	respBody, status, err := postJSON(ctx, defaultHTTPClient, webhookURL, payload)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if status != http.StatusOK {
		recordBreakerFailure(breaker)
		return fmt.Errorf("slack webhook returned status %d: %s", status, string(respBody))
	}
	recordBreakerSuccess(breaker)
	return nil
}

func SendSMS(ctx context.Context, cfg Config, breaker *CircuitBreaker, recipient, title, body string) error {
	if breaker != nil && !breaker.Allow() {
		return ErrCircuitOpen
	}
	if cfg.FailSMS || strings.Contains(body, "trigger_failure") {
		recordBreakerFailure(breaker)
		return fmt.Errorf("send failure triggered")
	}

	phone := recipient
	if phone == "" {
		phone = cfg.SMSDefaultRecipient
	}
	if cfg.SMSProviderURL == "" || phone == "" {
		if cfg.RequireCredentials {
			return fmt.Errorf("sms credentials not configured")
		}
		slog.Info("sms notification dry-run", "to", phone, "title", title, "body", body)
		recordBreakerSuccess(breaker)
		return nil
	}

	var text string
	if title != "" {
		text = "[" + title + "] " + body
	} else {
		text = body
	}

	payload := smsPayload{To: phone, Message: text}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.SMSProviderURL, bytes.NewReader(payloadBytes))
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.SMSAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.SMSAPIToken)
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		recordBreakerFailure(breaker)
		return fmt.Errorf("sms api returned status %d: %s", resp.StatusCode, string(respBody))
	}
	recordBreakerSuccess(breaker)
	return nil
}

func SendSMTP(ctx context.Context, cfg Config, breaker *CircuitBreaker, recipient, title, body string) error {
	if breaker != nil && !breaker.Allow() {
		return ErrCircuitOpen
	}
	if cfg.FailSMTP || strings.Contains(body, "trigger_failure") {
		recordBreakerFailure(breaker)
		return fmt.Errorf("send failure triggered")
	}
	if recipient == "" {
		slog.Warn("smtp notification skipped: recipient is required")
		return fmt.Errorf("smtp recipient is required")
	}
	if cfg.SMTPHost == "" || cfg.SMTPSender == "" {
		if cfg.RequireCredentials {
			return fmt.Errorf("smtp credentials not configured")
		}
		slog.Info("smtp notification dry-run", "to", recipient, "title", title, "body", body)
		recordBreakerSuccess(breaker)
		return nil
	}

	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	subject := title
	if subject == "" {
		subject = "Notification Alert"
	}

	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		recipient, cfg.SMTPSender, subject, body))

	addr := net.JoinHostPort(cfg.SMTPHost, port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: cfg.SMTPHost}); err != nil {
			recordBreakerFailure(breaker)
			return err
		}
	}
	if cfg.SMTPUsername != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
		if err = client.Auth(auth); err != nil {
			recordBreakerFailure(breaker)
			return err
		}
	}
	if err = client.Mail(cfg.SMTPSender); err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if err = client.Rcpt(recipient); err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	w, err := client.Data()
	if err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if _, err = w.Write(msg); err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if err = w.Close(); err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	if err = client.Quit(); err != nil {
		recordBreakerFailure(breaker)
		return err
	}
	recordBreakerSuccess(breaker)
	return nil
}

func sendProvider(ctx context.Context, cfg Config, breakers Breakers, provider db.NotifierProvider, recipient, title, body string) error {
	switch provider {
	case db.NotifierProviderTELEGRAM:
		err := SendTelegram(ctx, cfg, breakers.Telegram, recipient, title, body)
		recordBreakerMetrics("TELEGRAM", breakers.Telegram)
		return err
	case db.NotifierProviderSLACK:
		err := SendSlack(ctx, cfg, breakers.Slack, recipient, title, body)
		recordBreakerMetrics("SLACK", breakers.Slack)
		return err
	case db.NotifierProviderSMTP:
		err := SendSMTP(ctx, cfg, breakers.SMTP, recipient, title, body)
		recordBreakerMetrics("SMTP", breakers.SMTP)
		return err
	case db.NotifierProviderSMS:
		err := SendSMS(ctx, cfg, breakers.SMS, recipient, title, body)
		recordBreakerMetrics("SMS", breakers.SMS)
		return err
	default:
		return fmt.Errorf("provider %s not configured", provider)
	}
}

func postJSON(ctx context.Context, client *http.Client, url string, payload any) ([]byte, int, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func recordBreakerSuccess(breaker *CircuitBreaker) {
	if breaker != nil {
		breaker.RecordSuccess()
	}
}

func recordBreakerFailure(breaker *CircuitBreaker) {
	if breaker != nil {
		breaker.RecordFailure()
	}
}

func recordBreakerMetrics(provider string, breaker *CircuitBreaker) {
	if breaker != nil {
		recordCircuitBreakerState(provider, breaker.State())
	}
}

func parseTelegramRetryAfter(body []byte) time.Duration {
	if len(body) > 0 {
		var apiResp struct {
			Parameters struct {
				RetryAfter int `json:"retry_after"`
			} `json:"parameters"`
		}
		if json.Unmarshal(body, &apiResp) == nil && apiResp.Parameters.RetryAfter > 0 {
			return time.Duration(apiResp.Parameters.RetryAfter) * time.Second
		}
	}
	return 30 * time.Second
}

type telegramPayload struct {
	ChatID      string               `json:"chat_id"`
	Text        string               `json:"text"`
	ParseMode   string               `json:"parse_mode"`
	ReplyMarkup *telegramReplyMarkup `json:"reply_markup,omitempty"`
}

type telegramReplyMarkup struct {
	InlineKeyboard []telegramButtonRow `json:"inline_keyboard"`
}

type telegramButtonRow []telegramButton

type telegramButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type slackTextPayload struct {
	Text string `json:"text"`
}

type slackBlocksPayload struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type     string        `json:"type"`
	Text     *slackText    `json:"text,omitempty"`
	Elements []slackButton `json:"elements,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackButton struct {
	Type  string    `json:"type"`
	Style string    `json:"style,omitempty"`
	Text  slackText `json:"text"`
	URL   string    `json:"url"`
}

type smsPayload struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func ParseProviderName(name string) (db.NotifierProvider, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrUnsupportedProvider
	}
	upper := strings.ToUpper(trimmed)
	upper = strings.TrimPrefix(upper, "PROVIDER_")
	switch upper {
	case "TELEGRAM":
		return db.NotifierProviderTELEGRAM, nil
	case "SLACK":
		return db.NotifierProviderSLACK, nil
	case "SMTP":
		return db.NotifierProviderSMTP, nil
	case "SMS":
		return db.NotifierProviderSMS, nil
	default:
		return "", ErrUnsupportedProvider
	}
}

func ProviderDisplayName(provider db.NotifierProvider) string {
	switch provider {
	case db.NotifierProviderTELEGRAM:
		return "TELEGRAM"
	case db.NotifierProviderSLACK:
		return "SLACK"
	case db.NotifierProviderSMTP:
		return "SMTP"
	case db.NotifierProviderSMS:
		return "SMS"
	default:
		return "UNSPECIFIED"
	}
}

func pgUUIDFromString(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidNotificationID
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
