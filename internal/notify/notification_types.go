package notify

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/notify/db"
)

type Notification struct {
	ID                 string
	Provider           string
	Recipient          string
	Title              string
	Body               string
	Status             db.NotifierNotificationStatus
	RetryCount         int32
	ErrorMessage       string
	DeliveryMode       db.NotifierDeliveryMode
	BroadcastProviders []string
	DedupKey           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
