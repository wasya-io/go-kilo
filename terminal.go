package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalState は端末の元の状態を保持する構造体
// origTermios: 端末の元の設定を保存し、プログラム終了時に復元するために使用
type terminalState struct {
	origTermios *unix.Termios
}

// enableRawMode は端末をRawモードに設定する
// Rawモードでは以下の設定が行われる：
// - エコーを無効化 (入力文字が画面に表示されない)
// - カノニカルモードを無効化 (入力を1行ずつではなく即座に処理)
// - Ctrl+C, Ctrl+Zなどのシグナルを無効化
// - Ctrl+V等の特殊入力を無効化
// - Ctrl+S, Ctrl+Qのフロー制御を無効化
// - CR->NL変換を無効化
// - 8ビットデータを有効化
// - 出力の後処理を無効化
func enableRawMode() (*terminalState, error) {
	term := &terminalState{}

	// 現在の端末設定を保存
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}

	term.origTermios = termios

	// 端末をRawモードに設定
	raw := *termios
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN               // ローカルフラグの設定
	raw.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP // 入力フラグの設定
	raw.Cflag |= unix.CS8                                                         // 文字サイズを8ビットに設定
	raw.Oflag &^= unix.OPOST                                                      // 出力処理を無効化

	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, &raw); err != nil {
		return nil, err
	}

	return term, nil
}

// disableRawMode は端末の設定を元の状態に戻す
// プログラム終了時に必ず呼び出される必要がある
func (term *terminalState) disableRawMode() error {
	if term.origTermios != nil {
		return unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, term.origTermios)
	}
	return nil
}
