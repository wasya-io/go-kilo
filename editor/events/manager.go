package events

import (
	"fmt"
	"sync"
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

// EventManager はイベントの管理を行う
type EventManager struct {
	subscribers map[EventType][]EventSubscriber
	mu          sync.RWMutex
	batchMode   bool
	batchEvents []Event
	updateQueue *UpdateQueue
}

// NewEventManager は新しいEventManagerを作成する
func NewEventManager() *EventManager {
	return &EventManager{
		subscribers: make(map[EventType][]EventSubscriber),
		batchEvents: make([]Event, 0),
		updateQueue: NewUpdateQueue(),
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

// Publish はイベントを発行する
func (em *EventManager) Publish(event Event) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.batchMode {
		em.batchEvents = append(em.batchEvents, event)
		return
	}

	// 非バッチモードの場合は更新キューに追加
	em.updateQueue.Add(event)
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
