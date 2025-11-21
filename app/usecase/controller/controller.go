package controller

import (
	"fmt"
	"time"

	"github.com/wasya-io/go-kilo/app/boundary/filemanager"
	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/config"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/command"
)

type Controller struct {
	screen                *screen.Screen
	contents              *contents.Contents
	fileManager           filemanager.FileManager
	inputProvider         input.Provider
	logger                core.Logger
	eventBuffer           []key.KeyEvent
	quitWarningShown      bool
	debugMode             bool
	statusMessageDuration int
	Quit                  chan struct{}
	eventBus              *event.Bus // 追加: イベントバス
}

// GetContents はコントローラーが管理しているコンテンツを返します。
// 主にテスト用途で使用されます。
func (c *Controller) GetContents() *contents.Contents {
	return c.contents
}

func NewController(
	screen *screen.Screen,
	contents *contents.Contents,
	fileManager filemanager.FileManager,
	inputProvider input.Provider,
	logger core.Logger,
	eventBus *event.Bus, // 追加: イベントバス
) *Controller {
	c := &Controller{
		screen:                screen,
		contents:              contents,
		fileManager:           fileManager,
		inputProvider:         inputProvider,
		logger:                logger,
		Quit:                  make(chan struct{}),
		statusMessageDuration: 5,
		eventBus:              eventBus, // 追加: イベントバスの設定
	}

	// イベントハンドラーの登録
	c.registerEventHandlers()

	return c
}

// registerEventHandlers はイベントハンドラーを登録します
func (c *Controller) registerEventHandlers() {
	// 保存イベントのハンドラー
	saveHandler := event.NewSingleTypeHandler(event.TypeSave, func(e event.Event) (bool, error) {
		if saveEvent, ok := e.Payload.(event.SaveEvent); ok {
			c.logger.Log("event", fmt.Sprintf("Save event received: %s", saveEvent.Filename))
			c.setStatusMessage("Saving...")
			// イベントから渡されたファイル名を使用して保存
			// これにより、"Save As"で指定された新しいファイル名が使用される
			err := c.fileManager.SaveFile(saveEvent.Filename, c.contents.GetAllLines())
			if err != nil {
				c.setStatusMessage("Error saving file: %v", err)
				return false, err
			}
			c.setStatusMessage("File saved")

			// 画面を明示的に更新して、isDirtyの状態変化をステータスバーに反映する
			if err := c.RefreshScreen(); err != nil {
				c.logger.Log("error", fmt.Sprintf("Failed to refresh screen after save: %v", err))
			}

			return true, nil
		}
		return false, nil
	})

	// 終了イベントのハンドラー
	quitHandler := event.NewSingleTypeHandler(event.TypeQuit, func(e event.Event) (bool, error) {
		if quitEvent, ok := e.Payload.(event.QuitEvent); ok {
			c.logger.Log("event", fmt.Sprintf("Quit event received, force=%v", quitEvent.Force))

			// ダーティ状態かつ強制終了でなく、警告が未表示の場合
			if c.contents.IsDirty() && !quitEvent.Force && !c.quitWarningShown {
				c.quitWarningShown = true
				c.logger.Log("warning", "File has unsaved changes. Showing warning message.")

				// デバッグメッセージを一時的にクリア（警告メッセージを確実に表示するため）
				c.screen.ClearDebugMessage()

				// 警告メッセージを設定
				c.screen.SetMessage("Warning! File has unsaved changes. Press Ctrl-X or Ctrl-C again to quit.")

				// 画面を即座に更新して確実にメッセージを表示
				if err := c.RefreshScreen(); err != nil {
					c.logger.Log("error", fmt.Sprintf("Failed to refresh screen: %v", err))
				}

				// 0.1秒ウェイトを入れてメッセージを確実に表示
				time.Sleep(100 * time.Millisecond)

				return true, nil
			}

			// 終了処理を実行
			c.logger.Log("system", "Shutting down editor")

			// チャネルが既に閉じられているか確認して安全に閉じる
			if !c.isQuitChannelClosed() {
				close(c.Quit)
			} else {
				c.logger.Log("warning", "Attempted to close an already closed quit channel")
			}

			return true, nil
		}
		return false, nil
	})

	// イベントバスにハンドラーを登録
	c.eventBus.Subscribe(saveHandler)
	c.eventBus.Subscribe(quitHandler)
}

// isQuitChannelClosed はQuitチャネルが既に閉じられているかを非ブロッキングで確認します
func (c *Controller) isQuitChannelClosed() bool {
	select {
	case <-c.Quit:
		// チャネルから値を受信できた場合、チャネルは閉じられている
		return true
	default:
		// デフォルトケースがあるため非ブロッキングで、チャネルはまだ閉じられていない
		return false
	}
}

// PublishSaveEvent は保存イベントを発行します
func (c *Controller) PublishSaveEvent(filename string, force bool) {
	c.logger.Log("event", fmt.Sprintf("Publishing save event: %s", filename))
	c.eventBus.Publish(event.NewSaveEvent(filename, force))
}

// PublishQuitEvent は終了イベントを発行します
func (c *Controller) PublishQuitEvent(force bool) {
	c.logger.Log("event", fmt.Sprintf("Publishing quit event, force=%v", force))
	c.eventBus.Publish(event.NewQuitEvent(force))
}

func (c *Controller) RefreshScreen() error {
	// UI更新の前にスクロール位置を更新
	c.updateScroll()

	// ファイル名のロギングを追加
	filename := c.fileManager.GetFilename()
	c.logger.Log("screen", fmt.Sprintf("Refreshing screen with filename: '%s'", filename))

	// UIの更新処理を実行
	err := c.screen.Redraw(c.contents, filename)
	if err != nil {
		return err
	}

	// メッセージ表示後は即座にフラッシュする
	return c.screen.Flush()
}

// updateScroll はカーソル位置に基づいてスクロール位置を更新する
func (c *Controller) updateScroll() {
	// スクロール位置の更新処理
	offsetCol, offsetRow := c.screen.GetOffset()

	// UI経由でカーソル位置を取得
	pos := c.screen.GetCursor().ToPosition()
	row := c.contents.GetRow(pos.Y)
	if row == nil {
		return
	}

	screenRowLines := c.screen.GetRowLines()
	// ステータスバーとメッセージバー用に2行確保
	visibleLines := screenRowLines - 2

	// カーソル周辺に表示する余白行数
	const scrollMargin = 3

	// スクロール条件の計算
	// カーソルが表示領域の上端より上にある場合
	if pos.Y < offsetRow+scrollMargin {
		offsetRow = pos.Y - scrollMargin
		if offsetRow < 0 {
			offsetRow = 0
		}
	}
	// カーソルが表示領域の下端に近づいた場合（余白を確保）
	if pos.Y >= offsetRow+visibleLines-scrollMargin {
		offsetRow = pos.Y - visibleLines + scrollMargin + 1
	}

	// 水平方向のスクロール
	cursorScreenPos := row.OffsetToScreenPosition(pos.X)
	if cursorScreenPos < offsetCol {
		offsetCol = cursorScreenPos
	}

	screenColLines := c.screen.GetColLines()
	rightMargin := (screenColLines * 4) / 5
	if cursorScreenPos >= (offsetCol + rightMargin) {
		offsetCol = cursorScreenPos - rightMargin + 1
	}

	// スクロール位置の制限
	if offsetRow < 0 {
		offsetRow = 0
	}
	if offsetCol < 0 {
		offsetCol = 0
	}

	maxRow := c.contents.GetLineCount() - 1
	if offsetRow > maxRow {
		offsetRow = maxRow
	}

	// スクロール位置の更新
	c.screen.SetRowOffset(offsetRow)
	c.screen.SetColOffset(offsetCol)
}

// Process はキー入力を処理する
func (c *Controller) Process() error {
	event, err := c.readEvent()
	if err != nil {
		c.logger.Log("error", fmt.Sprintf("readEvent error: %v", err))
		return err
	}

	// コマンドを作成
	command, err := c.createCommand(event)
	if err != nil {
		return err
	}

	// 画面更新を必ず行う（コマンドの有無に関わらず）
	defer c.RefreshScreen()

	if command != nil {
		// コマンドを実行
		c.logger.Log("command", fmt.Sprintf("Executed command: %T", command))
		if err := command.Execute(); err != nil {
			return err
		}
	}

	return nil
}

// readEvent はイベントを読み取る
func (c *Controller) readEvent() (key.KeyEvent, error) {
	// バッファにイベントがある場合はそれを返す
	if len(c.eventBuffer) > 0 {
		event := c.eventBuffer[0]
		c.eventBuffer = c.eventBuffer[1:]
		return event, nil
	}

	event, remainingEvents, err := c.inputProvider.GetInputEvents()
	if err != nil {
		c.logger.Log("error", fmt.Sprintf("Keypress error: %v", err))
		return key.KeyEvent{}, err
	}

	// 残りのイベントがある場合はバッファに追加
	if len(remainingEvents) > 0 {
		c.eventBuffer = append(c.eventBuffer, remainingEvents...)
	}

	return event, nil
}

// OpenFile は指定されたファイルを読み込む
func (c *Controller) OpenFile(filename string) error {
	c.logger.Log("file", fmt.Sprintf("Opening file: '%s'", filename))
	err := c.fileManager.OpenFile(filename)
	if err != nil {
		c.logger.Log("error", fmt.Sprintf("Failed to open file: %v", err))
		return err
	}
	c.logger.Log("file", fmt.Sprintf("File opened successfully: '%s', current filename from fileManager: '%s'",
		filename, c.fileManager.GetFilename()))
	return nil
}

func (c *Controller) insertChar(ch rune) {
	pos := c.screen.GetCursor().ToPosition()
	c.contents.InsertChar(contents.Position{X: pos.X, Y: pos.Y}, ch)
	// カーソルを1つ進める
	c.screen.SetCursorPosition(pos.X+1, pos.Y)
	c.RefreshScreen()
}

func (c *Controller) deleteChar() {
	c.logger.Log("edit", "Deleting character")
	pos := c.screen.GetCursor().ToPosition()

	if pos.X > 0 {
		// 行の途中での削除
		c.contents.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
		c.screen.SetCursorPosition(pos.X-1, pos.Y) // カーソルを1つ左に移動
	} else if pos.Y > 0 {
		// 行頭での削除（前の行との結合）
		prevRow := c.contents.GetRow(pos.Y - 1)
		if prevRow != nil {
			targetX := prevRow.GetRuneCount() // 前の行の末尾位置
			c.contents.DeleteChar(contents.Position{X: pos.X, Y: pos.Y})
			c.screen.SetCursorPosition(targetX, pos.Y-1) // 前の行の末尾へ移動
		}
	}

	c.RefreshScreen()
}

func (c *Controller) moveCursor(movement cursor.Movement) {
	c.logger.Log("cursor", fmt.Sprintf("Moving cursor: %v", movement))
	// Buffer直接操作からUI経由に変更
	c.screen.MoveCursor(movement, c.contents)
	c.updateScroll()
}

func (c *Controller) insertNewline() {
	c.logger.Log("edit", "Inserting newline")
	cursor := c.screen.GetCursor()
	pos := cursor.ToPosition()

	// 現在行のインデント文字数を計測
	currentLine := c.contents.GetContentLine(pos.Y)

	// 行頭のインデントを計算する
	indentSize := 0
	for _, r := range currentLine {
		if r == '\t' || r == ' ' {
			indentSize++
		} else {
			break
		}
	}

	// カーソルが行頭のインデント部分の中にある場合は、その位置までをインデントとして次の行に適用する
	if pos.X < indentSize {
		indentSize = pos.X
	}

	// 改行をインデントサイズとともに挿入
	c.contents.InsertNewline(contents.Position{X: pos.X, Y: pos.Y}, indentSize)

	// カーソルを新しい行のインデント位置に設定
	cursor.NewLine() // まず次の行の行頭へ移動
	// インデント位置にカーソルを設定
	c.screen.SetCursorPosition(indentSize, pos.Y+1)

	c.updateScroll()
	c.RefreshScreen()
}

// setStatusMessage はステータスメッセージを設定する（非公開メソッド）
func (c *Controller) setStatusMessage(format string, args ...interface{}) {
	if c.debugMode {
		format = "[in Debug] " + format
	}
	// UIコンポーネントのSetMessageメソッドを呼び出す
	c.screen.SetMessage(format, args...)

	// 即座に画面を更新して変更を反映
	c.RefreshScreen()
}

// createCommand はキーイベントからコマンドを作成する
func (c *Controller) createCommand(event key.KeyEvent) (command.Command, error) {
	switch event.Type {
	case key.KeyEventChar, key.KeyEventSpecial:
		// 警告状態をクリア
		if c.quitWarningShown {
			c.quitWarningShown = false
			c.setStatusMessage("")
		}
		if event.Type == key.KeyEventChar {
			fn := func(r rune) error {
				c.insertChar(r)
				return nil
			}
			return command.NewInsertCharCommand(event.Rune, fn), nil
		}
		return c.createSpecialKeyCommand(event.Key), nil
	case key.KeyEventControl:
		return c.createControlKeyCommand(event.Key), nil
	case key.KeyEventMouse:
		if event.Key == key.KeyMouseWheel {
			// マウスホイールイベントは専用のカーソル移動コマンドを使用
			switch event.MouseAction {
			case key.MouseScrollUp:
				fn := func() error {
					c.logger.Log("cursor", "Moving cursor up")
					c.moveCursor(cursor.MouseWheelUp)
					c.updateScroll()
					return nil
				}
				return command.NewCommand(fn), nil
			case key.MouseScrollDown:
				fn := func() error {
					c.logger.Log("cursor", "Moving cursor down")
					c.moveCursor(cursor.MouseWheelDown)
					c.updateScroll()
					return nil
				}
				return command.NewCommand(fn), nil
			}
		} else if event.Key == key.KeyMouseClick {
			// マウスクリックイベントを処理
			switch event.MouseAction {
			case key.MouseLeftClick:
				fn := func() error {
					c.logger.Log("mouse", fmt.Sprintf("Mouse left click at row: %d, col: %d", event.MouseRow, event.MouseCol))
					c.handleMouseClick(event.MouseRow, event.MouseCol)
					return nil
				}
				return command.NewCommand(fn), nil
			}
			c.logger.Log("mouse", fmt.Sprintf("Unhandled mouse click event: %v", event.MouseAction))
		}
	}
	return nil, nil
}

// handleMouseClick はマウスクリックイベントを処理し、カーソルを移動します
func (c *Controller) handleMouseClick(row, col int) {
	// スクロールオフセットを考慮して、クリックされた画面上の位置をテキストバッファ上の位置に変換
	offsetCol, offsetRow := c.screen.GetOffset()

	// クリック位置にオフセットを加算して実際のテキスト位置を計算
	bufferRow := row + offsetRow
	bufferCol := col + offsetCol

	// バッファの範囲内かチェック
	if bufferRow >= c.contents.GetLineCount() {
		bufferRow = c.contents.GetLineCount() - 1
		if bufferRow < 0 {
			bufferRow = 0
		}
	}

	// 行を取得
	targetRow := c.contents.GetRow(bufferRow)
	if targetRow == nil {
		return
	}

	// 画面上の列位置をバッファ内の文字位置（バイト位置）に変換
	// この処理はタブ文字や全角文字を考慮する必要があります
	bufferCol = targetRow.ScreenPositionToOffset(bufferCol)

	// 行内の有効な位置にカーソルを制限
	maxCol := targetRow.GetRuneCount()
	if bufferCol > maxCol {
		bufferCol = maxCol
	}
	if bufferCol < 0 {
		bufferCol = 0
	}

	// カーソル位置を更新
	c.logger.Log("cursor", fmt.Sprintf("Setting cursor to row: %d, col: %d", bufferRow, bufferCol))
	c.screen.SetCursorPosition(bufferCol, bufferRow)

	// スクロール位置を更新（カーソル位置に応じて画面をスクロール）
	c.updateScroll()
}

// createSpecialKeyCommand は特殊キーに対応するコマンドを作成する
func (c *Controller) createSpecialKeyCommand(k key.Key) command.Command {
	switch k {
	case key.KeyArrowLeft:
		fn := func() error {
			c.moveCursor(cursor.CursorLeft)
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyArrowRight:
		fn := func() error {
			c.moveCursor(cursor.CursorRight)
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyArrowUp:
		fn := func() error {
			c.moveCursor(cursor.CursorUp)
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyArrowDown:
		fn := func() error {
			c.moveCursor(cursor.CursorDown)
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyBackspace:
		fn := func() error {
			c.logger.Log("edit", "Deleting character")
			c.deleteChar()
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyEnter:
		fn := func() error {
			c.logger.Log("edit", "Inserting newline")
			c.insertNewline()
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyTab:
		fn := func() error {
			c.logger.Log("edit", "Inserting tab")
			// タブは空白に展開
			tabWidth := config.GetTabWidth()
			for i := 0; i < tabWidth; i++ {
				c.insertChar(' ')
			}
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyShiftTab:
		fn := func() error {
			c.logger.Log("edit", "Inserting shift-tab")
			cur := c.screen.GetCursor()
			pos := cur.ToPosition()
			content := c.contents.GetContentLine(pos.Y)

			// カーソルが行頭にある場合は処理を行わない
			if pos.X <= 0 {
				return nil
			}

			// カーソル位置の左側のスペース数を数える
			leftSpaces := 0
			for i := pos.X - 1; i >= 0; i-- {
				if content[i] != ' ' {
					break
				}
				leftSpaces++
			}

			if leftSpaces == 0 {
				return nil // 左側にスペースがない場合は何もしない
			}

			// 削除するスペース数を計算
			tabWidth := config.GetTabWidth()
			spacesToDelete := leftSpaces % tabWidth
			if spacesToDelete == 0 {
				spacesToDelete = tabWidth
			}

			// スペースを削除
			for i := 0; i < spacesToDelete; i++ {
				c.deleteChar()
			}

			// 残りのカーソル移動
			for i := 1; i < (spacesToDelete - 1); i++ {
				c.moveCursor(cursor.CursorLeft)
			}

			return nil
		}
		return command.NewCommand(fn)
	default:
		return nil
	}
}

// createControlKeyCommand はコントロールキーに対応するコマンドを作成する
func (c *Controller) createControlKeyCommand(k key.Key) command.Command {
	switch k {
	case key.KeyCtrlS:
		// 保存処理をイベントベースに変更
		fn := func() error {
			filename := c.fileManager.GetFilename()
			if filename == "" {
				var err error
				filename, err = c.prompt("Save as: ")
				if err != nil {
					return err
				}
				if filename == "" {
					c.setStatusMessage("Save aborted")
					return nil
				}
			}
			c.logger.Log("command", "Saving file")
			c.PublishSaveEvent(filename, false)
			return nil
		}
		return command.NewCommand(fn)
	case key.KeyCtrlX, key.KeyCtrlC:
		// 終了処理をイベントベースに変更
		fn := func() error {
			c.logger.Log("command", "Quitting")
			c.PublishQuitEvent(false)
			return nil
		}
		return command.NewCommand(fn)
	default:
		return nil
	}
}

// prompt はユーザーに入力を求める
func (c *Controller) prompt(prompt string) (string, error) {
	c.setStatusMessage(prompt)

	var input []rune
	for {
		event, err := c.readEvent()
		if err != nil {
			return "", err
		}

		switch event.Type {
		case key.KeyEventChar:
			input = append(input, event.Rune)
			c.setStatusMessage(prompt + string(input))
		case key.KeyEventSpecial:
			switch event.Key {
			case key.KeyEnter:
				if len(input) > 0 {
					c.setStatusMessage("")
					return string(input), nil
				}
			case key.KeyBackspace:
				if len(input) > 0 {
					input = input[:len(input)-1]
					c.setStatusMessage(prompt + string(input))
				}
			case key.KeyEsc:
				c.setStatusMessage("")
				return "", nil
			}
		case key.KeyEventControl:
			// コントロールキー（Ctrl+Cなど）が押された場合はキャンセル扱い
			if event.Key == key.KeyCtrlC || event.Key == key.KeyCtrlX {
				c.setStatusMessage("")
				return "", nil
			}
		}
	}
}
