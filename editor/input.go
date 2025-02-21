package editor

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/wasya-io/go-kilo/editor/events"
)

// KeyEvent はキー入力イベントを表す
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
)

// MouseAction はマウスアクションの種類を表す
type MouseAction int

const (
	MouseScrollUp MouseAction = iota + 1
	MouseScrollDown
)

// StandardKeyReader は標準入力からキーを読み取る実装
type StandardKeyReader struct {
	inputBuffer []KeyEvent // 入力バッファ
}

// NewStandardKeyReader は標準入力からキーを読み取るKeyReaderを作成する
func NewStandardKeyReader() *StandardKeyReader {
	return &StandardKeyReader{
		inputBuffer: make([]KeyEvent, 0),
	}
}

// ReadKey は標準入力から1つのキーイベントを読み取る
func (kr *StandardKeyReader) ReadKey() (KeyEvent, error) {
	if len(kr.inputBuffer) > 0 {
		event := kr.inputBuffer[0]
		kr.inputBuffer = kr.inputBuffer[1:]
		return event, nil
	}

	buf := make([]byte, 32)
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return KeyEvent{}, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return KeyEvent{}, fmt.Errorf("no input")
	}

	// 特殊キーの処理
	if buf[0] == 3 { // Ctrl+C
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, nil
	} else if buf[0] == 17 { // Ctrl+Q
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlQ}, nil
	} else if buf[0] == 19 { // Ctrl-S
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlS}, nil
	} else if buf[0] == 127 { // Backspace
		return KeyEvent{Type: KeyEventSpecial, Key: KeyBackspace}, nil
	} else if buf[0] == '\r' { // Enter
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEnter}, nil
	} else if buf[0] == '\t' { // Tab
		return KeyEvent{Type: KeyEventSpecial, Key: KeyTab}, nil
	}

	// エスケープシーケンスの処理
	if buf[0] == '\x1b' {
		if n == 1 {
			return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
		}
		if n >= 3 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowUp}, nil
			case 'B':
				return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowDown}, nil
			case 'C':
				return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowRight}, nil
			case 'D':
				return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowLeft}, nil
			case 'Z':
				return KeyEvent{Type: KeyEventSpecial, Key: KeyShiftTab}, nil
			case 'M', '<':
				// マウスイベントの処理
				if n >= 6 && buf[2] == '<' {
					// SGR形式のマウスイベント
					var cb, cx, cy int
					if _, err := fmt.Sscanf(string(buf[3:n]), "%d;%d;%d", &cb, &cx, &cy); err == nil {
						if cb == 64 { // スクロールアップ
							return KeyEvent{
								Type:        KeyEventMouse,
								Key:         KeyMouseWheel,
								MouseRow:    cy - 1,
								MouseCol:    cx - 1,
								MouseAction: MouseScrollUp,
							}, nil
						} else if cb == 65 { // スクロールダウン
							return KeyEvent{
								Type:        KeyEventMouse,
								Key:         KeyMouseWheel,
								MouseRow:    cy - 1,
								MouseCol:    cx - 1,
								MouseAction: MouseScrollDown,
							}, nil
						}
					}
				}
			}
		}
		return KeyEvent{}, fmt.Errorf("unknown escape sequence")
	}

	// UTF-8文字の処理
	if (buf[0] & 0x80) != 0 {
		// マルチバイト文字の処理
		r, size := utf8.DecodeRune(buf[:n])
		if r != utf8.RuneError {
			// 残りのバイトがある場合は入力バッファに保存
			if n > size {
				remainingBytes := make([]byte, n-size)
				copy(remainingBytes, buf[size:n])
				// 残りのバイトを処理して入力バッファに追加
				for len(remainingBytes) > 0 {
					r, s := utf8.DecodeRune(remainingBytes)
					if r == utf8.RuneError {
						break
					}
					kr.inputBuffer = append(kr.inputBuffer, KeyEvent{Type: KeyEventChar, Rune: r})
					remainingBytes = remainingBytes[s:]
				}
			}
			return KeyEvent{Type: KeyEventChar, Rune: r}, nil
		}
	}

	// ASCII文字の処理
	if buf[0] >= 32 && buf[0] < 127 {
		return KeyEvent{Type: KeyEventChar, Rune: rune(buf[0])}, nil
	}

	return KeyEvent{}, fmt.Errorf("unknown input")
}

// InputHandler は入力処理を管理する構造体
type InputHandler struct {
	editor           EditorOperations
	eventManager     *events.EventManager
	keyReader        KeyReader
	quitWarningShown bool // Ctrl+C/Qで終了警告が表示されているかを追跡
}

// NewInputHandler は新しいInputHandlerインスタンスを作成する
func NewInputHandler(editor EditorOperations, eventManager *events.EventManager) *InputHandler {
	handler := &InputHandler{
		editor:           editor,
		eventManager:     eventManager,
		keyReader:        NewStandardKeyReader(),
		quitWarningShown: false,
	}

	return handler
}

// SetKeyReader はキーリーダーを設定する
func (h *InputHandler) SetKeyReader(reader KeyReader) {
	h.keyReader = reader
}

// HandleKeypress はキー入力を処理する
func (h *InputHandler) HandleKeypress() (Command, error) {
	event, err := h.keyReader.ReadKey()
	if err != nil {
		if err.Error() == "no input" {
			return nil, nil
		}
		return nil, err
	}

	// コマンドを作成
	command, err := h.createCommand(event)
	if err != nil {
		return nil, err
	}

	// イベントマネージャーを通して通知（コマンドが生成された場合のみ）
	if h.eventManager != nil && command != nil {
		inputEvent := &events.InputEvent{
			BaseEvent:  events.BaseEvent{Type: events.InputEventType},
			KeyType:    events.KeyEventType(event.Type),
			Rune:       event.Rune,
			SpecialKey: events.Key(event.Key),
		}
		// イベントを発行するだけで、ハンドラは呼び出さない
		h.eventManager.Publish(inputEvent)
	}

	return command, nil
}

// createCommand はキーイベントからコマンドを作成する
func (h *InputHandler) createCommand(event KeyEvent) (Command, error) {
	switch event.Type {
	case KeyEventChar, KeyEventSpecial:
		// 警告状態をクリア
		if h.quitWarningShown {
			h.quitWarningShown = false
			h.editor.SetStatusMessage("")
		}
		if event.Type == KeyEventChar {
			return NewInsertCharCommand(h.editor, event.Rune), nil
		}
		return h.createSpecialKeyCommand(event.Key), nil
	case KeyEventControl:
		return h.createControlKeyCommand(event.Key), nil
	case KeyEventMouse:
		if event.Key == KeyMouseWheel {
			// マウスホイールイベントは専用のカーソル移動コマンドを使用
			switch event.MouseAction {
			case MouseScrollUp:
				return NewMoveCursorCommand(h.editor, MouseWheelUp), nil
			case MouseScrollDown:
				return NewMoveCursorCommand(h.editor, MouseWheelDown), nil
			}
		}
	}
	return nil, nil
}

// createSpecialKeyCommand は特殊キーに対応するコマンドを作成する
func (h *InputHandler) createSpecialKeyCommand(key Key) Command {
	switch key {
	case KeyArrowLeft:
		return NewMoveCursorCommand(h.editor, CursorLeft)
	case KeyArrowRight:
		return NewMoveCursorCommand(h.editor, CursorRight)
	case KeyArrowUp:
		return NewMoveCursorCommand(h.editor, CursorUp)
	case KeyArrowDown:
		return NewMoveCursorCommand(h.editor, CursorDown)
	case KeyBackspace:
		return NewDeleteCharCommand(h.editor)
	case KeyEnter:
		return NewInsertNewlineCommand(h.editor)
	case KeyTab:
		return NewInsertTabCommand(h.editor)
	case KeyShiftTab:
		return NewUndentCommand(h.editor)
	default:
		return nil
	}
}

// createControlKeyCommand はコントロールキーに対応するコマンドを作成する
func (h *InputHandler) createControlKeyCommand(key Key) Command {
	switch key {
	case KeyCtrlS:
		return NewSaveCommand(h.editor)
	case KeyCtrlQ:
		if h.editor.IsDirty() && !h.quitWarningShown {
			h.quitWarningShown = true
			h.editor.SetStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q again to quit.")
			return nil
		}
		return NewQuitCommand(h.editor)
	default:
		return nil
	}
}
