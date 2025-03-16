package event_test

import (
	"sync"
	"testing"
	"time"

	"github.com/wasya-io/go-kilo/app/entity/event"
)

// テスト用のハンドラーを実装
type testHandler struct {
	eventTypes []event.EventType
	handleFn   func(event.Event) (bool, error)
	called     bool
	mutex      sync.Mutex
	eventCount int
}

func (h *testHandler) HandleEvent(e event.Event) (bool, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.called = true
	h.eventCount++
	return h.handleFn(e)
}

func (h *testHandler) GetHandledEventTypes() []event.EventType {
	return h.eventTypes
}

func (h *testHandler) WasCalled() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.called
}

func (h *testHandler) GetEventCount() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.eventCount
}

func newTestHandler(types []event.EventType, handleFn func(event.Event) (bool, error)) *testHandler {
	return &testHandler{
		eventTypes: types,
		handleFn:   handleFn,
		called:     false,
		eventCount: 0,
	}
}

func TestBusBasicPublishSubscribe(t *testing.T) {
	bus := event.NewBus()
	defer bus.Shutdown()

	// テストハンドラーの作成
	saveHandled := false
	saveHandler := newTestHandler(
		[]event.EventType{event.TypeSave},
		func(e event.Event) (bool, error) {
			saveHandled = true
			return true, nil
		},
	)

	// ハンドラーを登録
	bus.Subscribe(saveHandler)

	// イベントを発行
	bus.Publish(event.NewSaveEvent("test.txt", false))

	// ハンドラーが呼ばれたことを確認
	time.Sleep(50 * time.Millisecond) // 非同期処理を待つ
	if !saveHandled {
		t.Error("Save handler was not called")
	}
}

func TestBusPublishAndWaitResponse(t *testing.T) {
	bus := event.NewBus()
	defer bus.Shutdown()

	// テストハンドラーの作成
	handler := newTestHandler(
		[]event.EventType{event.TypeSave},
		func(e event.Event) (bool, error) {
			return true, nil
		},
	)

	// ハンドラーを登録
	bus.Subscribe(handler)

	// イベントを発行して応答を待つ
	response, err := bus.PublishAndWaitResponse(event.NewSaveEvent("test.txt", false))
	if err != nil {
		t.Errorf("Error waiting for response: %v", err)
	}

	// 応答の検証
	if response.Type != event.TypeResponse {
		t.Errorf("Expected response type %s, got %s", event.TypeResponse, response.Type)
	}

	if payload, ok := response.Payload.(event.ResponseEvent); !ok {
		t.Errorf("Expected ResponseEvent payload, got %T", response.Payload)
	} else if !payload.Success {
		t.Errorf("Expected success=true, got %v", payload.Success)
	}
}

func TestBusDefaultHandler(t *testing.T) {
	bus := event.NewBus()
	defer bus.Shutdown()

	defaultHandlerCalled := false
	defaultHandler := newTestHandler(
		[]event.EventType{event.TypeInput}, // このハンドラーはTypeInputのみ処理
		func(e event.Event) (bool, error) {
			defaultHandlerCalled = true
			return true, nil
		},
	)

	// デフォルトハンドラーを設定
	bus.SetDefaultHandler(defaultHandler)

	// 処理されないイベントタイプを発行
	bus.Publish(event.NewQuitEvent(false)) // TypeQuit

	// デフォルトハンドラーが呼ばれたことを確認
	time.Sleep(50 * time.Millisecond) // 非同期処理を待つ
	if !defaultHandlerCalled {
		t.Error("Default handler was not called")
	}
}

func TestBusMultipleHandlers(t *testing.T) {
	bus := event.NewBus()
	defer bus.Shutdown()

	// 複数のハンドラーを作成
	handler1 := newTestHandler(
		[]event.EventType{event.TypeSave},
		func(e event.Event) (bool, error) {
			return true, nil
		},
	)

	handler2 := newTestHandler(
		[]event.EventType{event.TypeSave},
		func(e event.Event) (bool, error) {
			return true, nil
		},
	)

	// ハンドラーを登録
	bus.Subscribe(handler1)
	bus.Subscribe(handler2)

	// イベントを発行
	bus.Publish(event.NewSaveEvent("test.txt", false))

	// 両方のハンドラーが呼ばれたことを確認
	time.Sleep(50 * time.Millisecond) // 非同期処理を待つ
	if !handler1.WasCalled() {
		t.Error("Handler 1 was not called")
	}
	if !handler2.WasCalled() {
		t.Error("Handler 2 was not called")
	}
}

func TestBusShutdown(t *testing.T) {
	bus := event.NewBus()

	// シャットダウン
	bus.Shutdown()

	// シャットダウン後の発行は無視されるはず
	handler := newTestHandler(
		[]event.EventType{event.TypeSave},
		func(e event.Event) (bool, error) {
			return true, nil
		},
	)
	bus.Subscribe(handler)
	bus.Publish(event.NewSaveEvent("test.txt", false))

	// ハンドラーが呼ばれていないことを確認
	time.Sleep(50 * time.Millisecond)
	if handler.WasCalled() {
		t.Error("Handler was called after shutdown")
	}
}
