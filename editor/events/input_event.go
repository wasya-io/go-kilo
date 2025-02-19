package events

// InputEvent はキー入力イベントを表す
type InputEvent struct {
	BaseEvent
	KeyType    KeyEventType
	Rune       rune // 通常の文字入力の場合
	SpecialKey Key  // 特殊キーの場合
}

// KeyEventType はキーイベントの種類
type KeyEventType int

const (
	KeyEventChar KeyEventType = iota
	KeyEventSpecial
	KeyEventControl
)

// Key は特殊キーの種類
type Key int

const (
	KeyArrowUp Key = iota
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyBackspace
	KeyEnter
	KeyCtrlQ
	KeyCtrlC
	KeyCtrlS
	KeyEsc
	KeyTab
	KeyShiftTab
)

// NewInputEvent は新しいInputEventを作成する
func NewInputEvent(keyType KeyEventType, r rune, key Key) *InputEvent {
	return &InputEvent{
		BaseEvent:  NewBaseEvent(InputEventType),
		KeyType:    keyType,
		Rune:       r,
		SpecialKey: key,
	}
}
