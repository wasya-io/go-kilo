package events

// KeyEventType はキーイベントの種類を表す
type KeyEventType int

const (
	KeyEventChar KeyEventType = iota
	KeyEventSpecial
	KeyEventControl
	KeyEventIME
	KeyEventMouse // 追加: マウスイベント
)

// Key は特殊キーの種類を表す
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
	KeyMouseWheel // 追加: マウスホイール
)

// MouseAction はマウスイベントの種類を表す
type MouseAction int

const (
	MouseWheelUp MouseAction = iota
	MouseWheelDown
)

// MouseEventData はマウスイベントのデータを表す
type MouseEventData struct {
	Action MouseAction
	Row    int
	Col    int
}

// ModifierKeys はキーの修飾子を表す
type ModifierKeys uint8

const (
	ModNone  ModifierKeys = 0
	ModShift ModifierKeys = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// InputEvent はキー入力イベントを表す
type InputEvent struct {
	BaseEvent
	KeyType      KeyEventType
	Rune         rune
	SpecialKey   Key
	Modifiers    ModifierKeys
	IsComposing  bool
	ComposedText string
	MouseData    *MouseEventData // 追加: マウスイベントデータ
}

// NewInputEvent は新しいInputEventを作成する
func NewInputEvent(keyType KeyEventType, r rune, key Key) *InputEvent {
	return &InputEvent{
		BaseEvent:  BaseEvent{Type: InputEventType},
		KeyType:    keyType,
		Rune:       r,
		SpecialKey: key,
		Modifiers:  ModNone,
	}
}

// WithModifiers は修飾キーを設定する
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

// WithMouseData はマウスイベントデータを設定する
func (e *InputEvent) WithMouseData(action MouseAction, row, col int) *InputEvent {
	e.MouseData = &MouseEventData{
		Action: action,
		Row:    row,
		Col:    col,
	}
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
