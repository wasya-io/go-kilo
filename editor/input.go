package editor

import (
	"fmt"
	"os"
	"unicode/utf8"
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
	KeyEventChar    KeyEventType = iota // 通常の文字入力
	KeyEventSpecial                     // 特殊キー（矢印キーなど）
	KeyEventControl                     // 制御キー（Ctrl+など）
)

// Key は特殊キーの種類を表す
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
	KeyTab // Add Tab key
)

// KeyReader はキー入力を読み取るインターフェース
type KeyReader interface {
	ReadKey() (KeyEvent, error)
}

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
	// バッファにイベントがある場合は、それを返す
	if len(kr.inputBuffer) > 0 {
		event := kr.inputBuffer[0]
		kr.inputBuffer = kr.inputBuffer[1:]
		return event, nil
	}

	// 最初の読み取り
	buf := make([]byte, 32)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		return KeyEvent{}, err
	}

	// タブキーの処理
	if n == 1 && buf[0] == '\t' {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyTab}, nil
	}

	// エスケープシーケンスの処理
	if n > 0 && buf[0] == '\x1b' {
		return kr.handleEscapeSequence(buf[:n])
	}

	// バックスペースの処理
	if n == 1 && buf[0] == 127 {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyBackspace}, nil
	}

	// 制御キーの処理
	if n == 1 && buf[0] < 32 {
		switch buf[0] {
		case 'q' & 0x1f:
			return KeyEvent{Type: KeyEventControl, Key: KeyCtrlQ}, nil
		case 'c' & 0x1f:
			return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, nil
		case 's' & 0x1f:
			return KeyEvent{Type: KeyEventControl, Key: KeyCtrlS}, nil
		case '\r':
			return KeyEvent{Type: KeyEventSpecial, Key: KeyEnter}, nil
		}
	}

	// マルチバイト文字の処理
	data := buf[:n]
	processed := 0
	for processed < len(data) {
		r, size := utf8.DecodeRune(data[processed:])
		if r == utf8.RuneError {
			if size == 1 {
				// 不完全なマルチバイト文字の場合、追加バイトを読み取る
				tmp := make([]byte, 4)
				n, err = os.Stdin.Read(tmp)
				if err != nil {
					return KeyEvent{}, err
				}
				data = append(data, tmp[:n]...)
				continue
			}
			return KeyEvent{}, fmt.Errorf("invalid input sequence")
		}

		// 有効な文字の場合はバッファに追加
		if utf8.ValidRune(r) && (r >= 32 || r == '\t') {
			kr.inputBuffer = append(kr.inputBuffer, KeyEvent{Type: KeyEventChar, Rune: r})
		}
		processed += size
	}

	// バッファから1つ取り出して返す
	if len(kr.inputBuffer) > 0 {
		event := kr.inputBuffer[0]
		kr.inputBuffer = kr.inputBuffer[1:]
		return event, nil
	}

	return KeyEvent{}, fmt.Errorf("no valid input")
}

// handleEscapeSequence はエスケープシーケンスを解析してKeyEventに変換する
func (kr *StandardKeyReader) handleEscapeSequence(buf []byte) (KeyEvent, error) {
	// ESCキー単体の場合の処理
	if len(buf) == 1 {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
	}

	// エスケープシーケンスの場合
	n := len(buf)
	if n < 3 {
		// より柔軟な読み取り処理
		seq := make([]byte, 3-n)
		nseq, err := os.Stdin.Read(seq)
		if err != nil {
			return KeyEvent{}, err
		}
		// タイムアウトまたは読み取りエラーの場合はESCキーとして扱う
		if nseq == 0 {
			return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
		}
		// バッファを結合
		buf = append(buf, seq[:nseq]...)
	}

	if buf[1] == '[' {
		switch buf[2] {
		case 'A':
			return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowUp}, nil
		case 'B':
			return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowDown}, nil
		case 'C':
			return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowRight}, nil
		case 'D':
			return KeyEvent{Type: KeyEventSpecial, Key: KeyArrowLeft}, nil
		}
	}

	// 未知のエスケープシーケンスの場合はESCキーとして扱う
	return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
}

// InputHandler はキー入力とその処理を管理する
type InputHandler struct {
	keyReader KeyReader
	editor    *Editor
}

// NewInputHandler は新しいInputHandlerを作成する
func NewInputHandler(editor *Editor) *InputHandler {
	return &InputHandler{
		keyReader: NewStandardKeyReader(),
		editor:    editor,
	}
}

// ProcessKeypress はキー入力を処理する
func (ih *InputHandler) ProcessKeypress() error {
	event, err := ih.keyReader.ReadKey()
	if err != nil {
		return err
	}

	switch event.Type {
	case KeyEventChar:
		// 括弧類の補完処理
		if shouldAutoClose := ih.handleBracketPair(event.Rune); !shouldAutoClose {
			// 通常の文字入力を処理
			ih.editor.buffer.InsertChar(event.Rune)
		}
	case KeyEventSpecial:
		// 特殊キーを処理
		if err := ih.handleSpecialKey(event.Key); err != nil {
			return err
		}
	case KeyEventControl:
		// コントロールキーを処理
		if err := ih.handleControlKey(event.Key); err != nil {
			return err
		}
	}

	return ih.editor.RefreshScreen()
}

// handleBracketPair は括弧や引用符の補完処理を行う
// 戻り値は補完処理を行ったかどうか
func (ih *InputHandler) handleBracketPair(r rune) bool {
	// 開き括弧と閉じ括弧のマッピング
	pairs := map[rune]rune{
		'(':  ')',
		'{':  '}',
		'[':  ']',
		'"':  '"',
		'\'': '\'',
		'`':  '`',
	}

	// 閉じ括弧があるかチェック
	closeChar, isPair := pairs[r]
	if !isPair {
		return false
	}

	// 引用符（", ', `）の場合、同じ行の左側に同じ文字があるかチェック
	if r == '"' || r == '\'' || r == '`' {
		if ih.hasQuoteInLine(r) {
			ih.editor.buffer.InsertChar(r)
			return true
		}
	}

	// 開き括弧を挿入
	ih.editor.buffer.InsertChar(r)
	// 閉じ括弧を挿入
	ih.editor.buffer.InsertChar(closeChar)
	// カーソルを一つ戻す
	ih.editor.buffer.MoveCursor(CursorLeft)
	return true
}

// hasQuoteInLine は現在の行の左側に指定された引用符があるかどうかを確認する
func (ih *InputHandler) hasQuoteInLine(quote rune) bool {
	cursor := ih.editor.buffer.GetCursor()
	content := ih.editor.buffer.GetContent(cursor.Y)
	if content == "" {
		return false
	}

	// カーソルの左側の文字列を検査
	runes := []rune(content)
	for i := 0; i < cursor.X && i < len(runes); i++ {
		if runes[i] == quote {
			return true
		}
	}
	return false
}

// handleSpecialKey は特殊キーの処理を行う
func (ih *InputHandler) handleSpecialKey(key Key) error {
	switch key {
	case KeyTab:
		// タブ文字をスペースに変換して挿入
		for i := 0; i < ih.editor.config.TabWidth; i++ {
			ih.editor.buffer.InsertChar(' ')
		}
	case KeyArrowUp:
		ih.editor.buffer.MoveCursor(CursorUp)
		ih.editor.UpdateScroll()
	case KeyArrowDown:
		ih.editor.buffer.MoveCursor(CursorDown)
		ih.editor.UpdateScroll()
	case KeyArrowRight:
		ih.editor.buffer.MoveCursor(CursorRight)
		ih.editor.UpdateScroll()
	case KeyArrowLeft:
		ih.editor.buffer.MoveCursor(CursorLeft)
		ih.editor.UpdateScroll()
	case KeyBackspace:
		ih.editor.buffer.DeleteChar()
	case KeyEnter:
		ih.editor.buffer.InsertNewline()
	}
	return nil
}

// handleControlKey は制御キーの処理を行う
func (ih *InputHandler) handleControlKey(key Key) error {
	switch key {
	case KeyCtrlQ, KeyCtrlC:
		if ih.editor.buffer.IsDirty() {
			ih.editor.setStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
			ih.editor.buffer.SetDirty(false)
			return nil
		}
		ih.editor.Quit()
	case KeyCtrlS:
		if err := ih.editor.SaveFile(); err != nil {
			ih.editor.setStatusMessage("Can't save! I/O error: %s", err)
		}
	}
	return nil
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (ih *InputHandler) SetKeyReader(reader KeyReader) {
	ih.keyReader = reader
}
