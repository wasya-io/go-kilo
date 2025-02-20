package editor

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalState は端末の元の状態を保持する構造体
type terminalState struct {
	origTermios *unix.Termios
}

// InitTerminal は端末の初期化を行う
func InitTerminal() error {
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

	// 設定を適用
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, term); err != nil {
		return err
	}

	return nil
}

// enableRawMode は端末をRawモードに設定する
func enableRawMode() (*terminalState, error) {
	term := &terminalState{}

	// 現在の設定を保存
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}
	term.origTermios = termios

	// 端末の初期化
	if err := InitTerminal(); err != nil {
		return nil, err
	}

	return term, nil
}

// disableRawMode は端末の設定を元の状態に戻す
func (term *terminalState) disableRawMode() error {
	if term.origTermios != nil {
		return unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, term.origTermios)
	}
	return nil
}
