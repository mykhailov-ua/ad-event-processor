package reportjob

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportJobNotifySpec struct {
	Channel    string `json:"channel,omitempty"`
	Email      string `json:"email,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

type ReportExportNotificationDTO struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	CustomerID string `json:"customer_id"`
	ReportKey  string `json:"report_key,omitempty"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Read       bool   `json:"read"`
	CreatedAt  string `json:"created_at"`
}

type ReportExportNotificationListDTO struct {
	Rows        []ReportExportNotificationDTO `json:"rows"`
	UnreadCount int                           `json:"unread_count"`
}

type exportNotificationRecord struct {
	id, jobID, userID, customerID, reportKey, kind, title, body string
	readAt                                                      *time.Time
	createdAt                                                   time.Time
}

func validateReportJobCompareSpec(spec ReportJobSpec) error {
	compareFrom := strings.TrimSpace(spec.CompareFrom)
	compareTo := strings.TrimSpace(spec.CompareTo)
	if compareFrom == "" && compareTo == "" {
		return nil
	}
	if compareFrom == "" || compareTo == "" {
		return fmt.Errorf("compare_from and compare_to must both be set")
	}
	_, _, err := ParseReportRangeFromStrings(compareFrom, compareTo)
	return err
}

// ValidateReportJobCompareStrings validates optional compare range timestamps for saved views and schedules.
func ValidateReportJobCompareStrings(compareFrom, compareTo string) error {
	return validateReportJobCompareSpec(ReportJobSpec{
		CompareFrom: compareFrom,
		CompareTo:   compareTo,
	})
}

func normalizeReportJobNotify(spec *ReportJobSpec) error {
	if spec == nil {
		return fmt.Errorf("report job spec required")
	}
	channel := strings.TrimSpace(spec.Notify.Channel)
	if channel == "" {
		channel = "none"
	}
	spec.Notify.Channel = channel
	switch channel {
	case "none", "in_app":
		return nil
	case "email":
		email := strings.TrimSpace(spec.Notify.Email)
		if email == "" {
			return fmt.Errorf("notify.email required for email channel")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("notify.email must be a valid email address")
		}
		spec.Notify.Email = email
		return nil
	case "slack_webhook":
		webhookURL := strings.TrimSpace(spec.Notify.WebhookURL)
		if webhookURL == "" {
			return fmt.Errorf("notify.webhook_url required for slack_webhook channel")
		}
		if !strings.HasPrefix(webhookURL, "https://") {
			return fmt.Errorf("notify.webhook_url must use https")
		}
		spec.Notify.WebhookURL = webhookURL
		return nil
	default:
		return fmt.Errorf("notify.channel must be none, in_app, email, or slack_webhook")
	}
}

// ValidateReportJobNotifySpec validates notify settings without mutating the caller-owned spec.
func ValidateReportJobNotifySpec(spec *ReportJobSpec) error {
	if spec == nil {
		return fmt.Errorf("report job spec required")
	}
	copy := *spec
	return normalizeReportJobNotify(&copy)
}

func (r *ReportJobRunner) notifyJobTerminal(ctx context.Context, jobID string, spec ReportJobSpec, terminalStatus, publicErr string) {
	if r == nil {
		return
	}
	channel := strings.TrimSpace(spec.Notify.Channel)
	if channel == "" {
		channel = "none"
	}
	if channel == "none" {
		return
	}
	userID := strings.TrimSpace(spec.ExportedBy)
	if userID == "" {
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		return
	}
	if channel == "in_app" {
		kind := exportNotificationKindCompleted
		title := fmt.Sprintf("Export ready: %s", spec.ReportKey)
		body := fmt.Sprintf("Report export job %s completed.", jobID)
		if terminalStatus == JobStatusFailed {
			kind = exportNotificationKindFailed
			title = fmt.Sprintf("Export failed: %s", spec.ReportKey)
			if publicErr != "" {
				body = publicErr
			} else {
				body = fmt.Sprintf("Report export job %s failed.", jobID)
			}
		}
		_ = r.insertExportNotification(ctx, jobID, userID, spec.CustomerID, spec.ReportKey, kind, title, body)
	}
}

const (
	exportNotificationKindCompleted = "completed"
	exportNotificationKindFailed    = "failed"
)

func (r *ReportJobRunner) insertExportNotification(
	ctx context.Context,
	jobID, userID, customerID, reportKey, kind, title, body string,
) error {
	if r.pgEnabled() {
		return insertExportNotificationPG(ctx, r.deps.Pool, jobID, userID, customerID, reportKey, kind, title, body)
	}
	r.notifMu.Lock()
	defer r.notifMu.Unlock()
	dedupKey := jobID + "\x00" + userID
	if _, ok := r.notifByJobUser[dedupKey]; ok {
		return nil
	}
	rec := exportNotificationRecord{
		id:         uuid.New().String(),
		jobID:      jobID,
		userID:     userID,
		customerID: customerID,
		reportKey:  reportKey,
		kind:       kind,
		title:      title,
		body:       body,
		createdAt:  time.Now().UTC(),
	}
	r.notifs = append([]exportNotificationRecord{rec}, r.notifs...)
	r.notifByJobUser[dedupKey] = rec.id
	return nil
}

func insertExportNotificationPG(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobID, userID, customerID, reportKey, kind, title, body string,
) error {
	if pool == nil {
		return fmt.Errorf("report notification store unavailable")
	}
	parsedJobID, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	parsedCustomerID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO report_export_notifications (job_id, user_id, customer_id, report_key, kind, title, body)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7)
ON CONFLICT (job_id, user_id) DO NOTHING`,
		parsedJobID, parsedUserID, parsedCustomerID, reportKey, kind, title, body,
	)
	return err
}

func (r *ReportJobRunner) ListExportNotifications(ctx context.Context, userID string, limit int) (ReportExportNotificationListDTO, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	parsedUser, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return ReportExportNotificationListDTO{}, fmt.Errorf("invalid user id")
	}
	if r.pgEnabled() {
		return listExportNotificationsPG(ctx, r.deps.Pool, parsedUser.String(), limit)
	}
	r.notifMu.RLock()
	defer r.notifMu.RUnlock()
	rows := make([]ReportExportNotificationDTO, 0, limit)
	unread := 0
	for _, rec := range r.notifs {
		if rec.userID != parsedUser.String() {
			continue
		}
		read := rec.readAt != nil
		if !read {
			unread++
		}
		rows = append(rows, exportNotificationToDTO(rec))
		if len(rows) >= limit {
			break
		}
	}
	return ReportExportNotificationListDTO{Rows: rows, UnreadCount: unread}, nil
}

func listExportNotificationsPG(ctx context.Context, pool *pgxpool.Pool, userID string, limit int) (ReportExportNotificationListDTO, error) {
	parsedUser, err := uuid.Parse(userID)
	if err != nil {
		return ReportExportNotificationListDTO{}, err
	}
	rows, err := pool.Query(ctx, `
SELECT id, job_id, customer_id, COALESCE(report_key, ''), kind, title, body, read_at, created_at
FROM report_export_notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2`, parsedUser, limit)
	if err != nil {
		return ReportExportNotificationListDTO{}, err
	}
	defer rows.Close()
	out := make([]ReportExportNotificationDTO, 0, limit)
	for rows.Next() {
		var (
			parsedID, parsedJobID, parsedCustomerID uuid.UUID
			reportKey, kind, title, body            string
			readAt                                  *time.Time
			createdAt                               time.Time
		)
		if err := rows.Scan(&parsedID, &parsedJobID, &parsedCustomerID, &reportKey, &kind, &title, &body, &readAt, &createdAt); err != nil {
			return ReportExportNotificationListDTO{}, err
		}
		out = append(out, ReportExportNotificationDTO{
			ID:         parsedID.String(),
			JobID:      parsedJobID.String(),
			CustomerID: parsedCustomerID.String(),
			ReportKey:  reportKey,
			Kind:       kind,
			Title:      title,
			Body:       body,
			Read:       readAt != nil,
			CreatedAt:  createdAt.UTC().Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		return ReportExportNotificationListDTO{}, err
	}
	var unread int
	err = pool.QueryRow(ctx, `
SELECT count(*) FROM report_export_notifications
WHERE user_id = $1 AND read_at IS NULL`, parsedUser).Scan(&unread)
	if err != nil {
		return ReportExportNotificationListDTO{}, err
	}
	return ReportExportNotificationListDTO{Rows: out, UnreadCount: unread}, nil
}

func exportNotificationToDTO(rec exportNotificationRecord) ReportExportNotificationDTO {
	return ReportExportNotificationDTO{
		ID:         rec.id,
		JobID:      rec.jobID,
		CustomerID: rec.customerID,
		ReportKey:  rec.reportKey,
		Kind:       rec.kind,
		Title:      rec.title,
		Body:       rec.body,
		Read:       rec.readAt != nil,
		CreatedAt:  rec.createdAt.UTC().Format(time.RFC3339),
	}
}

func (r *ReportJobRunner) AckExportNotification(ctx context.Context, userID, notificationID string) (ReportExportNotificationDTO, bool, error) {
	parsedUser, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return ReportExportNotificationDTO{}, false, fmt.Errorf("invalid user id")
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(notificationID))
	if err != nil {
		return ReportExportNotificationDTO{}, false, nil
	}
	if r.pgEnabled() {
		return ackExportNotificationPG(ctx, r.deps.Pool, parsedUser.String(), parsedID.String())
	}
	r.notifMu.Lock()
	defer r.notifMu.Unlock()
	for i := range r.notifs {
		if r.notifs[i].id != parsedID.String() || r.notifs[i].userID != parsedUser.String() {
			continue
		}
		now := time.Now().UTC()
		r.notifs[i].readAt = &now
		dto := exportNotificationToDTO(r.notifs[i])
		return dto, true, nil
	}
	return ReportExportNotificationDTO{}, false, nil
}

func ackExportNotificationPG(ctx context.Context, pool *pgxpool.Pool, userID, notificationID string) (ReportExportNotificationDTO, bool, error) {
	parsedUser, err := uuid.Parse(userID)
	if err != nil {
		return ReportExportNotificationDTO{}, false, err
	}
	parsedID, err := uuid.Parse(notificationID)
	if err != nil {
		return ReportExportNotificationDTO{}, false, nil
	}
	var dto ReportExportNotificationDTO
	var reportKey string
	var readAt *time.Time
	var createdAt time.Time
	var jobID, customerID uuid.UUID
	err = pool.QueryRow(ctx, `
UPDATE report_export_notifications
SET read_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING job_id, customer_id, COALESCE(report_key, ''), kind, title, body, read_at, created_at`,
		parsedID, parsedUser,
	).Scan(&jobID, &customerID, &reportKey, &dto.Kind, &dto.Title, &dto.Body, &readAt, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReportExportNotificationDTO{}, false, nil
		}
		return ReportExportNotificationDTO{}, false, err
	}
	dto.ID = parsedID.String()
	dto.JobID = jobID.String()
	dto.CustomerID = customerID.String()
	dto.ReportKey = reportKey
	dto.Read = readAt != nil
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return dto, true, nil
}
