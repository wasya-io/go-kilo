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
	subscribers        map[EventType][]EventSubscriber
	mu                 sync.RWMutex
	batchMode          bool
	batchEvents        []Event
	processedEvents    []Event // 新しいフィールド：処理済みイベントを保持
	updateQueue        *UpdateQueue
	onError            func(error)
	errorHandlers      map[EventType]func(Event, error)
	monitor            *EventMonitor
	recoveryManager    *RecoveryManager
	systemEventHandler SystemEventHandler
	processingEvents   map[EventType]int
	maxRecursionDepth  int
}

// NewEventManager は新しいEventManagerを作成する
func NewEventManager() *EventManager {
	return &EventManager{
		subscribers:       make(map[EventType][]EventSubscriber),
		batchEvents:       make([]Event, 0),
		processedEvents:   make([]Event, 0), // 初期化
		updateQueue:       NewUpdateQueue(),
		errorHandlers:     make(map[EventType]func(Event, error)),
		monitor:           NewEventMonitor(1000),
		processingEvents:  make(map[EventType]int),
		maxRecursionDepth: 3,
	}
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

// SetRecoveryManager はRecoveryManagerを設定する
func (em *EventManager) SetRecoveryManager(rm *RecoveryManager) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if rm == nil {
		return // nilの場合は何もしない
	}

	em.recoveryManager = rm
	// recoveryManagerに対してEventManagerのrecover処理を設定
	rm.SetRecoveryCallback(func(event Event, err error) error {
		// エラーハンドラが設定されている場合はそちらを優先
		if handler, ok := em.errorHandlers[event.GetType()]; ok {
			handler(event, err)
			return nil
		}
		// グローバルエラーハンドラにフォールバック
		if em.onError != nil {
			em.onError(err)
		}
		return nil
	})
}

// GetRecoveryManager はRecoveryManagerを取得する
func (em *EventManager) GetRecoveryManager() *RecoveryManager {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.recoveryManager
}

// handleError はエラーを処理し、必要に応じて復元を試みる
func (em *EventManager) handleError(event Event, err error) {
	// システムイベントの場合、特別なエラー処理を行う
	if systemEvent, ok := event.(SystemEvent); ok {
		em.handleSystemEventError(systemEvent, err)
		return
	}

	// エラーの構造化
	var structuredErr *StructuredError
	if !errors.As(err, &structuredErr) {
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

	// エラーをログに記録
	em.monitor.LogError(severity, event.GetType(), structuredErr, structuredErr.Message)

	// 復元を試みる
	if em.recoveryManager != nil {
		if recoveryErr := em.recoveryManager.AttemptRecovery(event, structuredErr); recoveryErr != nil {
			if handler, ok := em.errorHandlers[event.GetType()]; ok {
				handler(event, structuredErr)
			} else if em.onError != nil {
				em.onError(structuredErr)
			}
		}
	}
}

// handleSystemEventError はシステムイベントのエラー処理を行う
func (em *EventManager) handleSystemEventError(event SystemEvent, err error) {
	// エラーをログに記録
	em.monitor.LogError(SeverityError, SystemEventType, err, fmt.Sprintf("System event error: %v", err))

	// システムイベント固有の復元処理
	switch event.GetSystemType() {
	case SystemSave:
		// 保存エラーの場合、ステータスを維持
		if saveEvent, ok := event.(*SaveEvent); ok {
			em.monitor.LogError(SeverityWarning, SystemEventType, err,
				fmt.Sprintf("Save failed for file: %s", saveEvent.Filename))
		}
	case SystemQuit:
		// 終了エラーの場合、警告メッセージを記録
		if quitEvent, ok := event.(*QuitEvent); ok {
			statusMsg := "Failed to quit"
			if quitEvent.SaveNeeded {
				statusMsg += ": unsaved changes"
			}
			em.monitor.LogError(SeverityWarning, SystemEventType, err, statusMsg)
		}
	}
}

// SetRecoveryStrategy は復元戦略を設定する
func (em *EventManager) SetRecoveryStrategy(strategy RecoveryStrategy) {
	em.recoveryManager.SetStrategy(strategy)
}

// Publish はイベントを発行する
func (em *EventManager) Publish(event Event) error {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// イベントを処理済みリストに追加
	em.processedEvents = append(em.processedEvents, event)

	// イベントループ検出
	if depth := em.processingEvents[event.GetType()]; depth >= em.maxRecursionDepth {
		em.monitor.LogError(SeverityWarning, event.GetType(), fmt.Errorf("event loop detected"),
			"Maximum recursion depth exceeded")
		return nil // イベントを無視して処理を継続
	}

	// イベント処理深度をインクリメント
	em.processingEvents[event.GetType()]++
	defer func() {
		em.processingEvents[event.GetType()]--
	}()

	if event.HasError() {
		em.handleError(event, event.GetError())
		return event.GetError()
	}

	if em.batchMode {
		em.batchEvents = append(em.batchEvents, event)
		return nil
	}

	// システムイベントの場合は即時処理
	if _, isSystemEvent := event.(SystemEvent); isSystemEvent {
		em.processSystemEvent(event)
		// エラーがある場合はhandleErrorで処理済み
		return nil
	}

	// 非バッチモードの場合は更新キューに追加し、即時に処理
	em.updateQueue.Add(event)
	em.ProcessUpdates()
	return nil
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
	// バッチ開始時のイベントカウントを記録
	em.batchEvents = em.batchEvents[:0]
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

	// 処理済みイベントリストにバッチイベントを追加
	em.processedEvents = append(em.processedEvents, em.batchEvents...)
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
	// イベントを処理済みリストに追加
	em.processedEvents = append(em.processedEvents, event)
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

// GetCurrentEvents は現在までに処理されたすべてのイベントのリストを返す
func (em *EventManager) GetCurrentEvents() []Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// 処理済みイベントをコピー
	events := make([]Event, len(em.processedEvents))
	copy(events, em.processedEvents)

	return events
}

// ClearEvents は処理済みイベントをクリアする
func (em *EventManager) ClearEvents() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.processedEvents = em.processedEvents[:0]
}

func (em *EventManager) processSystemEvent(event Event) {
	if systemEvent, ok := event.(SystemEvent); ok {
		if handler := em.systemEventHandler; handler != nil {
			var err error
			switch systemEvent.GetSystemType() {
			case SystemSave:
				if saveEvent, ok := systemEvent.(*SaveEvent); ok {
					err = handler.HandleSaveEvent(saveEvent)
				}
			case SystemQuit:
				if quitEvent, ok := systemEvent.(*QuitEvent); ok {
					err = handler.HandleQuitEvent(quitEvent)
				}
			case SystemStatus:
				if statusEvent, ok := systemEvent.(*StatusEvent); ok {
					err = handler.HandleStatusEvent(statusEvent)
				}
			}
			if err != nil {
				em.handleError(event, err)
			}
		}
	}
}

func (em *EventManager) RegisterSystemEventHandler(handler SystemEventHandler) {
	em.systemEventHandler = handler
}
