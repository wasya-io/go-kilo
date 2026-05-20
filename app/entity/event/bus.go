package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wasya-io/go-kilo/app/entity/core"
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
	synchronous    bool
	metrics        *core.MetricsCollector
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

// SetMetricsCollector はメトリクス収集器を設定します。
func (b *Bus) SetMetricsCollector(m *core.MetricsCollector) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.metrics = m
}

// EventQueueLength は現在のイベントキュー長を返します。
func (b *Bus) EventQueueLength() int {
	return len(b.eventChan)
}

// SetSynchronous はイベントバスの同期モードを設定します。
// テスト用：trueの場合、Publishはイベントを即座に処理します。
func (b *Bus) SetSynchronous(sync bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.synchronous = sync
}

// Publish はイベントをバスに発行します。
func (b *Bus) Publish(event Event) {
	b.mutex.RLock()
	isSync := b.synchronous
	metrics := b.metrics
	b.mutex.RUnlock()

	if metrics != nil {
		metrics.RecordEventPublished(string(event.Type))
	}

	if isSync {
		b.dispatchEvent(event)
		if metrics != nil {
			metrics.RecordEventQueueLength(b.EventQueueLength())
		}
		return
	}

	select {
	case b.eventChan <- event:
		// イベントが送信された
		if metrics != nil {
			metrics.RecordEventQueueLength(b.EventQueueLength())
		}
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
			if b.metrics != nil {
				b.metrics.RecordEventQueueLength(b.EventQueueLength())
			}
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
	start := time.Now()

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

	duration := time.Since(start)
	if b.metrics != nil {
		b.metrics.RecordEventHandled(string(event.Type), handled, duration)
	}

	// 誰も処理しなかった場合はデフォルトハンドラーに配送
	if !handled && defaultHandler != nil {
		success, err := defaultHandler.HandleEvent(event)
		if err != nil {
			lastErr = fmt.Errorf("default handler error: %w", err)
		}
		if success {
			handled = true
		}
	}

	// カスタムエラー伝播: ハンドラーからエラーが返され、かつ対象が TypeError でなければ発行
	if lastErr != nil && event.Type != TypeError {
		errEvent := NewErrorEvent(&EventError{
			OriginalEventType: event.Type,
			Err:               lastErr,
		}, event)
		// デッドロックを回避するため非同期で発行
		go b.Publish(errEvent)
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
