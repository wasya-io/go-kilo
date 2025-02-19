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
	KeyTab
	KeyShiftTab // Add Shift+Tab key
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

	// Shift+Tab の処理（エスケープシーケンス '[Z' を検出）
	if n >= 3 && buf[0] == '\x1b' && buf[1] == '[' && buf[2] == 'Z' {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyShiftTab}, nil
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

// min は2つの整数のうち小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InputHandler はキー入力とその処理を管理する
type InputHandler struct {
	keyReader KeyReader
	editor    EditorOperations
}

// NewInputHandler は新しいInputHandlerを作成する
func NewInputHandler(editor EditorOperations) *InputHandler {
	return &InputHandler{
		keyReader: NewStandardKeyReader(),
		editor:    editor,
	}
}

// HandleKeypress はキー入力からコマンドを生成する
func (ih *InputHandler) HandleKeypress() (EditorCommand, error) {
	event, err := ih.keyReader.ReadKey()
	if err != nil {
		return nil, err
	}

	switch event.Type {
	case KeyEventChar:
		// 括弧類の補完処理
		if shouldAutoClose := ih.handleBracketPair(event.Rune); !shouldAutoClose {
			// 通常の文字入力を処理
			return NewInsertCharCommand(ih.editor, event.Rune), nil
		}
		return nil, nil
	case KeyEventSpecial:
		// 特殊キーを処理
		return ih.handleSpecialKey(event.Key)
	case KeyEventControl:
		// コントロールキーを処理
		return ih.handleControlKey(event.Key)
	}

	return nil, nil
}

// handleBracketPair は括弧や引用符の補完処理を行う
func (ih *InputHandler) handleBracketPair(r rune) bool {
	pairs := map[rune]rune{
		'(':  ')',
		'{':  '}',
		'[':  ']',
		'"':  '"',
		'\'': '\'',
		'`':  '`',
	}

	closeChar, isPair := pairs[r]
	if !isPair {
		return false
	}

	if r == '"' || r == '\'' || r == '`' {
		if ih.hasQuoteInLine(r) {
			NewInsertCharCommand(ih.editor, r).Execute()
			return true
		}
	}

	NewInsertCharCommand(ih.editor, r).Execute()
	NewInsertCharCommand(ih.editor, closeChar).Execute()
	NewMoveCursorCommand(ih.editor, CursorLeft).Execute()
	return true
}

// hasQuoteInLine は現在の行の左側に指定された引用符があるかどうかを確認する
func (ih *InputHandler) hasQuoteInLine(quote rune) bool {
	cursor := ih.editor.GetCursor()
	content := ih.editor.GetContent(cursor.Y)
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
func (ih *InputHandler) handleSpecialKey(key Key) (EditorCommand, error) {
	switch key {
	case KeyTab:
		// タブをスペースに変換する複合コマンドを作成
		tabWidth := ih.editor.GetConfig().TabWidth
		for i := 0; i < tabWidth; i++ {
			NewInsertCharCommand(ih.editor, ' ').Execute()
		}
		return nil, nil
	case KeyShiftTab:
		return ih.handleShiftTab(), nil
	case KeyArrowUp:
		return NewMoveCursorCommand(ih.editor, CursorUp), nil
	case KeyArrowDown:
		return NewMoveCursorCommand(ih.editor, CursorDown), nil
	case KeyArrowRight:
		return NewMoveCursorCommand(ih.editor, CursorRight), nil
	case KeyArrowLeft:
		return NewMoveCursorCommand(ih.editor, CursorLeft), nil
	case KeyBackspace:
		return NewDeleteCharCommand(ih.editor), nil
	case KeyEnter:
		return NewInsertNewlineCommand(ih.editor), nil
	}
	return nil, nil
}

// handleShiftTab は Shift+Tab の処理を行う
func (ih *InputHandler) handleShiftTab() EditorCommand {
	cursor := ih.editor.GetCursor()
	if cursor.X == 0 {
		return nil
	}

	content := ih.editor.GetContent(cursor.Y)
	if content == "" {
		return nil
	}

	runes := []rune(content)
	spaceCount := 0
	pos := cursor.X - 1
	for pos >= 0 && runes[pos] == ' ' {
		spaceCount++
		pos--
	}

	if spaceCount == 0 {
		return nil
	}

	tabWidth := ih.editor.GetConfig().TabWidth
	targetSpaces := (spaceCount - 1) / tabWidth * tabWidth
	if targetSpaces == spaceCount {
		targetSpaces = ((spaceCount-1)/tabWidth - 1) * tabWidth
	}
	targetSpaces = max(0, targetSpaces)

	deleteCount := min(spaceCount-targetSpaces, spaceCount)
	for i := 0; i < deleteCount; i++ {
		NewDeleteCharCommand(ih.editor).Execute()
	}
	return nil
}

// handleControlKey は制御キーの処理を行う
func (ih *InputHandler) handleControlKey(key Key) (EditorCommand, error) {
	switch key {
	case KeyCtrlQ, KeyCtrlC:
		return NewQuitCommand(ih.editor), nil
	case KeyCtrlS:
		return NewSaveCommand(ih.editor), nil
	}
	return nil, nil
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (ih *InputHandler) SetKeyReader(reader KeyReader) {
	ih.keyReader = reader
}
