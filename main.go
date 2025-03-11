package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/wasya-io/go-kilo/app/boundary/logger"
	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/boundary/reader"
	"github.com/wasya-io/go-kilo/app/boundary/writer"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core/term"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/parser"
	"github.com/wasya-io/go-kilo/editor"
	"github.com/wasya-io/go-kilo/editor/events"
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

	conf := config.LoadConfig()
	logger := logger.New(conf.DebugMode)

	// エディタの初期化
	eventManager := events.NewEventManager()

	buffer := contents.NewContents(logger)
	fileManager := editor.NewFileManager(buffer, eventManager)

	// インプットプロバイダの初期化
	parser := parser.NewStandardInputParser(logger)
	reader := reader.NewStandardKeyReader(logger)
	inputProvider := input.NewStandardInputProvider(logger, reader, parser)

	// 2. ウィンドウサイズの取得
	screenRows, screenCols := term.GetWinSize()

	builder := contents.NewBuilder()
	writer := writer.NewStandardScreenWriter()
	screen := screen.NewScreen(builder, writer, screenRows, screenCols)

	ed, err := editor.New(false, conf, logger, eventManager, buffer, fileManager, inputProvider, *screen)
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
