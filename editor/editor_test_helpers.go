package editor

import (
	"fmt"
	"unicode/utf8"

	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
)

// KeyReader はキー入力を読み取るインターフェース
// type KeyReader interface {
// 	ReadKey() (KeyEvent, []KeyEvent, error)
// }

// CursorMovement はカーソル移動の種類を表す型
type CursorMovement byte

const (
	CursorUp       CursorMovement = 'A'
	CursorDown     CursorMovement = 'B'
	CursorRight    CursorMovement = 'C'
	CursorLeft     CursorMovement = 'D'
	MouseWheelUp   CursorMovement = 'U' // マウスホイールでの上方向スクロール
	MouseWheelDown CursorMovement = 'V' // マウスホイールでの下方向スクロール
)

// EditorTestHelper はテスト用のヘルパーメソッドを提供する
type EditorTestHelper interface {
	TestSetCursor(x, y int) error
	TestGetCursor() (x, y int)
	TestMoveCursor(movement CursorMovement) error
	TestInput(ch rune) error
	TestDelete() error
	GetRows() []string
	GetContentForTest(lineNum int) string
	GetCharAtCursor() string
	GetLineCount() int
	// SetKeyReader(reader KeyReader)
	IsDirty() bool
	SetDirty(bool)
	UpdateScroll()
	GetConfig() *config.Config
	GetCursor() Cursor
	SetStatusMessage(format string, args ...interface{})
	InsertChars(chars []rune)
}

// GetRows は行のコンテンツを文字列のスライスとして返す
// テスト用に実装
func (e *Editor) GetRows() []string {
	return e.buffer.GetAllLines()
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
// func (e *Editor) SetKeyReader(reader KeyReader) {
// 	if e.input != nil {
// 		e.input.keyReader = reader
// 	}
// }

// TestInput はテスト用に1文字入力をシミュレートする
func (e *Editor) TestInput(ch rune) error {
	pos := e.ui.GetCursor()
	e.buffer.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, ch)
	// カーソルを1つ進める
	e.ui.SetCursor(pos.X+1, pos.Y)
	return nil
}

// TestSetCursor はテスト用にカーソル位置を設定する
func (e *Editor) TestSetCursor(x, y int) error {
	if y >= e.buffer.GetLineCount() {
		return fmt.Errorf("invalid cursor position: line %d is out of range", y)
	}
	e.ui.SetCursor(x, y)
	return nil
}

// TestGetCursor はテスト用にカーソル位置を取得する
func (e *Editor) TestGetCursor() (x, y int) {
	pos := e.ui.GetCursor()
	return pos.X, pos.Y
}

// TestMoveCursor はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursor(m CursorMovement) error {
	e.ui.MoveCursor(m, e.buffer)
	return nil
}

// isControl は制御文字かどうかを判定する
func isControl(c byte) bool {
	return c < 32 || c == 127
}

// TestProcessInput はテスト用にキー入力をシミュレートする
func (e *Editor) TestProcessInput(input []byte) error {
	pos := e.ui.GetCursor()

	if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
		// ESC [ で始まるシーケンス
		return e.TestMoveCursorByByte(input[2])
	}

	// バックスペースの処理
	if len(input) == 1 && input[0] == 127 {
		e.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
		return nil
	}

	// 通常の文字入力処理
	if len(input) == 1 {
		e.buffer.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, rune(input[0]))
		e.ui.SetCursor(pos.X+1, pos.Y)
	} else {
		// マルチバイト文字の処理
		r, _ := utf8.DecodeRune(input)
		if r != utf8.RuneError {
			e.buffer.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, r)
			e.ui.SetCursor(pos.X+1, pos.Y)
		}
	}

	return nil
}

// TestDelete はテスト用にバックスペースを実行する
func (e *Editor) TestDelete() error {
	pos := e.ui.GetCursor()

	if pos.X > 0 {
		// 行の途中での削除
		e.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
		e.ui.SetCursor(pos.X-1, pos.Y) // カーソルを1つ左に移動
	} else if pos.Y > 0 {
		// 行頭での削除（前の行との結合）
		prevRow := e.buffer.GetRow(pos.Y - 1)
		if prevRow != nil {
			targetX := prevRow.GetRuneCount() // 前の行の末尾位置
			e.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
			e.ui.SetCursor(targetX, pos.Y-1) // 前の行の末尾へ移動
		}
	}

	return nil
}

// SetRowsForTest はテスト用に行データを直接設定する
func (e *Editor) SetRowsForTest(rows []*contents.Row) {
	content := make([]string, len(rows))
	for i, row := range rows {
		if row != nil {
			content[i] = row.GetContent()
		} else {
			content[i] = ""
		}
	}
	e.buffer.LoadContent(content)
	// カーソル位置を先頭に戻す
	e.ui.SetCursor(0, 0)
}

// TestMoveCursorByByte はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursorByByte(direction byte) error {
	switch direction {
	case 'A': // Up
		e.ui.MoveCursor(CursorUp, e.buffer)
	case 'B': // Down
		e.ui.MoveCursor(CursorDown, e.buffer)
	case 'C': // Right
		e.ui.MoveCursor(CursorRight, e.buffer)
	case 'D': // Left
		e.ui.MoveCursor(CursorLeft, e.buffer)
	default:
		return fmt.Errorf("invalid direction byte: %c", direction)
	}
	return nil
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (e *Editor) GetCharAtCursor() string {
	pos := e.ui.GetCursor()
	row := e.buffer.GetRow(pos.Y)
	if row == nil {
		return ""
	}

	r, ok := row.GetRuneAt(pos.X)
	if !ok {
		return ""
	}
	return string(r)
}

// GetLineCount は行数を返す
func (e *Editor) GetLineCount() int {
	return e.buffer.GetLineCount()
}

// GetContentForTest は指定行の内容を返す（テスト用）
func (e *Editor) GetContentForTest(lineNum int) string {
	return e.buffer.GetContentLine(lineNum)
}

// MockStorage はテスト用のストレージモックです
type MockStorage struct {
	files map[string][]string
}

// NewMockStorage は新しいモックストレージを作成します
func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string][]string),
	}
}

// Load はモックされたファイルの内容を返します
func (ms *MockStorage) Load(filename string) ([]string, error) {
	if content, ok := ms.files[filename]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// Save はモックストレージにファイルの内容を保存します
func (ms *MockStorage) Save(filename string, content []string) error {
	ms.files[filename] = make([]string, len(content))
	copy(ms.files[filename], content)
	return nil
}

// GetSavedContent は保存された内容を取得します（テスト用）
func (ms *MockStorage) GetSavedContent(filename string) []string {
	if content, ok := ms.files[filename]; ok {
		return content
	}
	return nil
}

// // MockKeyReader はテスト用のキー入力シミュレータ
// type MockKeyReader struct {
// 	events []KeyEvent
// 	index  int
// }

// NewMockKeyReader は一連のキーイベントをシミュレートするMockKeyReaderを作成する
// func NewMockKeyReader(events []KeyEvent) *MockKeyReader {
// 	return &MockKeyReader{
// 		events: events,
// 		index:  0,
// 	}
// }

// // ReadKey は事前に設定されたキーイベントを順番に返す
// func (m *MockKeyReader) ReadKey() (KeyEvent, []KeyEvent, error) {
// 	if m.index >= len(m.events) {
// 		return KeyEvent{}, nil, fmt.Errorf("no more events")
// 	}

// 	event := m.events[m.index]
// 	m.index++
// 	return event, nil, nil
// }

// // ResetIndex はイベントインデックスをリセットする
// func (m *MockKeyReader) ResetIndex() {
// 	m.index = 0
// }

// TestEditorOperations はテスト用のEditorOperations実装
type TestEditorOperations struct {
	editor *Editor
}

// EditorOperationsインターフェースの実装
func (e *TestEditorOperations) InsertChar(ch rune) {
	pos := e.editor.ui.GetCursor()
	e.editor.buffer.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, ch)
	e.editor.ui.SetCursor(pos.X+1, pos.Y)
}

func (e *TestEditorOperations) InsertChars(chars []rune) {
	pos := e.editor.ui.GetCursor()
	e.editor.buffer.InsertChars(contents.Position{X: pos.X, Y: pos.Y}, chars)
	e.editor.ui.SetCursor(pos.X+len(chars), pos.Y)
}

func (e *TestEditorOperations) DeleteChar() {
	pos := e.editor.ui.GetCursor()
	e.editor.buffer.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
}

func (e *TestEditorOperations) InsertNewline() {
	pos := e.editor.ui.GetCursor()
	e.editor.buffer.InsertNewline(contents.Position{X: pos.X, Y: pos.Y})
	e.editor.ui.HandleNewLine()
}

func (e *TestEditorOperations) MoveCursor(movement CursorMovement) {
	e.editor.ui.MoveCursor(movement, e.editor.buffer)
}

func (e *TestEditorOperations) SaveFile() error {
	return e.editor.SaveFile()
}

func (e *TestEditorOperations) Quit() {
	e.editor.Quit()
}

func (e *TestEditorOperations) IsDirty() bool {
	return e.editor.IsDirty()
}

func (e *TestEditorOperations) SetDirty(dirty bool) {
	e.editor.SetDirty(dirty)
}

func (e *TestEditorOperations) SetStatusMessage(format string, args ...interface{}) {
	e.editor.SetStatusMessage(format, args...)
}

func (e *TestEditorOperations) UpdateScroll() {
	e.editor.UpdateScroll()
}

func (e *TestEditorOperations) GetCursor() Cursor {
	return e.editor.ui.GetCursor()
}

func (e *TestEditorOperations) GetContent(lineNum int) string {
	return e.editor.GetContent(lineNum)
}

func (e *TestEditorOperations) GetConfig() *config.Config {
	return e.editor.GetConfig()
}
