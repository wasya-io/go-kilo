package editor

import (
	"fmt"
	"unicode/utf8"
)

// CursorMovement はカーソル移動の種類を表す型
type CursorMovement byte

const (
	CursorUp    CursorMovement = 'A'
	CursorDown  CursorMovement = 'B'
	CursorRight CursorMovement = 'C'
	CursorLeft  CursorMovement = 'D'
)

// GetRows は行のコンテンツを文字列のスライスとして返す
// テスト用に実装
func (e *Editor) GetRows() []string {
	result := make([]string, len(e.rows))
	for i, row := range e.rows {
		result[i] = row.chars
	}
	return result
}

// TestInput はテスト用に1文字入力をシミュレートする
func (e *Editor) TestInput(r rune) error {
	e.insertChar(r)
	return nil
}

// TestSetCursor はテスト用にカーソル位置を設定する
func (e *Editor) TestSetCursor(x, y int) error {
	if y >= len(e.rows) {
		return fmt.Errorf("invalid y position: %d", y)
	}
	if y >= 0 && x >= 0 {
		e.cy = y
		runes := []rune(e.rows[y].chars)
		if x > len(runes) {
			x = len(runes)
		}
		e.cx = x
	}
	return nil
}

// TestGetCursor はテスト用にカーソル位置を取得する
func (e *Editor) TestGetCursor() (x, y int) {
	return e.cx, e.cy
}

// TestMoveCursor はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursor(m CursorMovement) error {
	switch m {
	case CursorUp:
		if e.cy > 0 {
			e.cy--
			e.adjustCursorX()
		}
	case CursorDown:
		if e.cy < len(e.rows)-1 {
			e.cy++
			e.adjustCursorX()
		}
	case CursorRight:
		if e.cy < len(e.rows) {
			runes := []rune(e.rows[e.cy].chars)
			if e.cx < len(runes) {
				e.cx++
			}
		}
	case CursorLeft:
		if e.cx > 0 {
			e.cx--
		}
	}
	return nil
}

// TestProcessInput はテスト用にキー入力をシミュレートする
func (e *Editor) TestProcessInput(input []byte) error {
	if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
		switch input[2] {
		case 'A': // 上矢印
			if e.cy > 0 {
				e.cy--
				e.adjustCursorX()
			}
		case 'B': // 下矢印
			if e.cy < len(e.rows)-1 {
				e.cy++
				e.adjustCursorX()
			}
		case 'C': // 右矢印
			if e.cy < len(e.rows) {
				runes := []rune(e.rows[e.cy].chars)
				if e.cx < len(runes) {
					e.cx++
				} else if e.cy < len(e.rows)-1 {
					e.cy++
					e.cx = 0
				}
			}
		case 'D': // 左矢印
			if e.cx > 0 {
				e.cx--
			} else if e.cy > 0 {
				e.cy--
				runes := []rune(e.rows[e.cy].chars)
				e.cx = len(runes)
			}
		}
		return nil
	}

	// バックスペースの処理
	if len(input) == 1 && input[0] == 127 {
		e.deleteChar()
		return nil
	}

	// 通常の文字入力処理
	if len(input) == 1 {
		if !iscntrl(input[0]) {
			e.insertChar(rune(input[0]))
		}
	} else {
		// マルチバイト文字の処理
		r, _ := utf8.DecodeRune(input)
		if r != utf8.RuneError {
			e.insertChar(r)
		}
	}

	return nil
}

// TestDelete はテスト用にバックスペースを実行する
func (e *Editor) TestDelete() error {
	e.deleteChar()
	return nil
}

// TestMoveCursorByByte はテスト用にカーソルを移動する
func (e *Editor) TestMoveCursorByByte(direction byte) error {
	switch direction {
	case 'A': // Up
		if e.cy > 0 {
			e.cy--
			e.adjustCursorX()
		}
	case 'B': // Down
		if e.cy < len(e.rows)-1 {
			e.cy++
			e.adjustCursorX()
		}
	case 'C': // Right
		if e.cy < len(e.rows) {
			runes := []rune(e.rows[e.cy].chars)
			if e.cx < len(runes) {
				e.cx++
			} else if e.cy < len(e.rows)-1 {
				e.cy++
				e.cx = 0
			}
		}
	case 'D': // Left
		if e.cx > 0 {
			e.cx--
		} else if e.cy > 0 {
			e.cy--
			runes := []rune(e.rows[e.cy].chars)
			e.cx = len(runes)
		}
	default:
		return fmt.Errorf("unknown direction: %c", direction)
	}
	return nil
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
