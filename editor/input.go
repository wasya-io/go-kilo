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
	if len(kr.inputBuffer) > 0 {
		event := kr.inputBuffer[0]
		kr.inputBuffer = kr.inputBuffer[1:]
		return event, nil
	}

	// より大きなバッファを使用して一度に読み取る
	buf := make([]byte, 32)
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		// Ctrl+Cが押された場合、KeyCtrlCイベントを返す
		if err == os.ErrInvalid || err.Error() == "inappropriate ioctl for device" {
			return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, nil
		}
		return KeyEvent{}, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return KeyEvent{}, fmt.Errorf("no input")
	}

	// Ctrl+C（ASCII値3）の直接的な検出
	if buf[0] == 3 {
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, nil
	}

	// エスケープシーケンスの検出
	if buf[0] == '\x1b' {
		// エスケープシーケンスの完全な読み取りを待つ
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
		// エスケープキー単体として処理
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
	}

	// 制御キーの処理
	switch buf[0] {
	case 3: // Ctrl-C
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlC}, nil
	case 17: // Ctrl-Q
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlQ}, nil
	case 19: // Ctrl-S
		return KeyEvent{Type: KeyEventControl, Key: KeyCtrlS}, nil
	case 127: // Backspace
		return KeyEvent{Type: KeyEventSpecial, Key: KeyBackspace}, nil
	case '\t': // Tab
		return KeyEvent{Type: KeyEventSpecial, Key: KeyTab}, nil
	case '\r': // Enter
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEnter}, nil
	}

	// UTF-8文字の処理
	if (buf[0] & 0x80) != 0 {
		// マルチバイト文字の完全な読み取りを試みる
		size := 1
		if (buf[0] & 0xE0) == 0xC0 {
			size = 2
		} else if (buf[0] & 0xF0) == 0xE0 {
			size = 3
		} else if (buf[0] & 0xF8) == 0xF0 {
			size = 4
		}

		// 必要なバイト数が揃っているか確認
		if n >= size {
			r, _ := utf8.DecodeRune(buf[:size])
			if r != utf8.RuneError {
				// 残りのバイトがある場合はバッファに保存
				if n > size {
					rest := buf[size:n]
					for len(rest) > 0 {
						r, size := utf8.DecodeRune(rest)
						if r == utf8.RuneError {
							break
						}
						kr.inputBuffer = append(kr.inputBuffer, KeyEvent{Type: KeyEventChar, Rune: r})
						rest = rest[size:]
					}
				}
				return KeyEvent{Type: KeyEventChar, Rune: r}, nil
			}
		}
	}

	// ASCII文字の処理
	if buf[0] >= 32 && buf[0] < 127 {
		// 残りのバイトがある場合はバッファに保存
		if n > 1 {
			for i := 1; i < n; i++ {
				if buf[i] >= 32 && buf[i] < 127 {
					kr.inputBuffer = append(kr.inputBuffer, KeyEvent{Type: KeyEventChar, Rune: rune(buf[i])})
				}
			}
		}
		return KeyEvent{Type: KeyEventChar, Rune: rune(buf[0])}, nil
	}

	return KeyEvent{}, fmt.Errorf("unknown input")
}

// handleEscapeSequence はエスケープシーケンスを処理する
func (kr *StandardKeyReader) handleEscapeSequence(buf []byte) (KeyEvent, error) {
	// ESCキー単体の場合の処理
	if len(buf) == 1 {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
	}

	// 矢印キーの場合、常に3バイトのシーケンスを期待
	moreBuf := make([]byte, 3)
	copied := copy(moreBuf, buf)
	if copied < 3 {
		n, err := os.Stdin.Read(moreBuf[copied:])
		if err != nil || n == 0 {
			return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
		}
		// bufの残りを上書き
		copy(moreBuf[copied:], moreBuf[copied:copied+n])
	}

	// 矢印キーの処理
	if moreBuf[0] == '\x1b' && moreBuf[1] == '[' {
		switch moreBuf[2] {
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

	return KeyEvent{Type: KeyEventSpecial, Key: KeyEsc}, nil
}

// processInputBuffer は入力バッファを処理する
func (kr *StandardKeyReader) processInputBuffer(buf []byte) (KeyEvent, error) {
	if len(buf) == 0 {
		return KeyEvent{}, fmt.Errorf("empty input")
	}

	// UTF-8文字のデコード
	r, size := utf8.DecodeRune(buf)
	if r == utf8.RuneError {
		return KeyEvent{}, fmt.Errorf("invalid UTF-8 sequence")
	}

	// 残りのバッファがある場合は保存
	if size < len(buf) {
		kr.inputBuffer = make([]KeyEvent, 0)
		rest := buf[size:]
		for len(rest) > 0 {
			r, size := utf8.DecodeRune(rest)
			if r == utf8.RuneError {
				break
			}
			kr.inputBuffer = append(kr.inputBuffer, KeyEvent{Type: KeyEventChar, Rune: r})
			rest = rest[size:]
		}
	}

	return KeyEvent{Type: KeyEventChar, Rune: r}, nil
}

// InputHandler はキー入力とその処理を管理する
type InputHandler struct {
	keyReader    KeyReader
	editor       EditorOperations
	eventManager *events.EventManager
}

// NewInputHandler は新しいInputHandlerを作成する
func NewInputHandler(editor EditorOperations, eventManager *events.EventManager) *InputHandler {
	return &InputHandler{
		keyReader:    NewStandardKeyReader(),
		editor:       editor,
		eventManager: eventManager,
	}
}

// HandleKeypress はキー入力をイベントとして発行する
func (ih *InputHandler) HandleKeypress() (EditorCommand, error) {
	event, err := ih.keyReader.ReadKey()
	if err != nil {
		return nil, err
	}

	// コマンドを生成する前にイベントを発行
	inputEvent := events.NewInputEvent(events.KeyEventType(event.Type), event.Rune, events.Key(event.Key))
	ih.eventManager.Publish(inputEvent)

	var command EditorCommand
	switch event.Type {
	case KeyEventChar:
		if event.Rune >= 32 && event.Rune != 127 {
			if !ih.handleBracketPair(event.Rune) {
				command = NewInsertCharCommand(ih.editor, event.Rune)
			}
		}
	case KeyEventSpecial:
		command = ih.handleSpecialKey(event.Key)
	case KeyEventControl:
		command = ih.handleControlKey(event.Key)
	}

	return command, nil
}

// createCommandFromEvent はイベントからEditorCommandを生成する
func (ih *InputHandler) createCommandFromEvent(e *events.InputEvent) EditorCommand {
	switch e.KeyType {
	case events.KeyEventChar:
		return NewInsertCharCommand(ih.editor, e.Rune)
	case events.KeyEventSpecial:
		return ih.createSpecialKeyCommand(e.SpecialKey)
	case events.KeyEventControl:
		return ih.createControlKeyCommand(e.SpecialKey)
	}
	return nil
}

// createSpecialKeyCommand は特殊キーに対応するコマンドを生成する
func (ih *InputHandler) createSpecialKeyCommand(key events.Key) EditorCommand {
	switch key {
	case events.KeyTab:
		return ih.createTabCommand()
	case events.KeyShiftTab:
		return ih.createShiftTabCommand()
	case events.KeyArrowUp:
		return NewMoveCursorCommand(ih.editor, CursorUp)
	case events.KeyArrowDown:
		return NewMoveCursorCommand(ih.editor, CursorDown)
	case events.KeyArrowRight:
		return NewMoveCursorCommand(ih.editor, CursorRight)
	case events.KeyArrowLeft:
		return NewMoveCursorCommand(ih.editor, CursorLeft)
	case events.KeyBackspace:
		return NewDeleteCharCommand(ih.editor)
	case events.KeyEnter:
		return NewInsertNewlineCommand(ih.editor)
	}
	return nil
}

// createControlKeyCommand は制御キーに対応するコマンドを生成する
func (ih *InputHandler) createControlKeyCommand(key events.Key) EditorCommand {
	switch key {
	case events.KeyCtrlQ, events.KeyCtrlC:
		return NewQuitCommand(ih.editor) // 同じように終了処理を行う
	case events.KeyCtrlS:
		return NewSaveCommand(ih.editor)
	}
	return nil
}

// createTabCommand はタブコマンドを生成する
func (ih *InputHandler) createTabCommand() EditorCommand {
	tabWidth := ih.editor.GetConfig().TabWidth
	spaces := make([]rune, tabWidth)
	for i := 0; i < tabWidth; i++ {
		spaces[i] = ' '
	}
	return NewInsertCharsCommand(ih.editor, spaces...)
}

// createShiftTabCommand はShift+Tabコマンドを生成する
func (ih *InputHandler) createShiftTabCommand() EditorCommand {
	return NewCompositeCommand(func() error {
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
			if err := NewDeleteCharCommand(ih.editor).Execute(); err != nil {
				return err
			}
		}
		return nil
	})
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

// CompositeCommand は複数のコマンドを1つのコマンドとして扱うための構造体
type CompositeCommand struct {
	execute func() error
}

// NewCompositeCommand は新しいCompositeCommandを作成する
func NewCompositeCommand(execute func() error) *CompositeCommand {
	return &CompositeCommand{execute: execute}
}

// Execute はコマンドを実行する
func (c *CompositeCommand) Execute() error {
	return c.execute()
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
func (ih *InputHandler) handleSpecialKey(key Key) EditorCommand {
	switch key {
	case KeyTab:
		// タブをスペースに変換
		return ih.createTabCommand()
	case KeyShiftTab:
		return ih.createShiftTabCommand()
	case KeyArrowUp:
		return NewMoveCursorCommand(ih.editor, CursorUp)
	case KeyArrowDown:
		return NewMoveCursorCommand(ih.editor, CursorDown)
	case KeyArrowRight:
		return NewMoveCursorCommand(ih.editor, CursorRight)
	case KeyArrowLeft:
		return NewMoveCursorCommand(ih.editor, CursorLeft)
	case KeyBackspace:
		return NewDeleteCharCommand(ih.editor)
	case KeyEnter:
		return NewInsertNewlineCommand(ih.editor)
	}
	return nil
}

// handleControlKey は制御キーの処理を行う
func (ih *InputHandler) handleControlKey(key Key) EditorCommand {
	switch key {
	case KeyCtrlQ, KeyCtrlC:
		return NewQuitCommand(ih.editor) // 同じように終了処理を行う
	case KeyCtrlS:
		return NewSaveCommand(ih.editor)
	}
	return nil
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (ih *InputHandler) SetKeyReader(reader KeyReader) {
	ih.keyReader = reader
}
