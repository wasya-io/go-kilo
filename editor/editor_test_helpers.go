package editor

import (
	"fmt"

	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
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
	GetCursor() cursor.Cursor
	SetStatusMessage(format string, args ...interface{})
	InsertChars(chars []rune)
}

// GetRows は行のコンテンツを文字列のスライスとして返す
// テスト用に実装
func (e *Editor) GetRows() []string {
	return e.buffer.GetAllLines()
}

// isControl は制御文字かどうかを判定する
func isControl(c byte) bool {
	return c < 32 || c == 127
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

// TestEditorOperations はテスト用のEditorOperations実装
type TestEditorOperations struct {
	editor *Editor
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

func (e *TestEditorOperations) GetContent(lineNum int) string {
	return e.editor.GetContent(lineNum)
}

func (e *TestEditorOperations) GetConfig() *config.Config {
	return e.editor.GetConfig()
}
