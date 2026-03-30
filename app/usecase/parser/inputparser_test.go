package parser

import (
	"testing"

	"github.com/wasya-io/go-kilo/app/boundary/logger"
	"github.com/wasya-io/go-kilo/app/entity/key"
)

func TestStandardInputParser_Parse(t *testing.T) {
	logger := logger.New(true)

	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte{0x1b, 0x5b, 0x41}          // キーのエスケープシーケンス
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type != key.KeyEventSpecial || events[0].Key != key.KeyArrowUp {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseControlKey(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte{0x03}                      // Ctrl+Cのコントロールキー
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type != key.KeyEventControl || events[0].Key != key.KeyCtrlC {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseSpecialKey(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte{0x0d}                      // Enterの特殊キー
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type != key.KeyEventSpecial || events[0].Key != key.KeyEnter {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseCharacter(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte{'a'}                       // 文字のバイトデータ
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Type != key.KeyEventChar || events[0].Rune != 'a' {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseCharacterUTF8(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	// buf := []byte{0xe3, 0x81, 0x82}    // UTF-8の文字のバイトデータ
	buf := []byte("あ")
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	a := string([]rune{events[0].Rune})
	if len(events) != 1 || events[0].Type != key.KeyEventChar || a != "あ" {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseCharacterASCII(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte{'a'}                       // ASCIIの文字のバイトデータ
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	a := string([]rune{events[0].Rune})
	if len(events) != 1 || events[0].Type != key.KeyEventChar || a != "a" {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseJapaneseString(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger) // テスト対象のインスタンスを生成
	buf := []byte("あいうえお")                   // 日本語文字列のバイトデータ
	n := len(buf)
	events, err := parser.Parse(buf, n) // テスト対象のメソッドを実行
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	a := string([]rune{events[0].Rune})
	i := string([]rune{events[1].Rune})
	u := string([]rune{events[2].Rune})
	e := string([]rune{events[3].Rune})
	o := string([]rune{events[4].Rune})
	if len(events) != 5 || events[0].Type != key.KeyEventChar || a != "あ" || i != "い" || u != "う" || e != "え" || o != "お" {
		t.Errorf("unexpected event: %v", events)
	}
}

func TestStandardInputParser_ParseLargeJapaneseString(t *testing.T) {
	logger := logger.New(true)
	parser := NewStandardInputParser(logger)
	
	// 25文字の日本語（75バイト -> 32バイト制限を超える）
	inputStr := "アイウエオアイウエオアイウエオアイウエオアイウエオ"
	buf := []byte(inputStr)
	n := len(buf)
	events, err := parser.Parse(buf, n)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	if len(events) != 25 {
		t.Errorf("expected 25 events, got %d", len(events))
	}
	
	// 先頭、中間、末尾などのサンプリングで文字を確認
	if len(events) > 0 && events[0].Rune != 'ア' {
		t.Errorf("expected first char to be 'ア', got %c", events[0].Rune)
	}
	if len(events) >= 25 && events[24].Rune != 'オ' {
		t.Errorf("expected last char to be 'オ', got %c", events[24].Rune)
	}
}
