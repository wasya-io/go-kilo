package editor

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/wasya-io/go-kilo/editor/events"
)

// KeyEvent はキー入力イベントを表す
type KeyEvent struct {
	Type KeyEventType
	Rune rune // 通常の文字入力の場合
	Key  Key  // 特殊キーの場合
}

// KeyEventType はキーイベントの種類を表す
type KeyEventType int

const (
	KeyEventChar KeyEventType = iota + 1 // 1から開始
	KeyEventSpecial
	KeyEventControl
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

	// 特殊キーの処理（バッファの破棄なし）
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
			}
		}
		// エスケープシーケンスでない場合は通常の文字として処理
	}

	// 通常文字の処理
	if buf[0] >= 32 && buf[0] < 127 {
		return KeyEvent{Type: KeyEventChar, Rune: rune(buf[0])}, nil
	}

	// UTF-8文字の処理
	if (buf[0] & 0x80) != 0 {
		r, _ := utf8.DecodeRune(buf[:n])
		if r != utf8.RuneError {
			return KeyEvent{Type: KeyEventChar, Rune: r}, nil
		}
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
	// キー入力の情報をより分かりやすく表示
	var keyInfo string
	switch event.Type {
	case KeyEventChar:
		keyInfo = fmt.Sprintf("Input: Char '%c'", event.Rune)
	case KeyEventSpecial:
		keyInfo = fmt.Sprintf("Input: %s", h.getKeyName(event.Key))
	case KeyEventControl:
		keyInfo = fmt.Sprintf("Input: %s", h.getKeyName(event.Key))
	default:
		keyInfo = "Input: Unknown"
	}

	// UIのデバッグメッセージを設定
	if ui, ok := h.editor.(interface{ GetUI() *UI }); ok {
		ui.GetUI().SetDebugMessage(keyInfo)
	}

	switch event.Type {
	case KeyEventChar, KeyEventSpecial:
		// 通常の文字入力または特殊キー入力時は警告状態をクリア
		if h.quitWarningShown {
			h.quitWarningShown = false
			h.editor.SetStatusMessage("")
		}
		if event.Type == KeyEventChar {
			return NewInsertCharCommand(h.editor, event.Rune), nil
		}
		return h.createSpecialKeyCommand(event.Key), nil
	case KeyEventControl:
		switch event.Key {
		case KeyCtrlC:
			isDirty := h.editor.IsDirty()
			if h.quitWarningShown {
				h.quitWarningShown = false // リセット
				return NewQuitCommand(h.editor), nil
			}
			// 未編集なら即終了、編集済みなら警告表示
			if !isDirty {
				return NewQuitCommand(h.editor), nil
			}
			h.quitWarningShown = true
			h.editor.SetStatusMessage("Warning! File has unsaved changes. Press Ctrl-C again to quit.")
			return nil, nil
		case KeyCtrlS, KeyCtrlQ:
			return h.createControlKeyCommand(event.Key), nil
		default:
			// 他のコントロールキーの場合は警告状態をクリア
			if h.quitWarningShown {
				h.quitWarningShown = false
				h.editor.SetStatusMessage("")
			}
			return nil, nil
		}
	default:
		return nil, nil
	}
}

// getKeyName は特殊キーの名前を返す
func (h *InputHandler) getKeyName(key Key) string {
	switch key {
	case KeyArrowUp:
		return "↑"
	case KeyArrowDown:
		return "↓"
	case KeyArrowLeft:
		return "←"
	case KeyArrowRight:
		return "→"
	case KeyBackspace:
		return "Backspace"
	case KeyEnter:
		return "Enter"
	case KeyTab:
		return "Tab"
	case KeyShiftTab:
		return "Shift+Tab"
	case KeyCtrlC:
		return "Ctrl+C"
	case KeyCtrlQ:
		return "Ctrl+Q"
	case KeyCtrlS:
		return "Ctrl+S"
	case KeyEsc:
		return "Esc"
	default:
		return fmt.Sprintf("Key(%d)", key)
	}
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
