package editor_test

import (
	"os"
	"testing"

	"github.com/go-kilo/editor"
)

func TestEditor(t *testing.T) {
	// テスト用のエディタを初期化
	ed, err := editor.New(true) // テストモードをtrueに設定
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	// テスト用のファイルを作成
	filename := "testfile.txt"
	content := "日本語入力テスト\nひらがな：あいうえお\n漢字：漢字混じり文\nカタカナ：アイウエオ\n英数字混在：Hello 世界 123!"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("テスト用ファイルの作成に失敗しました: %v", err)
	}
	defer os.Remove(filename)

	// ファイルをエディタで開く
	if err := ed.OpenFile(filename); err != nil {
		t.Fatalf("ファイルの読み込みに失敗しました: %v", err)
	}

	t.Run("ファイル読み込みのテスト", func(t *testing.T) {
		// ファイル内容の検証
		rows := ed.GetRows() // エクスポートされたメソッドを使用
		if len(rows) != 5 {
			t.Errorf("行数が正しくありません: got %d, want %d", len(rows), 5)
		}

		// 各行の内容を検証
		expectedLines := []string{
			"日本語入力テスト",
			"ひらがな：あいうえお",
			"漢字：漢字混じり文",
			"カタカナ：アイウエオ",
			"英数字混在：Hello 世界 123!",
		}
		for i, want := range expectedLines {
			if got := rows[i]; got != want {
				t.Errorf("行の内容が正しくありません: got %s, want %s", got, want)
			}
		}
	})
}

// TestMultibyteInput はマルチバイト文字の入力処理をテストする
func TestMultibyteInput(t *testing.T) {
	ed, err := editor.New(true)
	if err != nil {
		t.Fatalf("エディタの初期化に失敗しました: %v", err)
	}
	defer ed.Cleanup()

	t.Run("マルチバイト文字の入力テスト", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "ひらがな",
				input:    "あいうえお",
				expected: "あいうえお",
			},
			{
				name:     "漢字",
				input:    "漢字",
				expected: "漢字",
			},
			{
				name:     "カタカナ",
				input:    "カタカナ",
				expected: "カタカナ",
			},
			{
				name:     "混合文字",
				input:    "Hello世界123",
				expected: "Hello世界123",
			},
			{
				name:     "絵文字",
				input:    "😀🌟",
				expected: "😀🌟",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// テスト用の一時ファイルを作成
				filename := "test_input.txt"
				if err := os.WriteFile(filename, []byte(""), 0644); err != nil {
					t.Fatalf("テストファイルの作成に失敗しました: %v", err)
				}
				defer os.Remove(filename)

				// ファイルを開く
				if err := ed.OpenFile(filename); err != nil {
					t.Fatalf("ファイルを開けませんでした: %v", err)
				}

				// 文字を1文字ずつ入力
				for _, r := range tt.input {
					if err := ed.TestInput(r); err != nil {
						t.Fatalf("文字の入力に失敗しました: %v", err)
					}
				}

				// 結果を検証
				rows := ed.GetRows()
				if len(rows) == 0 {
					t.Fatal("テキストが空です")
				}
				if got := rows[0]; got != tt.expected {
					t.Errorf("入力結果が一致しません: got %q, want %q", got, tt.expected)
				}
			})
		}
	})
}
