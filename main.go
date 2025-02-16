package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kilo/editor"
)

func die(err error) {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func main() {
	// エディタを初期化
	ed, err := editor.New(false)
	if err != nil {
		die(err)
	}
	defer ed.Cleanup()

	// コマンドライン引数からファイル名を取得
	if len(os.Args) > 1 {
		if err := ed.OpenFile(os.Args[1]); err != nil {
			die(err)
		}
	}

	// シグナルハンドリングのための準備
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// シグナルを受け取った際の処理
	go func() {
		<-sigChan
		ed.Quit() // 終了メソッドを使用
	}()

	// エディタのメインループ
	for {
		select {
		case <-ed.QuitChan():
			return // cleanup関数が遅延実行される
		default:
			if err := ed.RefreshScreen(); err != nil {
				die(err)
			}

			if err := ed.ProcessKeypress(); err != nil {
				die(err)
			}
		}
	}
}
