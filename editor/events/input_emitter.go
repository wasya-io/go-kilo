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
