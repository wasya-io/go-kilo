package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

func main() {
	// グローバルなパニックハンドラを設定
	defer func() {
		if r := recover(); r != nil {
			// TODO: dieと統合できそう。どちらのエスケープシーケンスが正しいのか精査して改善する
			// 端末をリセットするエスケープシーケンス
			fmt.Print("\x1b[?1000l\x1b[?1002l\x1b[?1015l\x1b[?1006l") // マウスモードを無効化
			fmt.Print("\x1b[2J\x1b[H")                                // 画面をクリア
			fmt.Print("\x1b[?25h")                                    // カーソルを表示

			// エラー情報を出力
			fmt.Fprintf(os.Stderr, "Editor crashed: %v\n", r)
			fmt.Fprintf(os.Stderr, "Stack trace:\n%s", debug.Stack())
			os.Exit(1)
		}
	}()

	// シグナルハンドリングの設定
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ed, err := NewEditor()
	if err != nil {
		die(err)
	}
	defer ed.Cleanup() // 確実なクリーンアップを保証

	// コマンドライン引数の処理
	if len(os.Args) > 1 {
		if err := ed.OpenFile(os.Args[1]); err != nil {
			die(err)
		}
	}

	// シグナル処理用のゴルーチン
	go func() {
		<-sigChan
		ed.Cleanup() // クリーンアップを実行
		os.Exit(0)
	}()

	// エディタのメインループ
	if err := ed.Run(); err != nil {
		ed.Cleanup() // エラー時もクリーンアップを実行
		die(err)
	}
}

func die(err error) {
	fmt.Print("\x1b[2J") // 画面をクリア
	fmt.Print("\x1b[H")  // カーソルを左上に移動
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
