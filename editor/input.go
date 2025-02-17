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

	// エスケープシーケンスの処理
	if n > 0 && buf[0] == '\x1b' {
		return kr.handleEscapeSequence(buf[:n])
	}

	// バックスペースの処理
	if n == 1 && buf[0] == 127 {
		return KeyEvent{Type: KeyEventSpecial, Key: KeyBackspace}, nil
	}

	// 制御キーの処理
	if n == 1 {
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
	var events []KeyEvent

	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// 不正なUTF-8シーケンスの場合は1バイトスキップ
			data = data[1:]
			continue
		}

		if r != utf8.RuneError {
			events = append(events, KeyEvent{Type: KeyEventChar, Rune: r})
			data = data[size:]
		} else {
			// バッファが不完全なマルチバイト文字を含む場合、追加で読み取り
			tmp := make([]byte, 4)
			n, err = os.Stdin.Read(tmp)
			if err != nil {
				return KeyEvent{}, err
			}
			data = append(data, tmp[:n]...)
		}
	}

	// 複数の文字が入力された場合は、最初の文字を返し、残りはバッファに保存
	if len(events) > 0 {
		// 最初の文字を返し、残りをバッファに保存
		kr.inputBuffer = append(kr.inputBuffer, events[1:]...)
		return events[0], nil
	}

	return KeyEvent{}, fmt.Errorf("no valid input")
}

// handleEscapeSequence はエスケープシーケンスを解析してKeyEventに変換する
func (kr *StandardKeyReader) handleEscapeSequence(buf []byte) (KeyEvent, error) {
	n := len(buf)
	if n < 3 {
		// バッファに3バイト未満しかない場合は追加で読み取り
		seq := make([]byte, 2)
		nseq, err := os.Stdin.Read(seq)
		if err != nil || nseq != 2 {
			return KeyEvent{}, nil
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

	return KeyEvent{}, nil
}
