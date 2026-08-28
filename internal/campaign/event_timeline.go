package campaign

import (
	"encoding/json"
	"strings"
	"time"
)

type CampaignEventTimelineItemDTO struct {
	TitleLabel        string `json:"title_label"`
	ActorLabel        string `json:"actor_label"`
	ChangeSummary     string `json:"change_summary"`
	SectionID         string `json:"section_id,omitempty"`
	OccurredAtDisplay string `json:"occurred_at_display"`
}

type CampaignEventTimelineDayDTO struct {
	Day    string                         `json:"day"`
	Events []CampaignEventTimelineItemDTO `json:"events"`
}

type CampaignEventTimelineResponseDTO struct {
	Days []CampaignEventTimelineDayDTO `json:"days"`
}

func buildCampaignEventTimeline(items []CampaignEventDTO, masked bool) CampaignEventTimelineResponseDTO {
	if len(items) == 0 {
		return CampaignEventTimelineResponseDTO{Days: nil}
	}
	byDay := make(map[string][]CampaignEventTimelineItemDTO)
	for _, item := range items {
		day := timelineDayKey(item.CreatedAt)
		byDay[day] = append(byDay[day], campaignEventTimelineItem(item, masked))
	}
	days := make([]CampaignEventTimelineDayDTO, 0, len(byDay))
	for day, events := range byDay {
		days = append(days, CampaignEventTimelineDayDTO{Day: day, Events: events})
	}
	sortTimelineDays(days)
	return CampaignEventTimelineResponseDTO{Days: days}
}

func timelineDayKey(createdAt string) string {
	createdAt = strings.TrimSpace(createdAt)
	if createdAt == "" {
		return "unknown"
	}
	if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
		return ts.UTC().Format("2006-01-02")
	}
	if len(createdAt) >= 10 {
		return createdAt[:10]
	}
	return createdAt
}

func campaignEventTimelineItem(item CampaignEventDTO, masked bool) CampaignEventTimelineItemDTO {
	title := eventTypeTitleLabel(item.EventType)
	summary := eventTypeSummary(item)
	actor := strings.TrimSpace(item.UserID)
	if masked && actor != "" {
		actor = maskActorLabel(actor)
	}
	display := createdAtDisplay(item.CreatedAt)
	return CampaignEventTimelineItemDTO{
		TitleLabel:        title,
		ActorLabel:        actor,
		ChangeSummary:     summary,
		SectionID:         eventTypeSectionID(item.EventType),
		OccurredAtDisplay: display,
	}
}

func eventTypeTitleLabel(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "click":
		return "Click recorded"
	case "conversion":
		return "Conversion recorded"
	case "impression":
		return "Impression recorded"
	default:
		if eventType == "" {
			return "Event recorded"
		}
		return strings.ToUpper(eventType[:1]) + eventType[1:] + " recorded"
	}
}

func eventTypeSummary(item CampaignEventDTO) string {
	if len(item.Payload) == 0 {
		return item.EventType
	}
	var raw map[string]any
	if err := json.Unmarshal(item.Payload, &raw); err != nil {
		return item.EventType
	}
	if placement, ok := raw["placement_id"].(string); ok && placement != "" {
		return "placement " + placement
	}
	return item.EventType
}

func eventTypeSectionID(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "click", "impression":
		return "tracking"
	case "conversion":
		return "postbacks"
	default:
		return "integrations"
	}
}

func createdAtDisplay(createdAt string) string {
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(createdAt))
	if err != nil {
		return createdAt
	}
	return ts.UTC().Format("2006-01-02 15:04 UTC")
}

func maskActorLabel(actor string) string {
	if len(actor) <= 4 {
		return "[masked]"
	}
	return actor[:2] + "***" + actor[len(actor)-2:]
}

func sortTimelineDays(days []CampaignEventTimelineDayDTO) {
	for i := 0; i < len(days); i++ {
		for j := i + 1; j < len(days); j++ {
			if days[j].Day > days[i].Day {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
}
