package stream

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream/auditlog"
	"ad-event-processor/internal/stream/breaker"
	"ad-event-processor/internal/stream/codec"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

// StreamConsumer drains Redis stream consumer groups into domain.EventStore (ClickHouse
// spool or direct). Invariant: XAck only after StoreBatch succeeds; PEL retains messages
// on retriable store errors. Poison rows split via splitStoreBatch then DLQ.
//
// Verify:
// go test ./internal/stream/ -short -run TestStreamConsumer -count=1
type StreamConsumer struct {
	store              domain.EventStore
	redisClient        redis.UniversalClient
	streamName         string
	groupName          string
	consumerID         string
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	startMu            sync.Mutex
	flushInt           time.Duration
	writeTimeout       time.Duration
	retryInitWait      time.Duration
	retryMaxWait       time.Duration
	streamMinIdle      time.Duration
	drainTimeout       time.Duration
	batchSize          int
	maxWorkers         int
	maxRetries         int
	started            bool
	cb                 *breaker.CircuitBreaker
	logger             *logger.Logger
	auditLogSeq        atomic.Uint64
	auditLogSampleMask uint64
	dlqStreamName      string
	onMessageProcessed func(evt *domain.Event, msgID string)
	weightCtrl         *ProcessorWeightController
}

func (c *StreamConsumer) SetWeightController(w *ProcessorWeightController) {
	c.weightCtrl = w
}

func (c *StreamConsumer) SetOnMessageProcessed(cb func(evt *domain.Event, msgID string)) {
	c.onMessageProcessed = cb
}

func (c *StreamConsumer) SetLogger(l *logger.Logger) {
	c.logger = l
}

func (c *StreamConsumer) SetAuditLogSampleMask(mask int) {
	c.auditLogSampleMask = AuditLogSampleMaskFromConfig(mask)
}

func (c *StreamConsumer) SetDLQStream(name string) {
	c.dlqStreamName = name
}

func (c *StreamConsumer) dlqStream() string {
	if c.dlqStreamName != "" {
		return c.dlqStreamName
	}
	const suffix = ":stream"
	if strings.HasSuffix(c.streamName, suffix) {
		return c.streamName[:len(c.streamName)-len(suffix)] + ":dlq"
	}
	return "ad:events:dlq"
}

func (c *StreamConsumer) CircuitBreakerState() CircuitState {
	if c == nil || c.cb == nil {
		return CircuitClosed
	}
	return c.cb.State()
}

func NewStreamConsumer(
	store domain.EventStore,
	redisClient redis.UniversalClient,
	streamName, groupName, consumerID string,
	batchSize int,
	maxWorkers int,
	flushInt, writeTimeout time.Duration,
	retryInitWait, retryMaxWait time.Duration,
	maxRetries int,
	streamMinIdle time.Duration,
	drainTimeout time.Duration,
) *StreamConsumer {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	uniqueConsumerID := fmt.Sprintf("%s-%s-%s", consumerID, hostname, uuid.NewString()[:8])

	return &StreamConsumer{
		store:              store,
		redisClient:        redisClient,
		streamName:         streamName,
		groupName:          groupName,
		consumerID:         uniqueConsumerID,
		batchSize:          batchSize,
		flushInt:           flushInt,
		writeTimeout:       writeTimeout,
		maxWorkers:         maxWorkers,
		retryInitWait:      retryInitWait,
		retryMaxWait:       retryMaxWait,
		maxRetries:         maxRetries,
		streamMinIdle:      streamMinIdle,
		drainTimeout:       drainTimeout,
		cb:                 breaker.NewCircuitBreaker(maxRetries, retryMaxWait*2),
		auditLogSampleMask: auditlog.SampleMaskDefault,
	}
}

// Start creates the consumer group, maxWorkers XReadGroup loops, janitor (XAutoClaim),
// and DLQ depth monitor. Each worker gets a unique consumer ID suffix (-wN).
func (c *StreamConsumer) Start(ctx context.Context) {
	c.startMu.Lock()
	defer c.startMu.Unlock()
	if c.started {
		return
	}
	c.started = true

	procCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	err := c.redisClient.XGroupCreateMkStream(ctx, c.streamName, c.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		slog.Error("failed to create consumer group", "error", err, "stream", c.streamName, "group", c.groupName)
	}

	for i := range c.maxWorkers {
		c.wg.Add(1)
		go func(workerIdx int) {
			defer c.wg.Done()
			c.worker(procCtx, workerIdx)
		}(i)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.janitor(procCtx)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.dlqMonitor(procCtx)
	}()
}

func (c *StreamConsumer) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *StreamConsumer) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *StreamConsumer) workerConsumerID(workerIdx int) string {
	return fmt.Sprintf("%s-w%d", c.consumerID, workerIdx)
}

// worker: micro-batch XReadGroup (">"), optional ProcessorWeightController throttle,
// flush on batchSize or flushInt. Shutdown drains in-flight batch and pending ">" reads
// before returning (drainTimeout bounds wait).
func (c *StreamConsumer) worker(ctx context.Context, workerIdx int) {
	workerID := c.workerConsumerID(workerIdx)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("worker panic recovered - exiting process", "error", r, "worker", workerID)
			os.Exit(1)
		}
	}()

	initCtx, initCancel := context.WithTimeout(ctx, c.writeTimeout*2)
	c.recoverPending(initCtx, workerID)
	initCancel()

	batch := make([]*domain.Event, 0, c.batchSize)
	msgIDs := make([]string, 0, c.batchSize)

	retryWait := c.retryInitWait
	retryCount := 0
	lastFlush := time.Now()

	xreadArgs := &redis.XReadGroupArgs{
		Group:    c.groupName,
		Consumer: workerID,
		Streams:  []string{c.streamName, ">"},
	}

	for {
		if c.pauseStreamReads(ctx) {
			continue
		}

		select {
		case <-ctx.Done():
			drainCtx, drainCancel := context.WithTimeout(context.WithoutCancel(ctx), c.drainTimeout)
			if len(batch) > 0 {
				if err := c.flushBatch(drainCtx, batch, msgIDs, workerID); err == nil {
					for _, e := range batch {
						domain.EventPool.Put(e)
					}
				} else if !isRetriableStoreError(err) {
					slog.Error("drain flush of existing batch failed, GC will reclaim objects", "error", err, "group", c.groupName, "worker", workerID)
				} else {
					slog.Warn("drain flush deferred, retaining batch in PEL", "error", err, "group", c.groupName, "worker", workerID)
				}
				batch = batch[:0]
				msgIDs = msgIDs[:0]
			}

			c.drainNewMessages(drainCtx, workerID)
			c.recoverPending(drainCtx, workerID)
			drainCancel()
			return
		default:
		}

		if c.weightCtrl != nil {
			c.weightCtrl.ThrottleBeforeRead(ctx)
		}

		readCount := int64(c.batchSize - len(batch))
		if readCount <= 0 {
			c.tryFlush(ctx, &batch, &msgIDs, &retryCount, workerID, nil, &retryWait)
			lastFlush = time.Now()
			continue
		}
		if c.weightCtrl != nil {
			readCount = c.weightCtrl.EffectiveReadCount(int(readCount))
		}

		var blockTime time.Duration
		if len(batch) == 0 {
			blockTime = 200 * time.Millisecond
		} else {
			elapsed := time.Since(lastFlush)
			if elapsed >= c.flushInt {
				c.tryFlush(ctx, &batch, &msgIDs, &retryCount, workerID, nil, &retryWait)
				lastFlush = time.Now()
				continue
			}
			blockTime = c.flushInt - elapsed
			if blockTime > 200*time.Millisecond {
				blockTime = 200 * time.Millisecond
			}
		}

		xreadArgs.Count = readCount
		xreadArgs.Block = blockTime
		streams, err := c.redisClient.XReadGroup(ctx, xreadArgs).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				if len(batch) > 0 && time.Since(lastFlush) >= c.flushInt {
					c.tryFlush(ctx, &batch, &msgIDs, &retryCount, workerID, nil, &retryWait)
					lastFlush = time.Now()
				}
			} else {
				slog.Error("failed to read from redis stream", "error", err)
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
				}
			}
			continue
		}

		hadEmptyBatch := len(batch) == 0

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				evt := c.parseMessage(msg.ID, msg.Values)
				batch = append(batch, evt)
				msgIDs = append(msgIDs, msg.ID)
				if c.onMessageProcessed != nil {
					c.onMessageProcessed(evt, msg.ID)
				}
			}
		}

		if hadEmptyBatch && len(batch) > 0 {
			lastFlush = time.Now()
		}

		if len(batch) >= c.batchSize || time.Since(lastFlush) >= c.flushInt {
			c.tryFlush(ctx, &batch, &msgIDs, &retryCount, workerID, nil, &retryWait)
			lastFlush = time.Now()
		}
	}
}

// pauseStreamReads blocks XReadGroup while the store circuit breaker is open so workers
// do not grow the PEL during ClickHouse outages (ad_processor_stream_backpressure_active).
func (c *StreamConsumer) pauseStreamReads(ctx context.Context) bool {
	if c.cb.State() != CircuitOpen {
		metrics.ProcessorStreamBackpressureActive.WithLabelValues(c.groupName).Set(0)
		return false
	}

	wait := c.cb.WaitDuration()
	if wait <= 0 {

		metrics.ProcessorStreamBackpressureActive.WithLabelValues(c.groupName).Set(0)
		return false
	}

	metrics.ProcessorStreamBackpressureActive.WithLabelValues(c.groupName).Set(1)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(wait):
		return true
	}
}

func (c *StreamConsumer) recordSuccess(workerID string) {
	c.cb.RecordSuccess(workerID)
	metrics.CircuitBreakerState.WithLabelValues(c.groupName).Set(float64(c.cb.State()))
}

func (c *StreamConsumer) recordFailure(workerID string) {
	c.cb.RecordFailure(workerID)
	metrics.CircuitBreakerState.WithLabelValues(c.groupName).Set(float64(c.cb.State()))
}

func (c *StreamConsumer) recordCancellation(workerID string) {
	c.cb.RecordCancellation(workerID)
	metrics.CircuitBreakerState.WithLabelValues(c.groupName).Set(float64(c.cb.State()))
}

// tryFlush runs StoreBatch+XAck via flushBatch. On failure increments ad:events:retries
// per msg ID; after maxRetries non-retriable errors decomposes batch (splitStoreBatch)
// and moves poison rows to DLQ stream (XAck+XDel on success).
func (c *StreamConsumer) tryFlush(ctx context.Context, batch *[]*domain.Event, msgIDs *[]string, retryCount *int, workerID string, ticker *time.Ticker, retryWait *time.Duration) {
	if !c.cb.Allow() {
		wait := c.cb.WaitDuration()
		if wait <= 0 {
			wait = 100 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		return
	}
	err := c.flushBatch(ctx, *batch, *msgIDs, workerID)
	if err == nil {
		c.recordSuccess(workerID)
		_ = c.redisClient.HDel(ctx, "ad:events:retries", (*msgIDs)...).Err()
		for _, e := range *batch {
			domain.EventPool.Put(e)
		}
		*batch = (*batch)[:0]
		*msgIDs = (*msgIDs)[:0]
		if ticker != nil {
			ticker.Reset(c.flushInt)
		}
		*retryWait = 100 * time.Millisecond
		*retryCount = 0
		return
	}

	if errors.Is(err, context.Canceled) {
		c.recordCancellation(workerID)
		return
	}

	*retryCount++
	c.recordFailure(workerID)

	pipe := c.redisClient.Pipeline()
	incrCmds := make([]*redis.IntCmd, len(*msgIDs))
	for i, id := range *msgIDs {
		incrCmds[i] = pipe.HIncrBy(ctx, "ad:events:retries", id, 1)
	}
	_, _ = pipe.Exec(ctx)

	hasPoisonPill := false
	maxIncr := int64(0)
	for i := range *msgIDs {
		cVal, _ := incrCmds[i].Result()
		if cVal > maxIncr {
			maxIncr = cVal
		}
		if cVal > int64(c.maxRetries) {
			hasPoisonPill = true
		}
	}

	if maxIncr > int64(*retryCount) {
		*retryCount = int(maxIncr)
	}

	if hasPoisonPill {
		if isRetriableStoreError(err) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(*retryWait):
			}
			*retryWait *= 2
			if *retryWait > c.retryMaxWait {
				*retryWait = c.retryMaxWait
			}
			return
		}

		slog.Error("poison pill detected, decomposing batch", "error", err, "group", c.groupName, "worker", workerID)

		successIdx, failedIndices := c.splitStoreBatch(ctx, *batch, *msgIDs, 0)

		successfulMsgIDs := make([]string, 0, len(successIdx))
		for _, i := range successIdx {
			successfulMsgIDs = append(successfulMsgIDs, (*msgIDs)[i])
		}

		if len(successfulMsgIDs) > 0 {
			ackCtx, ackCancel := context.WithTimeout(ctx, c.writeTimeout)
			_ = c.redisClient.XAck(ackCtx, c.streamName, c.groupName, successfulMsgIDs...).Err()
			_ = c.redisClient.HDel(ackCtx, "ad:events:retries", successfulMsgIDs...).Err()
			ackCancel()
		}

		if len(failedIndices) > 0 {
			failedBatch := make([]*domain.Event, 0, len(failedIndices))
			failedMsgIDs := make([]string, 0, len(failedIndices))
			for _, i := range failedIndices {
				failedBatch = append(failedBatch, (*batch)[i])
				failedMsgIDs = append(failedMsgIDs, (*msgIDs)[i])
			}

			execErr := c.moveToDLQ(ctx, failedBatch, failedMsgIDs, workerID, *retryCount, fmt.Errorf("batch decomposed: %w", err))

			if execErr != nil {
				slog.Error("failed to exec dlq pipeline, retaining in PEL", "error", execErr, "group", c.groupName)
				newBatch := (*batch)[:0]
				newMsgIDs := (*msgIDs)[:0]
				for _, i := range failedIndices {
					newBatch = append(newBatch, (*batch)[i])
					newMsgIDs = append(newMsgIDs, (*msgIDs)[i])
				}
				fiIdx := 0
				for i, e := range *batch {
					if fiIdx < len(failedIndices) && i == failedIndices[fiIdx] {
						fiIdx++
					} else {
						domain.EventPool.Put(e)
					}
				}
				*batch = newBatch
				*msgIDs = newMsgIDs
				return
			}
		}

		for _, e := range *batch {
			domain.EventPool.Put(e)
		}
		*batch = (*batch)[:0]
		*msgIDs = (*msgIDs)[:0]
		if ticker != nil {
			ticker.Reset(c.flushInt)
		}
		*retryWait = 100 * time.Millisecond
		*retryCount = 0
	} else {
		select {
		case <-ctx.Done():
			return
		case <-time.After(*retryWait):
		}
		*retryWait *= 2
		if *retryWait > c.retryMaxWait {
			*retryWait = c.retryMaxWait
		}
	}
}

var (
	dlqEventPool = sync.Pool{
		New: func() any {
			return new(pb.AdDLQEvent)
		},
	}
	dlqValuesPool = sync.Pool{
		New: func() any {
			slice := make([]any, 2)
			slice[0] = "d"
			return &slice
		},
	}
)

// moveToDLQ marshals AdDLQEvent protobuf into dlq stream, then XAck+XDel source rows per
// successful XAdd. Partial pipeline failure retains failed rows in PEL (no silent drop).
func (c *StreamConsumer) moveToDLQ(ctx context.Context, batch []*domain.Event, msgIDs []string, workerID string, retryCount int, err error) error {
	errStr := err.Error()

	pipeWrite := c.redisClient.Pipeline()

	writtenMsgIDs := make([]string, 0, len(batch))
	valuesPtrs := make([]*[]any, 0, len(batch))
	bufPtrs := make([]*[]byte, 0, len(batch))
	wrapPtrs := make([]*ByteSliceValue, 0, len(batch))
	defer func() {
		for _, ptr := range valuesPtrs {
			dlqValuesPool.Put(ptr)
		}
		for _, ptr := range bufPtrs {
			codec.ByteBufPool.Put(ptr)
		}
		for _, ptr := range wrapPtrs {
			codec.ByteSliceValuePool.Put(ptr)
		}
	}()

	execCtx, execCancel := context.WithTimeout(ctx, c.writeTimeout)
	defer execCancel()

	for i, e := range batch {
		pbDLQ := dlqEventPool.Get().(*pb.AdDLQEvent)
		if pbDLQ.OriginalEvent == nil {
			pbDLQ.OriginalEvent = new(pb.AdStreamEvent)
		} else {
			DeepResetAdStreamEvent(pbDLQ.OriginalEvent)
		}
		pbDLQ.Error = append(pbDLQ.Error[:0], errStr...)
		pbDLQ.OriginalId = append(pbDLQ.OriginalId[:0], msgIDs[i]...)
		pbDLQ.FailedAtUnix = time.Now().Unix()
		pbDLQ.WorkerId = append(pbDLQ.WorkerId[:0], workerID...)
		pbDLQ.RetryCount = int32(retryCount)

		pbDLQ.OriginalEvent.ClickId = append(pbDLQ.OriginalEvent.ClickId[:0], e.ClickID...)
		pbDLQ.OriginalEvent.CampaignId = append(pbDLQ.OriginalEvent.CampaignId[:0], e.CampaignID[:]...)
		pbDLQ.OriginalEvent.EventType = append(pbDLQ.OriginalEvent.EventType[:0], e.Type...)
		pbDLQ.OriginalEvent.Payload = append(pbDLQ.OriginalEvent.Payload[:0], e.Payload...)
		pbDLQ.OriginalEvent.Ip = append(pbDLQ.OriginalEvent.Ip[:0], e.IP...)
		pbDLQ.OriginalEvent.Ua = append(pbDLQ.OriginalEvent.Ua[:0], e.UA...)
		pbDLQ.OriginalEvent.CreatedAtUnix = e.CreatedAt.Unix()

		size := pbDLQ.SizeVT()
		bufPtr := codec.ByteBufPool.Get().(*[]byte)
		buf := *bufPtr
		if cap(buf) < size {
			buf = make([]byte, size)
		} else {
			buf = buf[:size]
		}

		n, marshalErr := pbDLQ.MarshalToSizedBufferVT(buf)
		if marshalErr != nil {
			slog.Error("failed to marshal DLQ event", "error", marshalErr)
			DeepResetAdDLQEvent(pbDLQ)
			dlqEventPool.Put(pbDLQ)
			*bufPtr = buf
			codec.ByteBufPool.Put(bufPtr)
			continue
		}

		data := buf[:n]
		*bufPtr = buf
		bufPtrs = append(bufPtrs, bufPtr)

		DeepResetAdDLQEvent(pbDLQ)
		dlqEventPool.Put(pbDLQ)
		writtenMsgIDs = append(writtenMsgIDs, msgIDs[i])

		valuesPtr := dlqValuesPool.Get().(*[]any)
		values := *valuesPtr

		wrap := codec.ByteSliceValuePool.Get().(*ByteSliceValue)
		wrap.B = data
		values[1] = wrap
		wrapPtrs = append(wrapPtrs, wrap)

		valuesPtrs = append(valuesPtrs, valuesPtr)

		pipeWrite.XAdd(execCtx, &redis.XAddArgs{
			Stream: c.dlqStream(),
			MaxLen: 100000,
			Approx: true,
			Values: values,
		})
	}

	if len(writtenMsgIDs) == 0 {
		return nil
	}

	cmders, execErr := pipeWrite.Exec(execCtx)

	var hasError bool
	if execErr != nil && !errors.Is(execErr, redis.Nil) {
		slog.Error("DLQ write pipeline returned error", "error", execErr)
		hasError = true
	}

	pipeAck := c.redisClient.Pipeline()
	ackedMsgIDs := make([]string, 0, len(batch))

	ackCtx, ackCancel := context.WithTimeout(ctx, c.writeTimeout)
	defer ackCancel()

	for i, cmder := range cmders {
		if cmder.Err() == nil {
			msgID := writtenMsgIDs[i]
			pipeAck.XAck(ackCtx, c.streamName, c.groupName, msgID)
			pipeAck.XDel(ackCtx, c.streamName, msgID)
			ackedMsgIDs = append(ackedMsgIDs, msgID)
		} else {
			slog.Error("individual DLQ write failed", "error", cmder.Err(), "msgID", writtenMsgIDs[i])
			hasError = true
		}
	}

	if len(ackedMsgIDs) > 0 {
		_, ackErr := pipeAck.Exec(ackCtx)
		if ackErr != nil {
			slog.Error("DLQ ack/del pipeline failed", "error", ackErr)
			return ackErr
		}
	}

	if hasError || len(ackedMsgIDs) < len(writtenMsgIDs) {
		return fmt.Errorf("DLQ write partial failure: wrote %d of %d messages", len(ackedMsgIDs), len(writtenMsgIDs))
	}

	return nil
}

func (c *StreamConsumer) ParseMessage(id string, values map[string]interface{}) *domain.Event {
	return c.parseMessage(id, values)
}

func (c *StreamConsumer) FlushBatch(ctx context.Context, batch []*domain.Event, msgIDs []string, workerID string) error {
	return c.flushBatch(ctx, batch, msgIDs, workerID)
}

func (c *StreamConsumer) StartMaintenance(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.janitor(ctx)
	}()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.dlqMonitor(ctx)
	}()
}

// parseMessage decodes stream field "d" as AdStreamEvent protobuf (tracker hot path).
// Strings alias into event.StringBuffer (zero extra alloc). Legacy flat hash fields and
// fraud_aggregate_event_type remain for older stream rows.
func (c *StreamConsumer) parseMessage(id string, values map[string]interface{}) *domain.Event {
	event := domain.EventPool.Get().(*domain.Event)
	event.Reset()

	if raw, ok := streamFieldBytes(values, "d"); ok {
		pbEvt := codec.StreamEventPool.Get().(*pb.AdStreamEvent)
		DeepResetAdStreamEvent(pbEvt)

		if len(raw) > 0 {
			_ = raw[len(raw)-1]
		}

		if err := pbEvt.UnmarshalVT(raw); err == nil {
			totalLen := len(pbEvt.ClickId) + len(pbEvt.EventType) + len(pbEvt.Ip) + len(pbEvt.Ua)
			if cap(event.StringBuffer) < totalLen {
				event.StringBuffer = make([]byte, 0, totalLen+128)
			} else {
				event.StringBuffer = event.StringBuffer[:0]
			}

			event.StringBuffer = append(event.StringBuffer, pbEvt.ClickId...)
			event.ClickID = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.ClickId):])

			event.StringBuffer = append(event.StringBuffer, pbEvt.EventType...)
			event.Type = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.EventType):])

			event.StringBuffer = append(event.StringBuffer, pbEvt.Ip...)
			event.IP = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.Ip):])

			event.StringBuffer = append(event.StringBuffer, pbEvt.Ua...)
			event.UA = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.Ua):])

			_ = ParseUUID(pbEvt.CampaignId, &event.CampaignID)
			event.Payload = append(event.Payload[:0], pbEvt.Payload...)
			if len(pbEvt.FraudReason) > 0 {
				event.StringBuffer = append(event.StringBuffer, pbEvt.FraudReason...)
				event.FraudReason = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.FraudReason):])
			}
			event.FraudScore = pbEvt.FraudScore
			event.LayerDesyncCount = uint8(pbEvt.LayerDesyncCount)
			event.SilentRejectEvent = pbEvt.SilentRejectEvent
			event.ReviewRoutedEvent = pbEvt.ReviewRoutedEvent
			if len(pbEvt.UserId) > 0 {
				event.StringBuffer = append(event.StringBuffer, pbEvt.UserId...)
				event.UserID = unsafeString(event.StringBuffer[len(event.StringBuffer)-len(pbEvt.UserId):])
			}
			if pbEvt.CreatedAtUnix > 0 {
				event.CreatedAt = time.Unix(pbEvt.CreatedAtUnix, 0)
			}
			if isFraudStreamLayerDesyncTelemetry(event) {
				observeFraudStreamLayerDesync(event.LayerDesyncCount)
			}
		} else {
			slog.Error("failed to unmarshal stream event protobuf", "error", err)
		}
		DeepResetAdStreamEvent(pbEvt)
		codec.StreamEventPool.Put(pbEvt)
	} else if v, ok := values["type"].(string); ok && v == fraudAggregateEventType {
		event.Type = fraudAggregateEventType
		if v, ok := values["subnet"].(string); ok {
			event.IP = v
		}
		if v, ok := values["ipv6_prefix"].(string); ok {
			event.PlacementID = v
		}
		if v, ok := values["fraud_reason"].(string); ok {
			event.FraudReason = v
		}
		if v, ok := values["count"].(string); ok {
			event.ClickID = v
		}
		if v, ok := values["window_ms"].(string); ok {
			event.UserID = v
		}
	} else if _, ok := values["click_id"]; ok {
		parseFlatStreamMessage(event, values)
	}

	if event.CreatedAt.IsZero() {
		if idx := strings.IndexByte(id, '-'); idx > 0 {
			ms, err := strconv.ParseInt(id[:idx], 10, 64)
			if err == nil {
				event.CreatedAt = time.Unix(0, ms*int64(time.Millisecond))
			}
		}
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	return event
}

func streamFieldBytes(values map[string]interface{}, key string) ([]byte, bool) {
	v, ok := values[key]
	if !ok {
		return nil, false
	}
	switch b := v.(type) {
	case string:
		return UnsafeBytes(b), true
	case []byte:
		return b, true
	default:
		return nil, false
	}
}

func parseFlatStreamMessage(event *domain.Event, values map[string]interface{}) {
	if v, ok := values["click_id"].(string); ok {
		event.ClickID = v
	}
	if v, ok := values["campaign_id"].(string); ok {
		_ = ParseUUID(UnsafeBytes(v), &event.CampaignID)
	}
	if v, ok := values["user_id"].(string); ok {
		event.UserID = v
	}
	if v, ok := values["type"].(string); ok {
		event.Type = v
	}
	if raw, ok := streamFieldBytes(values, "payload"); ok {
		if len(raw) > 0 {
			_ = raw[len(raw)-1]
		}
		event.Payload = append(event.Payload[:0], raw...)
	}
	if v, ok := values["ip"].(string); ok {
		event.IP = v
	}
	if v, ok := values["ua"].(string); ok {
		event.UA = v
	}
}

func firstN(ids []string, n int) []string {
	if len(ids) <= n {
		return ids
	}
	return ids[:n]
}

// flushBatch: StoreBatch then XAck. domain.DeduplicationTokenKey scopes CH insert idempotency
// to first+last Redis msg ID in the batch. Never ACK on store error (message stays in PEL).
func (c *StreamConsumer) flushBatch(ctx context.Context, batch []*domain.Event, msgIDs []string, workerID string) error {
	if len(batch) == 0 {
		return nil
	}

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.Debug("flushing batch", "group", c.groupName, "batch_size", len(batch), "first_ids", firstN(msgIDs, 5))
	}

	storeCtx, storeCancel := context.WithTimeout(ctx, c.writeTimeout)
	if len(msgIDs) > 0 {
		token := fmt.Sprintf("%s_%s_%d", msgIDs[0], msgIDs[len(msgIDs)-1], len(msgIDs))
		storeCtx = context.WithValue(storeCtx, domain.DeduplicationTokenKey, token)
	}
	defer storeCancel()

	err := c.store.StoreBatch(storeCtx, batch)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("store failed, NOT ACKING", "error", err, "group", c.groupName, "batch_size", len(batch), "first_ids", firstN(msgIDs, 5))
		}
		return err
	}

	if len(batch) > 0 && !batch[0].CreatedAt.IsZero() {
		lagSec := time.Since(batch[0].CreatedAt).Seconds()
		instance := "local"
		if c.weightCtrl != nil {
			instance = c.weightCtrl.InstanceLabel()
		}
		metrics.ProcessorStreamLagSeconds.WithLabelValues(instance).Set(lagSec)
		SetProcessorStreamLagSec(int64(lagSec))
	}

	if c.logger != nil {
		workerIdx := 0
		if idx := strings.LastIndex(workerID, "-w"); idx != -1 {
			if val, err := strconv.Atoi(workerID[idx+2:]); err == nil {
				workerIdx = val
			}
		}
		for _, e := range batch {
			WriteAuditLog(c.logger, &c.auditLogSeq, c.auditLogSampleMask, workerIdx, e)
		}
	}

	ackCtx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()
	if err := c.redisClient.XAck(ackCtx, c.streamName, c.groupName, msgIDs...).Err(); err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("xack failed after successful store", "error", err, "group", c.groupName, "batch_size", len(batch), "first_ids", firstN(msgIDs, 5))
		}
		return err
	}
	return nil
}

// recoverPending replays this worker's PEL (stream "0") on startup before reading ">".
// Retriable store errors leave messages in PEL; hard failures go to DLQ.
func (c *StreamConsumer) recoverPending(ctx context.Context, consumerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			entries, err := c.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.groupName,
				Consumer: consumerID,
				Streams:  []string{c.streamName, "0"},
				Count:    int64(c.batchSize),
			}).Result()

			if err != nil || len(entries) == 0 || len(entries[0].Messages) == 0 {
				return
			}

			batch := make([]*domain.Event, 0, len(entries[0].Messages))
			msgIDs := make([]string, 0, len(entries[0].Messages))

			for _, msg := range entries[0].Messages {
				batch = append(batch, c.parseMessage(msg.ID, msg.Values))
				msgIDs = append(msgIDs, msg.ID)
			}

			if err := c.flushBatch(ctx, batch, msgIDs, consumerID); err != nil {
				if !errors.Is(err, context.Canceled) {
					c.recordFailure(consumerID)
					if isRetriableStoreError(err) {
						slog.Warn("recovery flush deferred, retaining PEL", "error", err, "group", c.groupName)
						for _, e := range batch {
							domain.EventPool.Put(e)
						}
						return
					}
					slog.Error("recovery flush failed, moving to DLQ", "error", err, "group", c.groupName)
					_ = c.moveToDLQ(ctx, batch, msgIDs, consumerID, 1, fmt.Errorf("recovery flush failed: %w", err))
					_ = c.redisClient.HDel(ctx, "ad:events:retries", msgIDs...).Err()
				}
				for _, e := range batch {
					domain.EventPool.Put(e)
				}
				return
			}
			c.recordSuccess(consumerID)
			_ = c.redisClient.HDel(ctx, "ad:events:retries", msgIDs...).Err()
			for _, e := range batch {
				domain.EventPool.Put(e)
			}
		}
	}
}

func (c *StreamConsumer) drainNewMessages(ctx context.Context, consumerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			streams, err := c.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.groupName,
				Consumer: consumerID,
				Streams:  []string{c.streamName, ">"},
				Count:    int64(c.batchSize),
				Block:    50 * time.Millisecond,
			}).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, redis.ErrClosed) || strings.Contains(err.Error(), "client is closed") {
					return
				}
				slog.Error("drain: failed to read from stream", "error", err, "group", c.groupName, "worker", consumerID)
				return
			}

			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				return
			}

			batch := make([]*domain.Event, 0, len(streams[0].Messages))
			msgIDs := make([]string, 0, len(streams[0].Messages))

			for _, msg := range streams[0].Messages {
				batch = append(batch, c.parseMessage(msg.ID, msg.Values))
				msgIDs = append(msgIDs, msg.ID)
			}

			if err := c.flushBatch(ctx, batch, msgIDs, consumerID); err != nil {
				if !errors.Is(err, context.Canceled) {
					if isRetriableStoreError(err) {
						slog.Warn("drain: flush deferred, retaining PEL", "error", err, "group", c.groupName, "worker", consumerID)
					} else {
						slog.Error("drain: failed to flush batch", "error", err, "group", c.groupName, "worker", consumerID)
					}
				}
				for _, e := range batch {
					domain.EventPool.Put(e)
				}
				return
			}

			for _, e := range batch {
				domain.EventPool.Put(e)
			}
		}
	}
}

// janitor periodically XAutoClaim messages idle longer than streamMinIdle from dead consumers.
func (c *StreamConsumer) janitor(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("janitor panic recovered - exiting process", "error", r)
			os.Exit(1)
		}
	}()
	ticker := time.NewTicker(c.streamMinIdle)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.claimStuckMessages(ctx)
		}
	}
}

// claimStuckMessages: autoclaim + retry counter in ad:events:retries; exceeds maxRetries -> DLQ.
func (c *StreamConsumer) claimStuckMessages(ctx context.Context) {
	startID := "0-0"
	for {
		entries, nextID, err := c.redisClient.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   c.streamName,
			Group:    c.groupName,
			Consumer: c.consumerID,
			MinIdle:  c.streamMinIdle,
			Start:    startID,
			Count:    int64(c.batchSize),
		}).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
				slog.Error("autoclaim failed", "error", err, "group", c.groupName)
			}
			return
		}

		if len(entries) > 0 {
			pipe := c.redisClient.Pipeline()
			incrCmds := make([]*redis.IntCmd, len(entries))
			for i, msg := range entries {
				incrCmds[i] = pipe.HIncrBy(ctx, "ad:events:retries", msg.ID, 1)
			}
			_, _ = pipe.Exec(ctx)

			batch := make([]*domain.Event, 0, len(entries))
			msgIDs := make([]string, 0, len(entries))
			var dlqBatch []*domain.Event
			var dlqMsgIDs []string
			var delMsgIDs []string

			for i, msg := range entries {
				event := c.parseMessage(msg.ID, msg.Values)
				count, _ := incrCmds[i].Result()
				if count > int64(c.maxRetries) {
					dlqBatch = append(dlqBatch, event)
					dlqMsgIDs = append(dlqMsgIDs, msg.ID)
					delMsgIDs = append(delMsgIDs, msg.ID)
				} else {
					batch = append(batch, event)
					msgIDs = append(msgIDs, msg.ID)
				}
			}

			if len(dlqBatch) > 0 {
				slog.Error("autoclaim retry limit exceeded, moving to DLQ", "group", c.groupName, "count", len(dlqBatch))
				_ = c.moveToDLQ(ctx, dlqBatch, dlqMsgIDs, "janitor", c.maxRetries+1, errors.New("autoclaim delivery limit exceeded"))
				for _, e := range dlqBatch {
					domain.EventPool.Put(e)
				}
				if len(delMsgIDs) > 0 {
					_ = c.redisClient.HDel(ctx, "ad:events:retries", delMsgIDs...).Err()
				}
			}

			if len(batch) > 0 {
				if err := c.flushBatch(ctx, batch, msgIDs, "janitor"); err != nil {
					c.recordFailure("janitor")
					if !errors.Is(err, context.Canceled) {
						slog.Error("janitor flush failed", "error", err, "group", c.groupName)
					}
				} else {
					c.recordSuccess("janitor")
					_ = c.redisClient.HDel(ctx, "ad:events:retries", msgIDs...).Err()
				}
				for _, e := range batch {
					domain.EventPool.Put(e)
				}
			}
		}

		if nextID == "0-0" {
			break
		}
		startID = nextID
	}
}

func (c *StreamConsumer) dlqMonitor(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("dlq monitor panic recovered - exiting process", "error", r)
			os.Exit(1)
		}
	}()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			size, err := c.redisClient.XLen(ctx, c.dlqStream()).Result()
			if err != nil {
				if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
					slog.Error("failed to get DLQ size", "error", err)
				}
				continue
			}
			metrics.DlqSize.Set(float64(size))
		}
	}
}
