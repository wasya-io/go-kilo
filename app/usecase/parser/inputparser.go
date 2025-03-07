package parser

import (
	"fmt"
	"unicode/utf8"

	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/key"
)

type StandardInputParser struct {
	logger core.Logger
}

type InputParser interface {
	Parse(buf []byte, n int) ([]key.KeyEvent, error)
}

func NewStandardInputParser(logger core.Logger) *StandardInputParser {
	return &StandardInputParser{
		logger: logger,
	}
}

// Parse はバイトデータを解析してキーイベントを返す
func (p *StandardInputParser) Parse(buf []byte, n int) ([]key.KeyEvent, error) {
	// コントロールキーの処理
	if event, ok := p.parseControlKey(buf[0]); ok {
		return []key.KeyEvent{event}, nil
	}

	// 特殊キーの処理
	if event, ok := p.parseSpecialKey(buf[0]); ok {
		return []key.KeyEvent{event}, nil
	}

	// エスケープシーケンスの処理
	if buf[0] == '\x1b' {
		event, err := p.parseEscapeSequence(buf, n)
		if err == nil {
			return []key.KeyEvent{event}, nil
		}
	}

	// 文字の処理（UTF-8とASCII）
	return p.parseCharacter(buf, n)
}

// parseControlKey はコントロールキーの解析を行う
func (p *StandardInputParser) parseControlKey(b byte) (key.KeyEvent, bool) {
	switch b {
	case 3: // Ctrl+C
		return key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlC}, true
	case 17: // Ctrl+Q
		return key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlQ}, true
	case 19: // Ctrl-S
		return key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlS}, true
	}
	return key.KeyEvent{}, false
}

// parseSpecialKey は特殊キーの解析を行う
func (p *StandardInputParser) parseSpecialKey(b byte) (key.KeyEvent, bool) {
	switch b {
	case 127: // Backspace
		return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyBackspace}, true
	case '\r': // Enter
		return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyEnter}, true
	case '\t': // Tab
		return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyTab}, true
	}
	return key.KeyEvent{}, false
}

// parseEscapeSequence はエスケープシーケンスの解析を行う
func (p *StandardInputParser) parseEscapeSequence(buf []byte, n int) (key.KeyEvent, error) {
	if n == 1 {
		return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyEsc}, nil
	}

	if n >= 3 && buf[1] == '[' {
		switch buf[2] {
		case 'A':
			return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowUp}, nil
		case 'B':
			return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowDown}, nil
		case 'C':
			return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowRight}, nil
		case 'D':
			return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyArrowLeft}, nil
		case 'Z':
			return key.KeyEvent{Type: key.KeyEventSpecial, Key: key.KeyShiftTab}, nil
		case 'M', '<':
			return p.parseMouseEvent(buf, n)
		}
	}

	return key.KeyEvent{}, fmt.Errorf("unknown escape sequence")
}

// parseMouseEvent はマウスイベントの解析を行う
func (p *StandardInputParser) parseMouseEvent(buf []byte, n int) (key.KeyEvent, error) {
	if n >= 6 && buf[2] == '<' {
		var cb, cx, cy int
		if _, err := fmt.Sscanf(string(buf[3:n]), "%d;%d;%d", &cb, &cx, &cy); err == nil {
			switch cb {
			case 64: // スクロールアップ
				return key.KeyEvent{
					Type:        key.KeyEventMouse,
					Key:         key.KeyMouseWheel,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: key.MouseScrollUp,
				}, nil
			case 65: // スクロールダウン
				return key.KeyEvent{
					Type:        key.KeyEventMouse,
					Key:         key.KeyMouseWheel,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: key.MouseScrollDown,
				}, nil
			case 0: // 左クリック
				return key.KeyEvent{
					Type:        key.KeyEventMouse,
					Key:         key.KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: key.MouseLeftClick,
				}, nil
			case 2: // 右クリック
				return key.KeyEvent{
					Type:        key.KeyEventMouse,
					Key:         key.KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: key.MouseRightClick,
				}, nil
			case 1: // 中クリック
				return key.KeyEvent{
					Type:        key.KeyEventMouse,
					Key:         key.KeyMouseClick,
					MouseRow:    cy - 1,
					MouseCol:    cx - 1,
					MouseAction: key.MouseMiddleClick,
				}, nil
			}
		}
	}
	return key.KeyEvent{}, fmt.Errorf("unknown mouse event")
}

// parseCharacter はUTF-8/ASCII文字の解析を行う
func (p *StandardInputParser) parseCharacter(buf []byte, n int) ([]key.KeyEvent, error) {
	// UTF-8文字の処理
	if (buf[0] & 0x80) != 0 {
		r, size := utf8.DecodeRune(buf[:n])
		if r != utf8.RuneError {
			var allEvents []key.KeyEvent
			allEvents = append(allEvents, key.KeyEvent{Type: key.KeyEventChar, Rune: r})
			if n > size {
				remainingBytes := make([]byte, n-size)
				copy(remainingBytes, buf[size:n])
				for len(remainingBytes) > 0 {
					r, s := utf8.DecodeRune(remainingBytes)
					if r == utf8.RuneError {
						break
					}
					allEvents = append(allEvents, key.KeyEvent{Type: key.KeyEventChar, Rune: r})
					remainingBytes = remainingBytes[s:]
				}
			}
			return allEvents, nil
		}
	}

	// ASCII文字の処理
	if buf[0] >= 32 && buf[0] < 127 {
		return []key.KeyEvent{{Type: key.KeyEventChar, Rune: rune(buf[0])}}, nil
	}

	return nil, fmt.Errorf("unknown input")
}
