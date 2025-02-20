package editor

import (
	"fmt"
	"os"
	"strings"

	"github.com/wasya-io/go-kilo/editor/events"
)

// UI は画面表示を管理する構造体
type UI struct {
	screenRows    int
	screenCols    int
	message       string
	messageArgs   []interface{}
	lastColOffset int                  // 前回のcolOffsetを保存
	eventManager  *events.EventManager // 追加：イベントマネージャー
	needsRefresh  bool                 // 追加：画面更新が必要かどうかのフラグ
}

// NewUI は新しいUIインスタンスを作成する
func NewUI(rows, cols int, eventManager *events.EventManager) *UI {
	ui := &UI{
		screenRows:    rows,
		screenCols:    cols,
		lastColOffset: 0,
		eventManager:  eventManager,
		needsRefresh:  false,
	}

	// バッファイベントを購読してUI更新を最適化
	eventManager.Subscribe(events.BufferEventType, ui.handleBufferEvent)

	return ui
}

// handleBufferEvent はバッファの変更に応じてUI更新を最適化する
func (ui *UI) handleBufferEvent(event events.Event) {
	if bufferEvent, ok := event.(*events.BufferEvent); ok {
		// バッファの状態が実際に変更された場合のみ更新を行う
		if bufferEvent.HasChanges() {
			// すべての更新を単一の画面更新にまとめる
			ui.needsRefresh = true
		}
	}
}

// handleUIEvent はUIイベントを処理する
func (ui *UI) handleUIEvent(event *events.UIEvent) {
	switch event.SubType {
	case events.UIRefresh:
		// 既存の全画面更新処理を維持
		ui.needsRefresh = true

	case events.UIScroll:
		if data, ok := event.Data.(events.ScrollData); ok {
			ui.handleScrollEvent(data)
		}

	case events.UIStatusMessage:
		if data, ok := event.Data.(events.StatusMessageData); ok {
			ui.handleStatusMessage(data)
		}

	case events.UIEditorPartialRefresh:
		if data, ok := event.Data.(events.EditorUpdateData); ok {
			ui.handlePartialRefresh(data)
		}

	case events.UICursorUpdate:
		if data, ok := event.Data.(events.CursorData); ok {
			ui.handleCursorUpdate(data)
		}

	case events.UIStatusBarRefresh:
		ui.refreshStatusBar()

	case events.UIMessageBarRefresh:
		ui.refreshMessageBar()
	}
}

// handleScrollEvent はスクロールイベントを処理する
func (ui *UI) handleScrollEvent(data events.ScrollData) {
	if data.IsSmooth {
		// スムーズスクロールの場合は徐々に更新
		ui.performSmoothScroll(data)
	} else {
		// 通常のスクロール処理
		ui.lastColOffset = data.ColOffset
		ui.needsRefresh = true
	}
}

// handlePartialRefresh は部分更新を処理する
func (ui *UI) handlePartialRefresh(data events.EditorUpdateData) {
	if data.ForceAll {
		ui.needsRefresh = true
		return
	}
	// 指定された行のみを更新対象としてマーク
	ui.markLinesForUpdate(data.Lines)
}

// handleCursorUpdate はカーソル位置の更新を処理する
func (ui *UI) handleCursorUpdate(data events.CursorData) {
	// カーソル位置の更新のみを行う（画面全体の更新は行わない）
	ui.updateCursorPosition(data)
}

// handleStatusMessage はステータスメッセージを処理する
func (ui *UI) handleStatusMessage(data events.StatusMessageData) {
	ui.message = data.Message
	ui.messageArgs = make([]interface{}, len(data.Args))
	copy(ui.messageArgs, data.Args)
	ui.refreshMessageBar()
}

// performSmoothScroll はスムーズスクロールを実行する
func (ui *UI) performSmoothScroll(data events.ScrollData) {
	// スムーズスクロールの実装（アニメーションなど）
	// 現時点では通常のスクロールと同様に処理
	ui.lastColOffset = data.ColOffset
	ui.needsRefresh = true
}

// markLinesForUpdate は更新が必要な行をマークする
func (ui *UI) markLinesForUpdate(lines []int) {
	// 部分更新のための行管理を実装
	ui.needsRefresh = true // 現時点では全体更新にフォールバック
}

// updateCursorPosition はカーソル位置を更新する
func (ui *UI) updateCursorPosition(data events.CursorData) {
	// カーソル位置の更新処理
	// 現時点では画面更新のトリガーのみ
	ui.needsRefresh = true
}

// refreshStatusBar はステータスバーを更新する
func (ui *UI) refreshStatusBar() {
	ui.needsRefresh = true
}

// refreshMessageBar はメッセージバーを更新する
func (ui *UI) refreshMessageBar() {
	ui.needsRefresh = true
}

// publishRefreshEvent は画面更新イベントを発行する
func (ui *UI) publishRefreshEvent(fullRefresh bool) {
	if ui.eventManager == nil {
		return
	}

	event := events.NewUIEvent(events.UIRefresh, struct {
		FullRefresh bool
	}{
		FullRefresh: fullRefresh,
	})
	ui.eventManager.Publish(event)
}

// publishCursorUpdateEvent はカーソル更新イベントを発行する
func (ui *UI) publishCursorUpdateEvent(pos events.Position) {
	if ui.eventManager == nil {
		return
	}

	event := events.NewUIEvent(events.UIScroll, pos)
	ui.eventManager.Publish(event)
}

// RefreshScreen は画面を更新する
func (ui *UI) RefreshScreen(buffer *Buffer, filename string, rowOffset, colOffset int) error {
	var builder strings.Builder

	// カーソルを非表示にする
	builder.WriteString("\x1b[?25l")

	// 画面をクリアして原点に移動
	builder.WriteString(ui.clearScreen())
	builder.WriteString(ui.moveCursorToHome())

	// バッファの内容を描画
	builder.WriteString(ui.drawRows(buffer, rowOffset, colOffset))
	builder.WriteString(ui.drawStatusBar(filename, buffer.IsDirty()))
	builder.WriteString(ui.drawMessageBar())

	// カーソル位置を更新
	x, y := buffer.GetCursorXY()
	row := buffer.getRow(y)

	// 画面上のカーソル位置を計算
	screenY := y - rowOffset + 1
	screenX := 1

	if row != nil {
		// スクリーン座標の計算
		// カーソルがある位置までの文字幅の合計を計算
		visualPos := 0
		for i := 0; i < x && i < row.GetRuneCount(); i++ {
			visualPos += row.GetRuneWidth(i)
		}
		screenX = visualPos - colOffset + 1

		// 範囲チェックと調整
		if screenX < 1 {
			screenX = 1
		}
		if screenX > ui.screenCols {
			screenX = ui.screenCols
		}
	}

	// カーソル位置の範囲チェックと調整
	screenY = max(1, min(screenY, ui.screenRows-2))

	// カーソルを新しい位置に移動
	builder.WriteString(fmt.Sprintf("\x1b[%d;%dH", screenY, screenX))

	// カーソルを再表示
	builder.WriteString("\x1b[?25h")

	// すべての変更を一度に出力
	_, err := os.Stdout.WriteString(builder.String())
	return err
}

// SetMessage はステータスメッセージを設定する
func (ui *UI) SetMessage(format string, args ...interface{}) {
	// 同じメッセージが既に設定されている場合は、イベントを発行しない
	if ui.message == format && len(ui.messageArgs) == len(args) {
		sameArgs := true
		for i, arg := range args {
			if arg != ui.messageArgs[i] {
				sameArgs = false
				break
			}
		}
		if sameArgs {
			return
		}
	}

	ui.message = format
	ui.messageArgs = make([]interface{}, len(args))
	copy(ui.messageArgs, args)

	// メッセージ更新イベントを発行（既存のメッセージと異なる場合のみ）
	if ui.eventManager != nil {
		event := events.NewUIEvent(events.UIStatusMessage, struct {
			Message string
			Args    []interface{}
		}{
			Message: format,
			Args:    args,
		})
		ui.eventManager.Publish(event)
	}
}

// drawStatusBar はステータスバーを描画する
func (ui *UI) drawStatusBar(filename string, isDirty bool) string {
	status := filename
	if status == "" {
		status = "[No Name]"
	}
	if isDirty {
		status += " [+]"
	}
	return "\x1b[7m" + ui.padLine(status) + "\x1b[m\r\n"
}

// drawMessageBar はメッセージバーを描画する
func (ui *UI) drawMessageBar() string {
	msg := ""
	if ui.message != "" {
		msg = fmt.Sprintf(ui.message, ui.messageArgs...)
		if len(msg) > ui.screenCols {
			msg = msg[:ui.screenCols]
		}
	}
	return "\x1b[K" + msg
}

// clearScreen は画面をクリアする
func (ui *UI) clearScreen() string {
	return "\x1b[2J"
}

// moveCursorToHome はカーソルを原点に移動する
func (ui *UI) moveCursorToHome() string {
	return "\x1b[H"
}

// padLine は行を画面幅に合わせてパディングする
func (ui *UI) padLine(line string) string {
	if len(line) > ui.screenCols {
		return line[:ui.screenCols]
	}
	return line + strings.Repeat(" ", ui.screenCols-len(line))
}

// drawRows は編集領域を描画する
func (ui *UI) drawRows(buffer *Buffer, rowOffset, colOffset int) string {
	var builder strings.Builder
	// 画面の各行について処理
	for y := 0; y < ui.screenRows-2; y++ {
		filerow := y + rowOffset
		// ファイル内の有効な行の場合
		if filerow < buffer.GetLineCount() {
			row := buffer.getRow(filerow)
			if row != nil {
				// テキスト行の描画（スクロール位置を考慮）
				builder.WriteString(ui.drawTextRow(row, colOffset))
			}
		} else {
			// ファイルの終端以降は空行を表示
			builder.WriteString(ui.drawEmptyRow(y, buffer.GetLineCount()))
		}
		// 行末をクリアして改行
		builder.WriteString("\x1b[K\r\n")
	}
	return builder.String()
}

// drawEmptyRow は空行（チルダ）またはウェルカムメッセージを描画
func (ui *UI) drawEmptyRow(y int, totalLines int) string {
	if totalLines == 0 && y == ui.screenRows/3 {
		return ui.drawWelcomeMessage()
	}
	return "~"
}

// drawWelcomeMessage はウェルカムメッセージを描画
func (ui *UI) drawWelcomeMessage() string {
	welcome := "Kilo editor -- version 1.0"
	if len(welcome) > ui.screenCols {
		welcome = welcome[:ui.screenCols]
	}
	padding := (ui.screenCols - len(welcome)) / 2
	var builder strings.Builder
	if padding > 0 {
		builder.WriteString("~")
		padding--
	}
	for ; padding > 0; padding-- {
		builder.WriteString(" ")
	}
	builder.WriteString(welcome)
	return builder.String()
}

// drawTextRow はテキスト行を描画
func (ui *UI) drawTextRow(row *Row, colOffset int) string {
	if row == nil {
		return ""
	}

	var builder strings.Builder
	runes := row.runeSlice // 直接runeSliceを使用
	currentPos := 0        // 現在の表示位置

	for i := 0; i < len(runes); i++ {
		width := row.GetRuneWidth(i)

		// colOffsetより前の文字はスキップ
		if currentPos < colOffset {
			currentPos += width
			continue
		}

		// 画面幅を超える場合は描画終了
		if currentPos-colOffset >= ui.screenCols {
			break
		}

		// 文字を描画
		builder.WriteRune(runes[i])
		currentPos += width
	}

	return builder.String()
}
