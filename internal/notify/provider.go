package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"espx/internal/config"
	"espx/internal/notify/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Provider interface {
	Send(ctx context.Context, recipient, title, body string) error
	Name() string
}

type MockSentNotification struct {
	Recipient string
	Title     string
	Body      string
	SentAt    time.Time
}

type MockProvider struct {
	breaker      *CircuitBreaker
	ProviderName string
	ShouldFail   bool
	Sent         []MockSentNotification
}

func NewMockProvider(breaker *CircuitBreaker) *MockProvider {
	return &MockProvider{breaker: breaker}
}

func (m *MockProvider) Name() string {
	if m.ProviderName != "" {
		return m.ProviderName
	}
	return "TELEGRAM"
}

func (m *MockProvider) Send(ctx context.Context, recipient, title, body string) error {
	_ = ctx
	if !m.breaker.Allow() {
		return ErrCircuitOpen
	}

	if strings.Contains(body, "trigger_failure") || m.ShouldFail {
		m.breaker.RecordFailure()
		return fmt.Errorf("mock send failure triggered")
	}

	m.Sent = append(m.Sent, MockSentNotification{
		Recipient: recipient,
		Title:     title,
		Body:      body,
		SentAt:    time.Now(),
	})
	m.breaker.RecordSuccess()
	return nil
}

type SlackProvider struct {
	defaultWebhook     string
	breaker            *CircuitBreaker
	requireCredentials bool
	client             *http.Client
}

func NewSlackProvider(defaultWebhook string, breaker *CircuitBreaker, requireCredentials bool) *SlackProvider {
	return &SlackProvider{
		defaultWebhook:     defaultWebhook,
		breaker:            breaker,
		requireCredentials: requireCredentials,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SlackProvider) Name() string {
	return "SLACK"
}

func (s *SlackProvider) Send(ctx context.Context, recipient, title, body string) error {
	if !s.breaker.Allow() {
		return ErrCircuitOpen
	}

	webhookURL := recipient
	if webhookURL == "" {
		webhookURL = s.defaultWebhook
	}

	if webhookURL == "" {
		if s.requireCredentials {
			return fmt.Errorf("slack webhook not configured")
		}
		slog.Info("slack notification dry-run", "title", title, "body", body)
		return nil
	}

	var text string
	if title != "" {
		text = fmt.Sprintf("*%s*\n%s", title, body)
	} else {
		text = body
	}

	payload := map[string]interface{}{}

	notificationID, _ := NotificationIDFromContext(ctx)
	actions := BuildInteractiveActions(notificationID, title, body)
	var buttons []map[string]interface{}

	if actions.AcknowledgeURL != "" {
		buttons = append(buttons, map[string]interface{}{
			"type": "button",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": "✅ Acknowledge",
			},
			"url": actions.AcknowledgeURL,
		})
	}

	if actions.BlockIPURL != "" {
		buttons = append(buttons, map[string]interface{}{
			"type":  "button",
			"style": "danger",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": fmt.Sprintf("🚫 Block IP %s", actions.BlockIP),
			},
			"url": actions.BlockIPURL,
		})
	}

	if len(buttons) > 0 {
		payload["blocks"] = []interface{}{
			map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": text,
				},
			},
			map[string]interface{}{
				"type":     "actions",
				"elements": buttons,
			},
		}
	} else {
		payload["text"] = text
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.breaker.RecordFailure()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(resp.Body)
		s.breaker.RecordFailure()
		if readErr != nil {
			return fmt.Errorf("slack webhook returned status %d: read body: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	s.breaker.RecordSuccess()
	return nil
}

type TelegramProvider struct {
	botToken           string
	defaultID          string
	breaker            *CircuitBreaker
	requireCredentials bool
	client             *http.Client
}

func NewTelegramProvider(botToken, defaultID string, breaker *CircuitBreaker, requireCredentials bool) *TelegramProvider {
	return &TelegramProvider{
		botToken:           botToken,
		defaultID:          defaultID,
		breaker:            breaker,
		requireCredentials: requireCredentials,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *TelegramProvider) Name() string {
	return "TELEGRAM"
}

func (t *TelegramProvider) Send(ctx context.Context, recipient, title, body string) error {
	if !t.breaker.Allow() {
		return ErrCircuitOpen
	}

	chatID := recipient
	if chatID == "" {
		chatID = t.defaultID
	}

	if t.botToken == "" || chatID == "" {
		if t.requireCredentials {
			return fmt.Errorf("telegram credentials not configured")
		}
		slog.Info("telegram notification dry-run", "title", title, "body", body)
		return nil
	}

	var htmlMessage string
	if title != "" {
		htmlMessage = fmt.Sprintf("<b>%s</b>\n\n%s", title, body)
	} else {
		htmlMessage = body
	}

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       htmlMessage,
		"parse_mode": "HTML",
	}

	notificationID, _ := NotificationIDFromContext(ctx)
	actions := BuildInteractiveActions(notificationID, title, body)
	var inlineKeyboard [][]map[string]interface{}

	if actions.AcknowledgeURL != "" {
		inlineKeyboard = append(inlineKeyboard, []map[string]interface{}{
			{
				"text": "✅ Acknowledge Incident",
				"url":  actions.AcknowledgeURL,
			},
		})
	}

	if actions.BlockIPURL != "" {
		inlineKeyboard = append(inlineKeyboard, []map[string]interface{}{
			{
				"text": fmt.Sprintf("🚫 Block IP %s", actions.BlockIP),
				"url":  actions.BlockIPURL,
			},
		})
	}

	if len(inlineKeyboard) > 0 {
		payload["reply_markup"] = map[string]interface{}{
			"inline_keyboard": inlineKeyboard,
		}
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		t.breaker.RecordFailure()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseTelegramRetryAfter(resp, respBody)
			slog.Warn("telegram api rate limited", "retry_after", retryAfter)
			return &ProviderRateLimitedError{Provider: "TELEGRAM", RetryAfter: retryAfter}
		}
		t.breaker.RecordFailure()
		if readErr != nil {
			return fmt.Errorf("telegram api returned status %d: read body: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("telegram api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	t.breaker.RecordSuccess()
	return nil
}

func parseTelegramRetryAfter(resp *http.Response, body []byte) time.Duration {
	if resp != nil {
		if header := resp.Header.Get("Retry-After"); header != "" {
			if sec, err := strconv.Atoi(header); err == nil && sec > 0 {
				return time.Duration(sec) * time.Second
			}
		}
	}
	var apiResp struct {
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if len(body) > 0 && json.Unmarshal(body, &apiResp) == nil && apiResp.Parameters.RetryAfter > 0 {
		return time.Duration(apiResp.Parameters.RetryAfter) * time.Second
	}
	return 30 * time.Second
}

type SMSProvider struct {
	providerURL        string
	apiToken           string
	defaultRecipient   string
	breaker            *CircuitBreaker
	requireCredentials bool
	client             *http.Client
}

func NewSMSProvider(providerURL, apiToken, defaultRecipient string, breaker *CircuitBreaker, requireCredentials bool) *SMSProvider {
	return &SMSProvider{
		providerURL:        providerURL,
		apiToken:           apiToken,
		defaultRecipient:   defaultRecipient,
		breaker:            breaker,
		requireCredentials: requireCredentials,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SMSProvider) Name() string {
	return "SMS"
}

func (s *SMSProvider) Send(ctx context.Context, recipient, title, body string) error {
	if !s.breaker.Allow() {
		return ErrCircuitOpen
	}

	phone := recipient
	if phone == "" {
		phone = s.defaultRecipient
	}

	if s.providerURL == "" || phone == "" {
		if s.requireCredentials {
			return fmt.Errorf("sms credentials not configured")
		}
		slog.Info("sms notification dry-run", "to", phone, "title", title, "body", body)
		return nil
	}

	var text string
	if title != "" {
		text = fmt.Sprintf("[%s] %s", title, body)
	} else {
		text = body
	}

	payload := map[string]interface{}{
		"to":      phone,
		"message": text,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.providerURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiToken))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.breaker.RecordFailure()
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, readErr := io.ReadAll(resp.Body)
		s.breaker.RecordFailure()
		if readErr != nil {
			return fmt.Errorf("sms api returned status %d: read body: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("sms api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	s.breaker.RecordSuccess()
	return nil
}

type SMTPProvider struct {
	host               string
	port               string
	username           string
	password           string
	sender             string
	breaker            *CircuitBreaker
	requireCredentials bool
}

func NewSMTPProvider(host, port, username, password, sender string, breaker *CircuitBreaker, requireCredentials bool) *SMTPProvider {
	if port == "" {
		port = "587"
	}
	return &SMTPProvider{
		host:               host,
		port:               port,
		username:           username,
		password:           password,
		sender:             sender,
		breaker:            breaker,
		requireCredentials: requireCredentials,
	}
}

func (s *SMTPProvider) Name() string {
	return "SMTP"
}

func (s *SMTPProvider) Send(ctx context.Context, recipient, title, body string) error {
	if !s.breaker.Allow() {
		return ErrCircuitOpen
	}

	if recipient == "" {
		slog.Warn("smtp notification skipped: recipient is required")
		return fmt.Errorf("smtp recipient is required")
	}

	if s.host == "" || s.sender == "" {
		if s.requireCredentials {
			return fmt.Errorf("smtp credentials not configured")
		}
		slog.Info("smtp notification dry-run", "to", recipient, "title", title, "body", body)
		return nil
	}

	subject := title
	if subject == "" {
		subject = "Notification Alert"
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s\r\n", recipient, s.sender, subject, body))

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- smtp.SendMail(addr, auth, s.sender, []string{recipient}, msg)
	}()

	select {
	case <-ctx.Done():
		s.breaker.RecordFailure()
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			s.breaker.RecordFailure()
			return err
		}
	}

	s.breaker.RecordSuccess()
	return nil
}

type ProviderBundle struct {
	Providers map[db.NotifierProvider]Provider
	Breakers  map[db.NotifierProvider]*CircuitBreaker
}

func isProdEnv(env string) bool {
	return env == "production" || env == "prod"
}

func NewProvidersFromConfig(cfg *config.Config) map[db.NotifierProvider]Provider {
	return NewProviderBundleFromConfig(cfg).Providers
}

func NewProviderBundleFromConfig(cfg *config.Config) ProviderBundle {
	if cfg == nil {
		return ProviderBundle{}
	}

	n := cfg.Notifier
	failThreshold := int64(n.BreakerFailThreshold)
	successThreshold := int64(n.BreakerSuccessThreshold)
	openTimeout := time.Duration(n.BreakerOpenTimeoutMs) * time.Millisecond
	requireCredentials := isProdEnv(cfg.Env)

	telegramBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	slackBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	smtpBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)
	smsBreaker := NewCircuitBreaker(failThreshold, successThreshold, openTimeout)

	return ProviderBundle{
		Providers: map[db.NotifierProvider]Provider{
			db.NotifierProviderTELEGRAM: NewTelegramProvider(
				string(n.TelegramBotToken),
				n.TelegramChatID,
				telegramBreaker,
				requireCredentials,
			),
			db.NotifierProviderSLACK: NewSlackProvider(
				string(n.SlackWebhookURL),
				slackBreaker,
				requireCredentials,
			),
			db.NotifierProviderSMTP: NewSMTPProvider(
				n.SMTPHost,
				n.SMTPPort,
				n.SMTPUsername,
				string(n.SMTPPassword),
				n.SMTPSender,
				smtpBreaker,
				requireCredentials,
			),
			db.NotifierProviderSMS: NewSMSProvider(
				n.SMSProviderURL,
				string(n.SMSAPIToken),
				n.SMSDefaultRecipient,
				smsBreaker,
				requireCredentials,
			),
		},
		Breakers: map[db.NotifierProvider]*CircuitBreaker{
			db.NotifierProviderTELEGRAM: telegramBreaker,
			db.NotifierProviderSLACK:    slackBreaker,
			db.NotifierProviderSMTP:     smtpBreaker,
			db.NotifierProviderSMS:      smsBreaker,
		},
	}
}

func StartCircuitBreakerMetricsScraper(ctx context.Context, breakers map[db.NotifierProvider]*CircuitBreaker, interval time.Duration) {
	if len(breakers) == 0 {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}

	scrape := func() {
		for provider, breaker := range breakers {
			if breaker == nil {
				continue
			}
			recordCircuitBreakerState(ProviderDisplayName(provider), breaker.State())
		}
	}

	scrape()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scrape()
		}
	}
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

func MapDBProviderStringsToDB(providers []string) []db.NotifierProvider {
	if len(providers) == 0 {
		return nil
	}
	out := make([]db.NotifierProvider, 0, len(providers))
	for _, provider := range providers {
		out = append(out, db.NotifierProvider(provider))
	}
	return out
}

func pgUUIDFromString(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidNotificationID
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
