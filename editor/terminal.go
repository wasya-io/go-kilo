package editor

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalState は端末の元の状態を保持する構造体
type terminalState struct {
	origTermios *unix.Termios
}

// enableRawMode は端末をRawモードに設定する
func enableRawMode() (*terminalState, error) {
	term := &terminalState{}

	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}

	term.origTermios = termios

	raw := *termios
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP
	raw.Cflag |= unix.CS8
	raw.Oflag &^= unix.OPOST

	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, &raw); err != nil {
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
