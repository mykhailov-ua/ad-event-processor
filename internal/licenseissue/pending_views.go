package licenseissue

import (
	"fmt"
	"strings"

	"ad-event-processor/internal/trialregistry"
)

type PendingItemView struct {
	ID               string `json:"id"`
	TelegramID       string `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username,omitempty"`
	RequestedAt      string `json:"requested_at"`
	Status           string `json:"status"`
	OfferVersion     string `json:"offer_version,omitempty"`
	OfferAcceptedAt  string `json:"offer_accepted_at,omitempty"`
	AcceptSource     string `json:"accept_source,omitempty"`
	Notes            string `json:"notes,omitempty"`
	Summary          string `json:"summary"`
	ApproveCallback  string `json:"approve_callback_data"`
	RejectCallback   string `json:"reject_callback_data"`
}

type PendingListResponse struct {
	Items           []PendingItemView  `json:"items"`
	AdminButtonRows [][]TelegramButton `json:"admin_button_rows"`
	EmptyMessage    string             `json:"empty_message,omitempty"`
}

func PendingListView(items []trialregistry.PendingRequest) PendingListResponse {
	out := PendingListResponse{
		Items: make([]PendingItemView, 0, len(items)),
	}
	if len(items) == 0 {
		out.EmptyMessage = "No open pilot requests."
		return out
	}
	rows := make([][]TelegramButton, 0, len(items))
	for _, item := range items {
		view := pendingItemView(item)
		out.Items = append(out.Items, view)
		rows = append(rows, []TelegramButton{
			{Text: "Approve " + view.Summary, CallbackData: view.ApproveCallback},
			{Text: "Reject", CallbackData: view.RejectCallback},
		})
	}
	out.AdminButtonRows = rows
	return out
}

func pendingItemView(item trialregistry.PendingRequest) PendingItemView {
	user := strings.TrimSpace(item.TelegramUsername)
	if user != "" {
		user = "@" + strings.TrimPrefix(user, "@")
	} else {
		user = item.TelegramID
	}
	summary := fmt.Sprintf("%s (%s)", user, shortID(item.ID))
	view := PendingItemView{
		ID:               item.ID,
		TelegramID:       item.TelegramID,
		TelegramUsername: item.TelegramUsername,
		RequestedAt:      item.RequestedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Status:           string(item.Status),
		OfferVersion:     strings.TrimSpace(item.OfferVersion),
		Notes:            item.Notes,
		Summary:          summary,
		ApproveCallback:  "pending:issue:" + item.ID,
		RejectCallback:   "pending:reject:" + item.ID,
	}
	if !item.OfferAcceptedAt.IsZero() {
		view.OfferAcceptedAt = item.OfferAcceptedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	view.AcceptSource = strings.TrimSpace(item.AcceptSource)
	return view
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
