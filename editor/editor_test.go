package editor_test

import (
	"os"
	"testing"

	"github.com/go-kilo/editor"
)

// テスト用のファイルを準備する
func setupTestFile(t *testing.T, content string) (string, func()) {
	t.Helper()
	filename := "test_file.txt"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗しました: %v", err)
	}
	cleanup := func() {
		os.Remove(filename)
	}
	return filename, cleanup
}

// TestEditor はエディタの基本機能をテストする
func TestEditor(t *testing.T) {
	ed, err := editor.New(true)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	filename, cleanup := setupTestFile(t, "first\nsecond")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	// 初期状態の確認
	t.Run("初期状態", func(t *testing.T) {
		x, y := ed.TestGetCursor()
		if x != 0 || y != 0 {
			t.Errorf("初期カーソル位置が正しくありません: got (%d,%d), want (0,0)", x, y)
		}

		rows := ed.GetRows()
		if len(rows) != 2 {
			t.Errorf("行数が正しくありません: got %d, want 2", len(rows))
		}
	})

	// カーソル位置の設定
	t.Run("カーソル位置の設定", func(t *testing.T) {
		if err := ed.TestSetCursor(2, 0); err != nil {
			t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
		}

		x, y := ed.TestGetCursor()
		if x != 2 || y != 0 {
			t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (2,0)", x, y)
		}
	})
}

// TestCursorMovement はカーソル移動をテストする
func TestCursorMovement(t *testing.T) {
	ed, err := editor.New(true)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	filename, cleanup := setupTestFile(t, "first\nsecond")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name     string
		dir      byte
		wantX    int
		wantY    int
		initialX int
		initialY int
	}{
		{
			name:     "下に移動",
			dir:      'B',
			initialX: 0,
			initialY: 0,
			wantX:    0,
			wantY:    1,
		},
		{
			name:     "右に移動",
			dir:      'C',
			initialX: 0,
			initialY: 0,
			wantX:    1,
			wantY:    0,
		},
		{
			name:     "上に移動",
			dir:      'A',
			initialX: 0,
			initialY: 1,
			wantX:    0,
			wantY:    0,
		},
		{
			name:     "左に移動",
			dir:      'D',
			initialX: 1,
			initialY: 0,
			wantX:    0,
			wantY:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ed.TestSetCursor(tt.initialX, tt.initialY); err != nil {
				t.Fatalf("カーソル位置の初期化に失敗しました: %v", err)
			}

			if err := ed.TestMoveCursor(editor.CursorMovement(tt.dir)); err != nil {
				t.Fatalf("カーソル移動に失敗しました: %v", err)
			}

			x, y := ed.TestGetCursor()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (%d,%d)",
					x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

// TestInput は文字入力をテストする
func TestInput(t *testing.T) {
	ed, err := editor.New(true)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	filename, cleanup := setupTestFile(t, "")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	tests := []struct {
		name  string
		input rune
		want  string
		wantX int
		wantY int
	}{
		{
			name:  "ASCII文字",
			input: 'a',
			want:  "a",
			wantX: 1,
			wantY: 0,
		},
		{
			name:  "日本語文字",
			input: 'あ',
			want:  "aあ",
			wantX: 2,
			wantY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ed.TestInput(tt.input); err != nil {
				t.Fatalf("文字の入力に失敗しました: %v", err)
			}

			rows := ed.GetRows()
			if len(rows) == 0 || rows[0] != tt.want {
				t.Errorf("入力結果が正しくありません: got %q, want %q", rows[0], tt.want)
			}

			x, y := ed.TestGetCursor()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (%d,%d)",
					x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

// TestDelete はバックスペースをテストする
func TestDelete(t *testing.T) {
	ed, err := editor.New(true)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	filename, cleanup := setupTestFile(t, "abc")
	defer cleanup()

	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルを開けませんでした: %v", err)
	}

	if err := ed.TestSetCursor(3, 0); err != nil {
		t.Fatalf("カーソル位置の設定に失敗しました: %v", err)
	}

	// バックスペースを実行
	if err := ed.TestDelete(); err != nil {
		t.Fatalf("バックスペースの処理に失敗しました: %v", err)
	}

	// 結果を検証
	rows := ed.GetRows()
	if len(rows) == 0 || rows[0] != "ab" {
		t.Errorf("削除結果が正しくありません: got %q, want %q", rows[0], "ab")
	}

	x, y := ed.TestGetCursor()
	if x != 2 || y != 0 {
		t.Errorf("カーソル位置が正しくありません: got (%d,%d), want (2,0)", x, y)
	}
}

func TestEditorKeyInput(t *testing.T) {
	// テストケース用のキーイベントを準備
	events := []editor.KeyEvent{
		{Type: editor.KeyEventChar, Rune: 'H'},
		{Type: editor.KeyEventChar, Rune: 'i'},
		{Type: editor.KeyEventSpecial, Key: editor.KeyEnter},
		{Type: editor.KeyEventChar, Rune: '!'},
	}

	// MockKeyReaderを使用してエディタを初期化
	e, err := editor.New(true) // テストモードで初期化
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}
	mockReader := editor.NewMockKeyReader(events)
	e.SetKeyReader(mockReader)

	// キー入力をシミュレート
	for i := 0; i < len(events); i++ {
		err := e.ProcessKeypress()
		if err != nil {
			t.Errorf("Failed to process keypress: %v", err)
		}
	}

	// 期待される結果を検証
	expectedLines := []string{
		"Hi",
		"!",
	}

	lineCount := e.GetLineCount()
	if lineCount != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), lineCount)
	}

	for i, expected := range expectedLines {
		if i >= lineCount {
			break
		}
		if content := e.GetContent(i); content != expected {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected, content)
		}
	}
}
