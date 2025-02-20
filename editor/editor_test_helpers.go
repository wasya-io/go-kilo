package editor

import (
	"fmt"
	"unicode/utf8"
)

// KeyReader はキー入力を読み取るインターフェース
type KeyReader interface {
	ReadKey() (KeyEvent, error)
}

// CursorMovement はカーソル移動の種類を表す型
type CursorMovement byte

const (
	CursorUp    CursorMovement = 'A'
	CursorDown  CursorMovement = 'B'
	CursorRight CursorMovement = 'C'
	CursorLeft  CursorMovement = 'D'
)

// EditorTestHelper はテスト用のヘルパーメソッドを提供する
type EditorTestHelper interface {
	TestSetCursor(x, y int) error
	TestGetCursor() (x, y int)
	TestMoveCursor(movement CursorMovement) error
	TestInput(ch rune) error
	TestDelete() error
	GetRows() []string
	GetContent(lineNum int) string
	GetCharAtCursor() string
	GetLineCount() int
	SetKeyReader(reader KeyReader)
	IsDirty() bool
	SetDirty(bool)
	UpdateScroll()
	GetConfig() *Config
	GetCursor() Cursor
	SetStatusMessage(format string, args ...interface{})
	InsertChars(chars []rune)
}

// GetRows は行のコンテンツを文字列のスライスとして返す
// テスト用に実装
func (e *Editor) GetRows() []string {
	return e.buffer.GetAllContent()
}

// SetKeyReader はキー入力読み取りインターフェースを設定する
func (e *Editor) SetKeyReader(reader KeyReader) {
	if e.input != nil {
		e.input.SetKeyReader(reader)
	}
}

// TestInput はテスト用に1文字入力をシミュレートする
func (e *Editor) TestInput(ch rune) error {
	e.buffer.InsertChar(ch)
	return nil
}

// TestSetCursor はテスト用にカーソル位置を設定する
func (e *Editor) TestSetCursor(x, y int) error {
	if y >= e.buffer.GetLineCount() {
		return fmt.Errorf("invalid y position: %d", y)
	}
	e.buffer.SetCursor(x, y)
	return nil
}

// TestGetCursor はテスト用にカーソル位置を取得する
func (e *Editor) TestGetCursor() (x, y int) {
	return e.buffer.GetCursorXY()
}

// TestMoveCursor はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursor(m CursorMovement) error {
	e.buffer.MoveCursor(m)
	return nil
}

// isControl は制御文字かどうかを判定する
func isControl(c byte) bool {
	return c < 32 || c == 127
}

// TestProcessInput はテスト用にキー入力をシミュレートする
func (e *Editor) TestProcessInput(input []byte) error {
	if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
		switch input[2] {
		case 'A': // 上矢印
			e.buffer.MoveCursor(CursorUp)
		case 'B': // 下矢印
			e.buffer.MoveCursor(CursorDown)
		case 'C': // 右矢印
			e.buffer.MoveCursor(CursorRight)
		case 'D': // 左矢印
			e.buffer.MoveCursor(CursorLeft)
		}
		return nil
	}

	// バックスペースの処理
	if len(input) == 1 && input[0] == 127 {
		e.buffer.DeleteChar()
		return nil
	}

	// 通常の文字入力処理
	if len(input) == 1 {
		if !isControl(input[0]) {
			e.buffer.InsertChar(rune(input[0]))
		}
	} else {
		// マルチバイト文字の処理
		r, _ := utf8.DecodeRune(input)
		if r != utf8.RuneError {
			e.buffer.InsertChar(r)
		}
	}

	return nil
}

// TestDelete はテスト用にバックスペースを実行する
func (e *Editor) TestDelete() error {
	e.buffer.DeleteChar()
	return nil
}

// SetRowsForTest はテスト用に行データを直接設定する
func (e *Editor) SetRowsForTest(rows []*Row) {
	content := make([]string, len(rows))
	for i, row := range rows {
		content[i] = row.GetContent()
	}
	e.buffer.LoadContent(content)
}

// TestMoveCursorByByte はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursorByByte(direction byte) error {
	switch direction {
	case 'A': // Up
		e.buffer.MoveCursor(CursorUp)
	case 'B': // Down
		e.buffer.MoveCursor(CursorDown)
	case 'C': // Right
		e.buffer.MoveCursor(CursorRight)
	case 'D': // Left
		e.buffer.MoveCursor(CursorLeft)
	default:
		return fmt.Errorf("unknown direction: %c", direction)
	}
	return nil
}

// GetCharAtCursor は現在のカーソル位置の文字を返す
func (e *Editor) GetCharAtCursor() string {
	return e.buffer.GetCharAtCursor()
}

// GetLineCount は行数を返す
func (e *Editor) GetLineCount() int {
	return e.buffer.GetLineCount()
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
	if content, exists := ms.files[filename]; exists {
		return content, nil
	}
	return []string{}, nil
}

// Save はモックストレージにファイルの内容を保存します
func (ms *MockStorage) Save(filename string, content []string) error {
	ms.files[filename] = content
	return nil
}

// GetSavedContent は保存された内容を取得します（テスト用）
func (ms *MockStorage) GetSavedContent(filename string) []string {
	return ms.files[filename]
}

// MockKeyReader はテスト用のキー入力シミュレータ
type MockKeyReader struct {
	events []KeyEvent
	index  int
}

// NewMockKeyReader は一連のキーイベントをシミュレートするMockKeyReaderを作成する
func NewMockKeyReader(events []KeyEvent) *MockKeyReader {
	return &MockKeyReader{
		events: events,
		index:  0,
	}
}

// ReadKey は事前に設定されたキーイベントを順番に返す
func (m *MockKeyReader) ReadKey() (KeyEvent, error) {
	if m.index >= len(m.events) {
		return KeyEvent{}, nil
	}
	event := m.events[m.index]
	m.index++
	return event, nil
}

// ResetIndex はイベントインデックスをリセットする
func (m *MockKeyReader) ResetIndex() {
	m.index = 0
}

// TestEditorOperations はテスト用のEditorOperations実装
type TestEditorOperations struct {
	editor *Editor
}

// EditorOperationsインターフェースの実装
func (e *TestEditorOperations) InsertChar(ch rune) {
	e.editor.InsertChar(ch)
}

func (e *TestEditorOperations) InsertChars(chars []rune) {
	e.editor.InsertChars(chars)
}

func (e *TestEditorOperations) DeleteChar() {
	e.editor.DeleteChar()
}

func (e *TestEditorOperations) InsertNewline() {
	e.editor.InsertNewline()
}

func (e *TestEditorOperations) MoveCursor(movement CursorMovement) {
	e.editor.MoveCursor(movement)
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
	return e.editor.GetCursor()
}

func (e *TestEditorOperations) GetContent(lineNum int) string {
	return e.editor.GetContent(lineNum)
}

func (e *TestEditorOperations) GetConfig() *Config {
	return e.editor.GetConfig()
}
