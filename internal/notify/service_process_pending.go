package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"ad-event-processor/internal/notify/db"
)

type notificationGroup struct {
	key   groupKey
	items []db.NotifierNotification
}

type groupKey struct {
	provider     db.NotifierProvider
	recipient    string
	title        string
	deliveryMode db.NotifierDeliveryMode
}

func groupClaimedNotifications(notifications []db.NotifierNotification) (groups []notificationGroup, total int) {
	byKey := make(map[groupKey][]db.NotifierNotification)
	var orderedKeys []groupKey

	for i := range notifications {
		notification := &notifications[i]
		key := groupKey{
			provider:     notification.Provider,
			recipient:    notification.Recipient,
			title:        notification.Title.String,
			deliveryMode: notification.DeliveryMode,
		}
		if _, exists := byKey[key]; !exists {
			orderedKeys = append(orderedKeys, key)
		}
		byKey[key] = append(byKey[key], *notification)
	}

	groups = make([]notificationGroup, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		groups = append(groups, notificationGroup{key: key, items: byKey[key]})
	}
	return groups, len(notifications)
}

func (s *Service) processGroupsParallel(ctx context.Context, groups []notificationGroup) (int, error) {
	if len(groups) == 0 {
		return 0, nil
	}

	parallelism := s.options.groupParallelism()
	if parallelism > len(groups) {
		parallelism = len(groups)
	}

	sem := make(chan struct{}, parallelism)
	var (
		wg           sync.WaitGroup
		processed    int
		processedMu  sync.Mutex
		firstErr     error
		firstErrOnce sync.Once
	)

	for _, group := range groups {
		wg.Add(1)
		go func(group notificationGroup) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			count, err := s.finalizeGroup(ctx, group)
			processedMu.Lock()
			processed += count
			processedMu.Unlock()
			if err != nil {
				firstErrOnce.Do(func() { firstErr = err })
			}
		}(group)
	}

	wg.Wait()
	return processed, firstErr
}

func (s *Service) finalizeGroup(ctx context.Context, group notificationGroup) (int, error) {
	items := group.items
	lead := items[0]
	leadID := uuidString(lead.ID)
	finalBody := buildAggregatedBody(items)
	isAggregated := len(items) > 1
	if isAggregated {
		dedupAggregatedTotal.Add(float64(len(items) - 1))
	}

	var sendErr error
	var sentProvider db.NotifierProvider
	var deliveryNote string

	if lead.DeliveryMode == db.NotifierDeliveryModeBROADCAST {
		var targets []db.NotifierProvider
		if len(lead.BroadcastProviders) > 0 {
			targets = make([]db.NotifierProvider, 0, len(lead.BroadcastProviders))
			for _, provider := range lead.BroadcastProviders {
				targets = append(targets, db.NotifierProvider(provider))
			}
		}
		targets = s.resolveBroadcastTargets(targets)
		result := s.deliverBroadcast(ctx, leadID, lead.Provider, lead.Recipient, targets, lead.Title.String, finalBody)
		sendErr = result.err
		sentProvider = result.sentProvider
		deliveryNote = result.partialNote
	} else {
		sentProvider, sendErr = s.deliverFallback(ctx, leadID, lead.Provider, lead.Recipient, lead.Title.String, finalBody)
	}

	if sendErr == nil {
		return s.markGroupSent(ctx, lead, items, leadID, sentProvider, deliveryNote, isAggregated)
	}
	return s.markGroupFailed(ctx, lead, sendErr)
}

func (s *Service) markGroupSent(
	ctx context.Context,
	lead db.NotifierNotification,
	items []db.NotifierNotification,
	leadID string,
	sentProvider db.NotifierProvider,
	deliveryNote string,
	isAggregated bool,
) (int, error) {
	dbSentProvider := db.NullNotifierProvider{NotifierProvider: sentProvider, Valid: true}
	errorMessage := pgtypeTextOptional(deliveryNote)

	if _, err := s.queries.UpdateNotificationStatus(ctx, db.UpdateNotificationStatusParams{
		ID:           lead.ID,
		Status:       db.NotifierNotificationStatusSENT,
		Provider:     dbSentProvider,
		RetryCount:   pgtypeInt4(lead.RetryCount),
		ErrorMessage: errorMessage,
	}); err != nil {
		return 0, fmt.Errorf("update lead notification status to SENT: %w", err)
	}

	if isAggregated {
		for i := 1; i < len(items); i++ {
			subItem := items[i]
			if _, err := s.queries.UpdateNotificationStatus(ctx, db.UpdateNotificationStatusParams{
				ID:           subItem.ID,
				Status:       db.NotifierNotificationStatusSENT,
				Provider:     dbSentProvider,
				RetryCount:   pgtypeInt4(subItem.RetryCount),
				ErrorMessage: pgtypeText(fmt.Sprintf("Deduplicated and aggregated into %s", leadID)),
			}); err != nil {
				return 0, fmt.Errorf("update sub notification status to SENT: %w", err)
			}
		}
	}
	return len(items), nil
}

func (s *Service) markGroupFailed(ctx context.Context, lead db.NotifierNotification, sendErr error) (int, error) {
	nextRetryCount := lead.RetryCount + 1
	nextStatus := db.NotifierNotificationStatusPENDING
	if nextRetryCount >= maxDeliveryAttempts {
		nextStatus = db.NotifierNotificationStatusFAILED
		permanentFailuresTotal.Inc()
		slog.Error("notification delivery failed permanently", "error", sendErr, "notification_id", uuidString(lead.ID), "retries", nextRetryCount)
	} else {
		slog.Warn("notification delivery failed, will retry", "error", sendErr, "notification_id", uuidString(lead.ID), "retries", nextRetryCount)
	}

	if _, err := s.queries.UpdateNotificationStatus(ctx, db.UpdateNotificationStatusParams{
		ID:           lead.ID,
		Status:       nextStatus,
		RetryCount:   pgtypeInt4(nextRetryCount),
		ErrorMessage: pgtypeText(sendErr.Error()),
	}); err != nil {
		return 0, fmt.Errorf("update lead notification status after failure: %w", err)
	}
	return 1, nil
}

func (s *Service) ProcessPending(ctx context.Context, batchSize int32) (int, error) {
	if _, err := s.queries.ReclaimStaleProcessing(ctx, int64(s.options.claimStale().Seconds())); err != nil {
		return 0, fmt.Errorf("reclaim stale processing notifications: %w", err)
	}

	notifications, err := s.queries.ClaimPendingNotifications(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim pending notifications: %w", err)
	}
	if len(notifications) == 0 {
		return 0, nil
	}

	groups, _ := groupClaimedNotifications(notifications)
	return s.processGroupsParallel(ctx, groups)
}

func (s *Service) ProcessPendingSequential(ctx context.Context, batchSize int32) (int, error) {
	old := s.options.GroupParallelism
	s.options.GroupParallelism = 1
	defer func() { s.options.GroupParallelism = old }()
	return s.ProcessPending(ctx, batchSize)
}
