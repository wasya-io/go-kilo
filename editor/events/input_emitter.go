package events

// InputEventEmitter はキー入力イベントの発行を担当する
type InputEventEmitter interface {
	// EmitInputEvent はキー入力イベントを発行する
	EmitInputEvent(event *InputEvent)
	// Subscribe はInputEventのリスナーを登録する
	Subscribe(handler func(*InputEvent))
	// Unsubscribe はInputEventのリスナーを登録解除する
	Unsubscribe(handler func(*InputEvent))
}

// StandardInputEventEmitter は標準的なInputEventEmitterの実装
type StandardInputEventEmitter struct {
	eventManager *EventManager
}

// NewStandardInputEventEmitter は新しいStandardInputEventEmitterを作成する
func NewStandardInputEventEmitter(eventManager *EventManager) *StandardInputEventEmitter {
	return &StandardInputEventEmitter{
		eventManager: eventManager,
	}
}

// EmitInputEvent はキー入力イベントを発行する
func (e *StandardInputEventEmitter) EmitInputEvent(event *InputEvent) {
	if e.eventManager != nil {
		// バッチモードを開始
		e.eventManager.BeginBatch()
		defer e.eventManager.EndBatch()

		// 入力イベントを発行
		e.eventManager.Publish(event)
	}
}

// Subscribe はInputEventのリスナーを登録する
func (e *StandardInputEventEmitter) Subscribe(handler func(*InputEvent)) {
	if e.eventManager != nil {
		e.eventManager.Subscribe(InputEventType, func(event Event) {
			if inputEvent, ok := event.(*InputEvent); ok {
				handler(inputEvent)
			}
		})
	}
}

// Unsubscribe はInputEventのリスナーを登録解除する
func (e *StandardInputEventEmitter) Unsubscribe(handler func(*InputEvent)) {
	if e.eventManager != nil {
		e.eventManager.Unsubscribe(InputEventType, func(event Event) {
			if inputEvent, ok := event.(*InputEvent); ok {
				handler(inputEvent)
			}
		})
	}
}

// EmitKeyEvent はキーイベントを発行するヘルパーメソッド
func (e *StandardInputEventEmitter) EmitKeyEvent(keyType KeyEventType, r rune, key Key, mods ModifierKeys) {
	event := NewInputEvent(keyType, r, key).WithModifiers(mods)
	e.EmitInputEvent(event)
}

// EmitIMEEvent はIMEイベントを発行するヘルパーメソッド
func (e *StandardInputEventEmitter) EmitIMEEvent(isComposing bool, text string) {
	event := NewInputEvent(KeyEventIME, 0, KeyNone).WithIMEComposition(isComposing, text)
	e.EmitInputEvent(event)
}
