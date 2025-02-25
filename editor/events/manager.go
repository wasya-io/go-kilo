package events

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// EventSubscriber はイベントのサブスクライバーを表す型
type EventSubscriber func(Event)

// UpdateQueue はイベントの更新キューを管理する構造体
type UpdateQueue struct {
	queue []Event
	mu    sync.Mutex
}

// NewUpdateQueue は新しいUpdateQueueを作成する
func NewUpdateQueue() *UpdateQueue {
	return &UpdateQueue{
		queue: make([]Event, 0),
	}
}

// Add はキューにイベントを追加する
func (q *UpdateQueue) Add(event Event) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = append(q.queue, event)
}

// Flush はキューを処理して空にする
func (q *UpdateQueue) Flush(handler func(Event)) {
	q.mu.Lock()
	pendingEvents := q.queue
	q.queue = make([]Event, 0)
	q.mu.Unlock()

	for _, event := range pendingEvents {
		handler(event)
	}
}

// RecoverySnapshot はイベントマネージャーの状態スナップショットを表す
type RecoverySnapshot struct {
	Timestamp time.Time
	Events    []Event
	States    map[string]interface{}
}

// ErrorSeverity はエラーの重大度を表す
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// ErrorLog はエラーログエントリを表す
type ErrorLog struct {
	Timestamp time.Time
	Severity  ErrorSeverity
	EventType EventType
	Message   string
	Error     error
}

// EventMonitor はイベントの監視とエラー追跡を行う
type EventMonitor struct {
	errorLogs []ErrorLog
	mu        sync.RWMutex
	maxLogs   int
}

// NewEventMonitor は新しいEventMonitorを作成する
func NewEventMonitor(maxLogs int) *EventMonitor {
	return &EventMonitor{
		errorLogs: make([]ErrorLog, 0),
		maxLogs:   maxLogs,
	}
}

// LogError はエラーを記録する
func (em *EventMonitor) LogError(severity ErrorSeverity, eventType EventType, err error, msg string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	log := ErrorLog{
		Timestamp: time.Now(),
		Severity:  severity,
		EventType: eventType,
		Message:   msg,
		Error:     err,
	}

	em.errorLogs = append(em.errorLogs, log)
	if len(em.errorLogs) > em.maxLogs {
		em.errorLogs = em.errorLogs[1:]
	}
}

// GetErrors は指定された重大度以上のエラーログを取得する
func (em *EventMonitor) GetErrors(minSeverity ErrorSeverity) []ErrorLog {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]ErrorLog, 0)
	for _, log := range em.errorLogs {
		if log.Severity >= minSeverity {
			result = append(result, log)
		}
	}
	return result
}

// ClearLogs はログをクリアする
func (em *EventMonitor) ClearLogs() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.errorLogs = em.errorLogs[:0]
}

// EventManager はイベントの管理を行う
type EventManager struct {
	subscribers     map[EventType][]EventSubscriber
	mu              sync.RWMutex
	batchMode       bool
	batchEvents     []Event
	updateQueue     *UpdateQueue
	snapshots       []RecoverySnapshot
	maxSnapshots    int
	onError         func(error)
	errorHandlers   map[EventType]func(Event, error)
	monitor         *EventMonitor
	recoveryManager *RecoveryManager
}

// NewEventManager は新しいEventManagerを作成する
func NewEventManager() *EventManager {
	em := &EventManager{
		subscribers:   make(map[EventType][]EventSubscriber),
		batchEvents:   make([]Event, 0),
		updateQueue:   NewUpdateQueue(),
		snapshots:     make([]RecoverySnapshot, 0),
		maxSnapshots:  10,
		errorHandlers: make(map[EventType]func(Event, error)),
		monitor:       NewEventMonitor(1000),
	}
	em.recoveryManager = NewRecoveryManager(em, em.monitor)
	return em
}

// Subscribe はイベントタイプに対するサブスクライバーを登録する
func (em *EventManager) Subscribe(eventType EventType, subscriber EventSubscriber) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.subscribers[eventType] == nil {
		em.subscribers[eventType] = make([]EventSubscriber, 0)
	}
	em.subscribers[eventType] = append(em.subscribers[eventType], subscriber)
}

// Unsubscribe はイベントタイプに対するサブスクライバーを登録解除する
func (em *EventManager) Unsubscribe(eventType EventType, subscriber EventSubscriber) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if subscribers, ok := em.subscribers[eventType]; ok {
		// サブスクライバーを検索して削除
		for i, sub := range subscribers {
			// 関数ポインタの比較は直接できないため、文字列表現を比較
			if getFuncPtr(sub) == getFuncPtr(subscriber) {
				// スライスから要素を削除
				em.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
}

// getFuncPtr は関数のポインタを文字列として取得する
func getFuncPtr(f interface{}) string {
	return fmt.Sprintf("%v", f)
}

// SetErrorHandler はイベントタイプ別のエラーハンドラを設定する
func (em *EventManager) SetErrorHandler(eventType EventType, handler func(Event, error)) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.errorHandlers[eventType] = handler
}

// SetGlobalErrorHandler はグローバルエラーハンドラを設定する
func (em *EventManager) SetGlobalErrorHandler(handler func(error)) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.onError = handler
}

// CreateSnapshot は現在の状態のスナップショットを作成する
func (em *EventManager) CreateSnapshot() {
	em.mu.Lock()
	defer em.mu.Unlock()

	snapshot := RecoverySnapshot{
		Timestamp: time.Now(),
		Events:    make([]Event, len(em.batchEvents)),
		States:    make(map[string]interface{}),
	}
	copy(snapshot.Events, em.batchEvents)

	em.snapshots = append(em.snapshots, snapshot)
	if len(em.snapshots) > em.maxSnapshots {
		em.snapshots = em.snapshots[1:]
	}
}

// RecoverFromSnapshot は指定された時点のスナップショットから復元する
func (em *EventManager) RecoverFromSnapshot(timestamp time.Time) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	for i := len(em.snapshots) - 1; i >= 0; i-- {
		if em.snapshots[i].Timestamp.Before(timestamp) || em.snapshots[i].Timestamp.Equal(timestamp) {
			// バッチモードで状態を復元
			em.BeginBatch()
			for _, event := range em.snapshots[i].Events {
				em.Publish(event)
			}
			em.EndBatch()
			return nil
		}
	}
	return fmt.Errorf("no snapshot found before %v", timestamp)
}

// handleError はエラーを処理し、必要に応じて復元を試みる
func (em *EventManager) handleError(event Event, err error) {
	// エラーの構造化
	var structuredErr *StructuredError
	if !errors.As(err, &structuredErr) {
		// 未構造化のエラーを構造化する
		structuredErr = NewStructuredError(
			ErrorCategoryUnknown,
			"Unhandled error occurred",
			err,
			event,
		)
	}

	// エラーの重大度を判定
	severity := SeverityError
	if _, ok := err.(*RecoveryError); ok {
		severity = SeverityCritical
	}

	// エラーレポートを作成
	report := NewErrorReport(structuredErr, event.GetType())

	// エラーをログに記録
	em.monitor.LogError(severity, event.GetType(), structuredErr, structuredErr.Message)

	// 復元を試みる
	if recoveryErr := em.recoveryManager.AttemptRecovery(event, structuredErr); recoveryErr != nil {
		// リカバリー情報を追加
		report.WithRecoveryInfo(&RecoveryInfo{
			AttemptCount:  1,
			LastStrategy:  em.recoveryManager.strategy,
			SuccessCount:  0,
			FailureCount:  1,
			LastAttemptAt: time.Now(),
		})

		// 復元に失敗した場合、エラーハンドラを呼び出す
		if handler, ok := em.errorHandlers[event.GetType()]; ok {
			handler(event, structuredErr)
		} else if em.onError != nil {
			em.onError(structuredErr)
		}
		return
	}

	// 復元に成功した場合
	report.WithRecoveryInfo(&RecoveryInfo{
		AttemptCount:   1,
		LastStrategy:   em.recoveryManager.strategy,
		SuccessCount:   1,
		FailureCount:   0,
		LastAttemptAt:  time.Now(),
		RecoveredState: event.GetCurrentState(),
	})

	// 成功ログを記録
	em.monitor.LogError(SeverityInfo, event.GetType(), nil,
		fmt.Sprintf("Successfully recovered from error: %s", structuredErr.Message))
}

// SetRecoveryStrategy は復元戦略を設定する
func (em *EventManager) SetRecoveryStrategy(strategy RecoveryStrategy) {
	em.recoveryManager.SetStrategy(strategy)
}

// Publish はイベントを発行する
func (em *EventManager) Publish(event Event) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if event.HasError() {
		em.handleError(event, event.GetError())
	}

	if em.batchMode {
		em.batchEvents = append(em.batchEvents, event)
		return
	}

	// 非バッチモードの場合は更新キューに追加し、即時に処理
	em.updateQueue.Add(event)
	// TODO: 非バッチモード時は即時に処理したいが不具合があるので保留。コマンドで行なっている処理をすべて代替する必要がある
	// em.ProcessUpdates()
}

// ProcessUpdates は更新キューを処理する
func (em *EventManager) ProcessUpdates() {
	em.updateQueue.Flush(func(event Event) {
		em.publishEvent(event)
	})
}

// BeginBatch はバッチモードを開始する
func (em *EventManager) BeginBatch() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.batchMode = true
}

// EndBatch はバッチモードを終了し、蓄積されたイベントを発行する
func (em *EventManager) EndBatch() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.batchMode = false

	// バッファイベントを優先して処理
	for _, event := range em.batchEvents {
		if event.GetType() == BufferEventType {
			em.updateQueue.Add(event)
		}
	}

	// その他のイベントを処理
	for _, event := range em.batchEvents {
		if event.GetType() != BufferEventType {
			em.updateQueue.Add(event)
		}
	}

	em.batchEvents = em.batchEvents[:0]

	// 更新キューを処理
	em.ProcessUpdates()
}

// publishEvent は単一のイベントを発行する
func (em *EventManager) publishEvent(event Event) {
	if subscribers, ok := em.subscribers[event.GetType()]; ok {
		for _, subscriber := range subscribers {
			subscriber(event)
		}
	}
}

// ClearBatch はバッチモードをキャンセルし、蓄積されたイベントをクリアする
func (em *EventManager) ClearBatch() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.batchMode = false
	em.batchEvents = em.batchEvents[:0]
}

// GetErrorStats はエラー統計を取得する
func (em *EventManager) GetErrorStats() map[EventType]int {
	stats := make(map[EventType]int)
	logs := em.monitor.GetErrors(SeverityError)
	for _, log := range logs {
		stats[log.EventType]++
	}
	return stats
}
