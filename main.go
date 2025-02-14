package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

type Editor struct {
	term       *terminalState
	screenRows int
	screenCols int
	quit       chan struct{} // 終了制御用のチャネル
}

func initEditor() (*Editor, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return nil, err
	}

	e := &Editor{
		screenRows: int(ws.Row),
		screenCols: int(ws.Col),
		quit:       make(chan struct{}),
	}

	// Rawモードを有効化
	term, err := enableRawMode()
	if err != nil {
		return nil, err
	}
	e.term = term

	return e, nil
}

// cleanup は終了時の後処理を行う
func (e *Editor) cleanup() {
	e.term.disableRawMode()
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
}

func main() {
	// エディタを初期化
	editor, err := initEditor()
	if err != nil {
		die(err)
	}
	defer editor.cleanup()

	// シグナルハンドリングのための準備
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// シグナルを受け取った際の処理
	go func() {
		<-sigChan
		close(editor.quit) // 終了シグナルを送信
	}()

	// エディタのメインループ
	for {
		select {
		case <-editor.quit:
			return // cleanup関数が遅延実行される
		default:
			if err := editor.refreshScreen(); err != nil {
				die(err)
			}

			if err := editor.processKeypress(); err != nil {
				die(err)
			}
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
		close(e.quit)
		return nil
	}

	return nil
}

func die(err error) {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
