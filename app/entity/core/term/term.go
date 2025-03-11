package term

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalState は端末の元の状態を保持する構造体
type TerminalState struct {
	origTermios *unix.Termios
}

var globalTermState *TerminalState

// InitTerminal は端末の初期化を行う
func InitTerminal() error {
	// パニックハンドラを設定
	defer func() {
		if r := recover(); r != nil {
			// パニック時に必ず端末状態を復元
			if globalTermState != nil {
				globalTermState.DisableRawMode()
			}
			panic(r) // 元のパニックを再スロー
		}
	}()

	// 現在の端末設定を取得
	term, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return err
	}

	// 端末をCブレーク（raw-ish）モードに設定
	term.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	term.Oflag &^= unix.OPOST
	term.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	term.Cflag &^= unix.CSIZE | unix.PARENB
	term.Cflag |= unix.CS8

	// 入力バッファリングの設定
	term.Cc[unix.VTIME] = 1
	term.Cc[unix.VMIN] = 1

	// マウスサポートを有効化
	if _, err := os.Stdout.WriteString("\x1b[?1000h\x1b[?1002h\x1b[?1015h\x1b[?1006h"); err != nil {
		return err
	}

	// 設定を適用
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, term); err != nil {
		return err
	}

	return nil
}

func GetWinSize() (screenRows, screenCols int) {
	// ウィンドウサイズの取得
	var ws *unix.Winsize
	var err error
	ws, err = unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		panic(err)
	}

	screenRows = int(ws.Row)
	screenCols = int(ws.Col)

	return screenRows, screenCols
}

// enableRawMode は端末をRawモードに設定する
func EnableRawMode() (*TerminalState, error) {
	term := &TerminalState{}

	// 現在の設定を保存
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}
	term.origTermios = termios

	// グローバル状態を設定
	globalTermState = term

	// 端末の初期化
	if err := InitTerminal(); err != nil {
		return nil, err
	}

	return term, nil
}

// disableRawMode は端末の設定を元の状態に戻す
func (term *TerminalState) DisableRawMode() error {
	// マウスサポートを無効化
	os.Stdout.WriteString("\x1b[?1000l\x1b[?1002l\x1b[?1015l\x1b[?1006l")

	if term.origTermios != nil {
		if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, term.origTermios); err != nil {
			return err
		}
		// 状態をクリア
		globalTermState = nil
		term.origTermios = nil
	}
	return nil
}
