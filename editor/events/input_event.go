package events

// InputEvent はキー入力イベントを表す
type InputEvent struct {
	BaseEvent
	KeyType      KeyEventType
	Rune         rune // 通常の文字入力の場合
	SpecialKey   Key  // 特殊キーの場合
	Modifiers    ModifierKeys
	IsComposing  bool   // IME入力中かどうか
	ComposedText string // IME確定文字列
}

// KeyEventType はキーイベントの種類
type KeyEventType int

const (
	KeyEventChar KeyEventType = iota + 1 // 0ではなく1から開始
	KeyEventSpecial
	KeyEventControl
	KeyEventIME // IME関連のイベント
)

// Key は特殊キーの種類
type Key int

const (
	KeyNone Key = iota
	KeyArrowUp
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
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
)

// ModifierKeys は修飾キーの状態を表す
type ModifierKeys uint8

const (
	ModShift ModifierKeys = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
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

// WithModifiers は修飾キーの状態を設定する
func (e *InputEvent) WithModifiers(mods ModifierKeys) *InputEvent {
	e.Modifiers = mods
	return e
}

// WithIMEComposition はIME入力の状態を設定する
func (e *InputEvent) WithIMEComposition(isComposing bool, text string) *InputEvent {
	e.IsComposing = isComposing
	e.ComposedText = text
	return e
}

// HasModifier は指定された修飾キーが押されているかを確認する
func (e *InputEvent) HasModifier(mod ModifierKeys) bool {
	return e.Modifiers&mod != 0
}

// String はイベントの文字列表現を返す
func (e *InputEvent) String() string {
	switch e.KeyType {
	case KeyEventChar:
		return string(e.Rune)
	case KeyEventSpecial:
		return e.SpecialKey.String()
	case KeyEventControl:
		return "Ctrl+" + e.SpecialKey.String()
	case KeyEventIME:
		if e.IsComposing {
			return "Composing: " + e.ComposedText
		}
		return "Composed: " + e.ComposedText
	default:
		return "Unknown"
	}
}

// String はKeyの文字列表現を返す
func (k Key) String() string {
	switch k {
	case KeyArrowUp:
		return "↑"
	case KeyArrowDown:
		return "↓"
	case KeyArrowLeft:
		return "←"
	case KeyArrowRight:
		return "→"
	case KeyBackspace:
		return "⌫"
	case KeyEnter:
		return "⏎"
	case KeyTab:
		return "⇥"
	case KeyShiftTab:
		return "⇤"
	case KeyEsc:
		return "Esc"
	default:
		return "Unknown"
	}
}
