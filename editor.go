package main

import (
	"github.com/wasya-io/go-kilo/app/boundary/filemanager"
	"github.com/wasya-io/go-kilo/app/boundary/logger"
	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/boundary/reader"
	"github.com/wasya-io/go-kilo/app/boundary/writer"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core/term"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/controller"
	"github.com/wasya-io/go-kilo/app/usecase/editor"
	"github.com/wasya-io/go-kilo/app/usecase/parser"
)

func NewEditor() (*editor.Editor, error) {
	conf := config.LoadConfig()
	logger := logger.New(conf.DebugMode)

	// イベントバスの初期化
	eventBus := event.NewBus()

	// エディタの初期化
	c := contents.NewContents(logger)
	fileManager := filemanager.NewFileManager(c)

	// インプットプロバイダの初期化
	parser := parser.NewStandardInputParser(logger)
	reader := reader.NewStandardKeyReader(logger)
	inputProvider := input.NewStandardInputProvider(logger, reader, parser)

	// 2. ウィンドウサイズの取得
	screenRows, screenCols := term.GetWinSize()

	builder := contents.NewBuilder()
	writer := writer.NewStandardScreenWriter()
	message := contents.NewMessage("", nil)
	cursor := cursor.NewCursor()
	screen := screen.NewScreen(builder, writer, message, cursor, screenRows, screenCols)

	// イベントバスをコントローラーに渡す
	controller := controller.NewController(screen, c, fileManager, inputProvider, logger, eventBus)

	ed, err := editor.New(
		false,
		conf,
		logger,
		c,
		inputProvider,
		screen,
		controller,
		eventBus, // イベントバスをエディターに渡す
	)

	return ed, err
}
