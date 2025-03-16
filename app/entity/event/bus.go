package event

import (
	"context"
	"fmt"
	"sync"
)

// Bus はイベントの発行と購読を管理するイベントバスです。
type Bus struct {
	handlers       map[EventType][]Handler
	eventChan      chan Event
	responseChans  map[EventType]chan Event
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	defaultHandler Handler
}

// NewBus は新しいイベントバスを作成します。
func NewBus() *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := &Bus{
		handlers:      make(map[EventType][]Handler),
		eventChan:     make(chan Event, 100), // バッファ付きチャネル
		responseChans: make(map[EventType]chan Event),
		ctx:           ctx,
		cancel:        cancel,
	}

	// イベント処理のゴルーチンを開始
	bus.wg.Add(1)
	go bus.processEvents()

	return bus
}

// Subscribe はイベントタイプに対するハンドラーを登録します。
func (b *Bus) Subscribe(handler Handler) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, eventType := range handler.GetHandledEventTypes() {
		b.handlers[eventType] = append(b.handlers[eventType], handler)
	}
}

// RegisterResponseChannel はイベントタイプに対するレスポンスチャネルを登録します。
func (b *Bus) RegisterResponseChannel(eventType EventType, responseChan chan Event) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.responseChans[eventType] = responseChan
}

// SetDefaultHandler はどのハンドラーにも処理されなかったイベントを処理するデフォルトハンドラーを設定します。
func (b *Bus) SetDefaultHandler(handler Handler) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.defaultHandler = handler
}

// Publish はイベントをバスに発行します。
func (b *Bus) Publish(event Event) {
	select {
	case b.eventChan <- event:
		// イベントが送信された
	case <-b.ctx.Done():
		// バスが停止された
	}
}

// Shutdown はイベントバスを終了します。
func (b *Bus) Shutdown() {
	b.cancel()  // コンテキストをキャンセルして全てのゴルーチンに停止を通知
	b.wg.Wait() // 全てのゴルーチンが終了するのを待つ
}

// processEvents はイベントチャネルからイベントを受け取り、適切なハンドラーに配送します。
func (b *Bus) processEvents() {
	defer b.wg.Done()

	for {
		select {
		case event := <-b.eventChan:
			b.dispatchEvent(event)
		case <-b.ctx.Done():
			return // コンテキストがキャンセルされたら終了
		}
	}
}

// dispatchEvent はイベントを適切なハンドラーに配送します。
func (b *Bus) dispatchEvent(event Event) {
	b.mutex.RLock()
	handlers, exists := b.handlers[event.Type]
	defaultHandler := b.defaultHandler
	responseChan, hasResponseChan := b.responseChans[event.Type]
	b.mutex.RUnlock()

	var handled bool
	var lastErr error

	// 登録されたハンドラーにイベントを配送
	if exists {
		for _, handler := range handlers {
			success, err := handler.HandleEvent(event)
			if err != nil {
				lastErr = fmt.Errorf("handler error: %w", err)
			}
			if success {
				handled = true
			}
		}
	}

	// 誰も処理しなかった場合はデフォルトハンドラーに配送
	if !handled && defaultHandler != nil {
		defaultHandler.HandleEvent(event)
	}

	// レスポンスチャネルがあれば結果を送信
	if hasResponseChan {
		response := NewResponseEvent(handled && lastErr == nil, "", lastErr)
		select {
		case responseChan <- response:
			// レスポンスが送信された
		case <-b.ctx.Done():
			// バスが停止された
		}
	}
}

// PublishAndWaitResponse はイベントを発行し、応答を待ちます。
func (b *Bus) PublishAndWaitResponse(event Event) (Event, error) {
	responseChan := make(chan Event, 1)
	defer close(responseChan)

	// 一時的にレスポンスチャネルを登録
	b.RegisterResponseChannel(event.Type, responseChan)
	defer func() {
		b.mutex.Lock()
		delete(b.responseChans, event.Type)
		b.mutex.Unlock()
	}()

	// イベントを発行
	b.Publish(event)

	// 応答を待つ
	select {
	case response := <-responseChan:
		return response, nil
	case <-b.ctx.Done():
		return Event{}, fmt.Errorf("bus is shutting down")
	}
}
