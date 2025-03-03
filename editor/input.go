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

// InputParser は入力解析の責務を担うインターフェース
type InputParser interface {
	parseControlKey(byte) (KeyEvent, bool)
	parseSpecialKey(byte) (KeyEvent, bool)
	parseEscapeSequence([]byte, int) (KeyEvent, error)
	parseCharacter([]byte, int) (KeyEvent, []KeyEvent, error)
	Parse([]byte, int) (KeyEvent, []KeyEvent, error)
}

// StandardInputParser は標準的な入力解析を実装する
type StandardInputParser struct{}

func NewStandardInputParser() *StandardInputParser {
	return &StandardInputParser{}
}

// Parse はバイトデータを解析してキーイベントを返す
func (p *StandardInputParser) Parse(buf []byte, n int) (KeyEvent, []KeyEvent, error) {
	// コントロールキーの処理
	if event, ok := p.parseControlKey(buf[0]); ok {
		return event, nil, nil
	}

	// 特殊キーの処理
	if event, ok := p.parseSpecialKey(buf[0]); ok {
		return event, nil, nil
	}

	// エスケープシーケンスの処理
	if buf[0] == '\x1b' {
		event, err := p.parseEscapeSequence(buf, n)
		if err == nil {
			return event, nil, nil
		}
	}

	// 文字の処理（UTF-8とASCII）
	return p.parseCharacter(buf, n)
}

// StandardKeyReader は標準入力からキーを読み取る実装
type StandardKeyReader struct {
	inputBuffer []KeyEvent  // 入力バッファ
	parser      InputParser // 入力解析器
}

// NewStandardKeyReader は標準入力からキーを読み取るKeyReaderを作成する
func NewStandardKeyReader() *StandardKeyReader {
	return &StandardKeyReader{
		inputBuffer: make([]KeyEvent, 0),
		parser:      NewStandardInputParser(),
	}
}

// ReadKey は標準入力から1つのキーイベントを読み取る
func (kr *StandardKeyReader) ReadKey() (KeyEvent, []KeyEvent, error) {
	// 標準入力から読み取り
	buf := make([]byte, 32)
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return KeyEvent{}, nil, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return KeyEvent{}, nil, fmt.Errorf("no input")
	}

	event, remainingEvents, err := kr.parser.Parse(buf, n)
	if err != nil {
		return KeyEvent{}, nil, err
	}

	return event, remainingEvents, nil
}

// parseControlKey はコントロールキーの解析を行う
func (p *StandardInputParser) parseControlKey(b byte) (KeyEvent, bool) {
	switch b {
	case 3: // Ctrl+C
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, true
	case 17: // Ctrl+Q
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlQ}, true
	case 19: // Ctrl-S
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlS}, true
	}
	return KeyEvent{}, false
}

// parseSpecialKey は特殊キーの解析を行う
func (p *StandardInputParser) parseSpecialKey(b byte) (KeyEvent, bool) {
	switch b {
	case 127: // Backspace
		return KeyEvent{Type: KeyEventSpecial, Key: KeyBackspace}, true
	case '\r': // Enter
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEnter}, true
	case '\t': // Tab
		return KeyEvent{Type: KeyEventSpecial, Key: KeyTab}, true
	}
	return KeyEvent{}, false
}

// parseEscapeSequence はエスケープシーケンスの解析を行う
func (p *StandardInputParser) parseEscapeSequence(buf []byte, n int) (KeyEvent, error) {
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
			return p.parseMouseEvent(buf, n)
		}
	}

	return KeyEvent{}, fmt.Errorf("unknown escape sequence")
}

// parseMouseEvent はマウスイベントの解析を行う
func (p *StandardInputParser) parseMouseEvent(buf []byte, n int) (KeyEvent, error) {
	if n >= 6 && buf[2] == '<' {
		var cb, cx, cy int
		if _, err := fmt.Sscanf(string(buf[3:n]), "%d;%d;%d", &cb, &cx, &cy); err == nil {
			switch cb {
			case 64: // スクロールアップ
				return KeyEvent{
					Type:        KeyEventMouse,
					Key:         KeyMouseWheel,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: MouseScrollUp,
				}, nil
			case 65: // スクロールダウン
				return KeyEvent{
					Type:        KeyEventMouse,
					Key:         KeyMouseWheel,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: MouseScrollDown,
				}, nil
			case 0: // 左クリック
				return KeyEvent{
					Type:        KeyEventMouse,
					Key:         KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: MouseLeftClick,
				}, nil
			case 2: // 右クリック
				return KeyEvent{
					Type:        KeyEventMouse,
					Key:         KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: MouseRightClick,
				}, nil
			case 1: // 中クリック
				return KeyEvent{
					Type:        KeyEventMouse,
					Key:         KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: MouseMiddleClick,
				}, nil
			}
		}
	}
	return KeyEvent{}, fmt.Errorf("unknown mouse event")
}

// parseCharacter はUTF-8/ASCII文字の解析を行う
func (p *StandardInputParser) parseCharacter(buf []byte, n int) (KeyEvent, []KeyEvent, error) {
	// UTF-8文字の処理
	if (buf[0] & 0x80) != 0 {
		r, size := utf8.DecodeRune(buf[:n])
		if r != utf8.RuneError {
			var remainingEvents []KeyEvent
			if n > size {
				remainingBytes := make([]byte, n-size)
				copy(remainingBytes, buf[size:n])
				for len(remainingBytes) > 0 {
					r, s := utf8.DecodeRune(remainingBytes)
					if r == utf8.RuneError {
						break
					}
					remainingEvents = append(remainingEvents, KeyEvent{Type: KeyEventChar, Rune: r})
					remainingBytes = remainingBytes[s:]
				}
			}
			return KeyEvent{Type: KeyEventChar, Rune: r}, remainingEvents, nil
		}
	}

	// ASCII文字の処理
	if buf[0] >= 32 && buf[0] < 127 {
		return KeyEvent{Type: KeyEventChar, Rune: rune(buf[0])}, nil, nil
	}

	return KeyEvent{}, nil, fmt.Errorf("unknown input")
}

// InputHandler は入力処理を管理する構造体
type InputHandler struct {
	editor           EditorOperations
	eventManager     *events.EventManager
	keyReader        KeyReader
	parser           InputParser
	quitWarningShown bool // Ctrl+C/Qで終了警告が表示されているかを追跡
}

// NewInputHandler は新しいInputHandlerインスタンスを作成する
func NewInputHandler(editor EditorOperations, eventManager *events.EventManager, keyReader KeyReader, parser InputParser) *InputHandler {
	handler := &InputHandler{
		editor:           editor,
		eventManager:     eventManager,
		keyReader:        keyReader,
		parser:           parser,
		quitWarningShown: false,
	}

	return handler
}

// SetKeyReader はキーリーダーを設定する
func (h *InputHandler) SetKeyReader(reader KeyReader) {
	h.keyReader = reader
}

// HandleKeypress はキー入力を処理する
func (h *InputHandler) HandleKeypress() (KeyEvent, []KeyEvent, error) {
	// TODO: KeyPressの流れと責務を整理する
	// ReadKey... Stdinから読み取り、入力をパース、バッファ操作(追加と読み出し)
	// ReadKeyはキー入力をイベントにパース、イベント配列を返すだけにしたい
	// 整理後・・・
	// HandleKeypressではキー入力をイベントにパース、イベント配列を返す
	// 入力バッファはEditorで保持する
	// 先頭イベントからcreateCommandする
	// RefreshScreen、command.Executeは今の流れどおり
	event, remainingEvents, err := h.keyReader.ReadKey()
	if err != nil {
		if err.Error() == "no input" {
			return KeyEvent{}, nil, nil
		}
		return KeyEvent{}, nil, err
	}

	return event, remainingEvents, nil
}
