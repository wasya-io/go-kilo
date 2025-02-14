package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type Editor struct {
	term       *terminalState
	screenRows int
	screenCols int
}

func initEditor() (*Editor, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return nil, err
	}

	e := &Editor{
		screenRows: int(ws.Row),
		screenCols: int(ws.Col),
	}

	// Rawモードを有効化
	term, err := enableRawMode()
	if err != nil {
		return nil, err
	}
	e.term = term

	return e, nil
}

func main() {
	// エディタを初期化
	editor, err := initEditor()
	if err != nil {
		die(err)
	}
	defer editor.term.disableRawMode()

	// エディタのメインループ
	for {
		// 画面の再描画
		if err := editor.refreshScreen(); err != nil {
			die(err)
		}

		// キー入力の処理
		if err := editor.processKeypress(); err != nil {
			die(err)
		}
	}
}

func (e *Editor) refreshScreen() error {
	var b strings.Builder

	// 画面のクリアシーケンス
	b.WriteString("\x1b[2J") // 画面クリア
	b.WriteString("\x1b[H")  // カーソルを左上に移動

	// 各行に'~'を表示
	for y := 0; y < e.screenRows; y++ {
		if y == e.screenRows/3 {
			// 画面中央付近にウェルカムメッセージを表示
			welcome := "Go-Kilo editor -- version 0.0.1"
			if len(welcome) > e.screenCols {
				welcome = welcome[:e.screenCols]
			}
			padding := (e.screenCols - len(welcome)) / 2
			if padding > 0 {
				b.WriteString("~")
				padding--
			}
			for ; padding > 0; padding-- {
				b.WriteString(" ")
			}
			b.WriteString(welcome)
		} else {
			b.WriteString("~")
		}

		// 行末までクリアして改行
		b.WriteString("\x1b[K") // 現在の行の残りをクリア
		if y < e.screenRows-1 {
			b.WriteString("\r\n")
		}
	}

	// カーソルを左上に戻す
	b.WriteString("\x1b[H")

	// バッファの内容を一度に出力
	_, err := fmt.Print(b.String())
	return err
}

func (e *Editor) processKeypress() error {
	buf := make([]byte, 1)
	_, err := os.Stdin.Read(buf)
	if err != nil {
		return err
	}

	// Ctrl-Q または Ctrl-C で終了
	if buf[0] == 'q'&0x1f || buf[0] == 'c'&0x1f {
		fmt.Print("\x1b[2J")
		fmt.Print("\x1b[H")
		os.Exit(0)
	}

	return nil
}

func die(err error) {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
