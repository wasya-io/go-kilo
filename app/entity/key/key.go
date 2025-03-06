package key

type KeyEvent struct {
	Type        KeyEventType
	Rune        rune        // 通常の文字入力の場合
	Key         Key         // 特殊キーの場合
	MouseRow    int         // マウスイベントの行位置
	MouseCol    int         // マウスイベントの列位置
	MouseAction MouseAction // マウスイベントの種類（型をintからMouseActionに変更）
}

// KeyEventType はキーイベントの種類を表す
type KeyEventType int

const (
	KeyEventChar KeyEventType = iota + 1 // 1から開始
	KeyEventSpecial
	KeyEventControl
	KeyEventMouse
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
	KeyShiftTab // Add Shift+Tab key
	KeyMouseWheel
	KeyMouseClick // 追加：マウスクリック用のキー
)

// MouseAction はマウスアクションの種類を表す
type MouseAction int

const (
	MouseScrollUp MouseAction = iota + 1
	MouseScrollDown
	MouseLeftClick   // 追加：左クリック
	MouseRightClick  // 追加：右クリック
	MouseMiddleClick // 追加：中クリック
)
