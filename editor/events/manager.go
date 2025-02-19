package events

import (
	"sync"
)

// EventHandler はイベントを処理するハンドラ関数の型
type EventHandler func(Event)

// EventManager はイベントの発行と購読を管理する
type EventManager struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// NewEventManager は新しいEventManagerを作成する
func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Subscribe は指定されたイベントタイプのハンドラを登録する
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 重複登録を防ぐ
	for _, h := range em.handlers[eventType] {
		if &h == &handler {
			return
		}
	}

	em.handlers[eventType] = append(em.handlers[eventType], handler)
}

// Unsubscribe は指定されたイベントタイプのハンドラを登録解除する
func (em *EventManager) Unsubscribe(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	handlers := em.handlers[eventType]
	for i, h := range handlers {
		if &h == &handler {
			em.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish はイベントを発行し、登録されているハンドラに通知する
func (em *EventManager) Publish(event Event) {
	em.mu.RLock()
	handlers := em.handlers[event.Type()]
	em.mu.RUnlock()

	// ハンドラを同期的に実行
	for _, handler := range handlers {
		handler(event)
	}
}

// Clear は全てのイベントハンドラの登録を解除する
func (em *EventManager) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.handlers = make(map[EventType][]EventHandler)
}
