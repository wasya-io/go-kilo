package editor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wasya-io/go-kilo/editor/events"
)

// UpdatePriority は更新の優先度を表す
type UpdatePriority int

const (
	LowPriority UpdatePriority = iota
	MediumPriority
	HighPriority
)

// UpdateRequest は画面更新要求を表す
type UpdateRequest struct {
	area     UpdateArea
	priority UpdatePriority
	data     interface{}
}

// UpdateArea は更新が必要な領域を表す
type UpdateArea int

const (
	AreaNone UpdateArea = iota
	AreaCursor
	AreaLine
	AreaStatus
	AreaMessage
	AreaFull
)

// ScrollState はスクロールの状態を管理する
type ScrollState struct {
	targetColOffset  int
	currentColOffset int
	smoothScroll     bool
	scrollSteps      int
}

// LineUpdateState は行の更新状態を管理する
type LineUpdateState struct {
	dirty    map[int]bool // 更新が必要な行番号をマップで管理
	forceAll bool         // 全行の更新が必要かどうか
}

// UI は画面表示を管理する構造体
type UI struct {
	screenRows     int
	screenCols     int
	message        string
	messageArgs    []interface{}
	messageTime    int64                        // メッセージの表示時間を制御
	lastColOffset  int                          // 前回のcolOffsetを保存
	eventManager   *events.EventManager         // 追加：イベントマネージャー
	needsRefresh   bool                         // 追加：画面更新が必要かどうかのフラグ
	updateQueue    []UpdateRequest              // 更新要求のキュー
	batchMode      bool                         // バッチモード中かどうか
	pendingUpdates map[UpdateArea]UpdateRequest // 保留中の更新
	scrollState    ScrollState
	lineState      LineUpdateState
	buffer         strings.Builder // 追加：バッファ
	debugMessage   string          // デバッグメッセージ用
}

// Position はカーソル位置を表す
type Position struct {
	Row, Col int
}

// NewUI は新しいUIインスタンスを作成する
func NewUI(rows, cols int, eventManager *events.EventManager) *UI {
	ui := &UI{
		screenRows:     rows,
		screenCols:     cols,
		lastColOffset:  0,
		eventManager:   eventManager,
		needsRefresh:   false,
		updateQueue:    make([]UpdateRequest, 0),
		batchMode:      false,
		pendingUpdates: make(map[UpdateArea]UpdateRequest),
		scrollState: ScrollState{
			smoothScroll: true,
			scrollSteps:  3,
		},
		lineState: LineUpdateState{
			dirty:    make(map[int]bool),
			forceAll: true, // 初期表示時に全行を更新対象とする
		},
		message:      "",
		messageArgs:  make([]interface{}, 0),
		messageTime:  0,
		debugMessage: "",
	}

	// バッファイベントを購読してUI更新を最適化
	eventManager.Subscribe(events.BufferEventType, ui.handleBufferEvent)

	return ui
}

// handleBufferEvent はバッファイベントを処理する
func (ui *UI) handleBufferEvent(event events.Event) {
	if bufferEvent, ok := event.(*events.BufferEvent); ok {
		ui.BeginBatchUpdate()
		defer ui.EndBatchUpdate()

		switch bufferEvent.SubType {
		case events.BufferContentChanged:
			if data, ok := bufferEvent.Data.(events.BufferChangeData); ok {
				ui.QueueUpdate(AreaLine, MediumPriority, events.EditorUpdateData{
					Lines:    data.AffectedLines,
					ForceAll: true, // 全行を更新対象とする
				})
			}
		case events.BufferCursorMoved:
			if pos, ok := bufferEvent.Data.(events.Position); ok {
				ui.QueueUpdate(AreaCursor, HighPriority, pos)
				// カーソル移動時も全行を更新対象とする
				ui.QueueUpdate(AreaLine, MediumPriority, events.EditorUpdateData{
					ForceAll: true,
				})
			}
		case events.BufferStructuralChange:
			ui.QueueUpdate(AreaFull, MediumPriority, nil)
		}

		if bufferEvent.HasChanges() {
			ui.QueueUpdate(AreaStatus, LowPriority, nil)
		}
	}
}

// handleUIEvent はUIイベントを処理する
func (ui *UI) handleUIEvent(event *events.UIEvent) {
	switch event.SubType {
	case events.UIRefresh:
		ui.QueueUpdate(AreaFull, HighPriority, nil)
	case events.UIScroll:
		if data, ok := event.Data.(events.ScrollData); ok {
			ui.BeginBatchUpdate()
			ui.handleScrollEvent(data)
			ui.QueueUpdate(AreaFull, MediumPriority, nil)
			ui.EndBatchUpdate()
		}
	case events.UIStatusMessage:
		if data, ok := event.Data.(events.StatusMessageData); ok {
			ui.QueueUpdate(AreaMessage, LowPriority, data)
		}
	case events.UIEditorPartialRefresh:
		if data, ok := event.Data.(events.EditorUpdateData); ok {
			ui.QueueUpdate(AreaLine, MediumPriority, data)
		}
	case events.UICursorUpdate:
		if pos, ok := event.Data.(events.Position); ok {
			ui.QueueUpdate(AreaCursor, HighPriority, pos)
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
		ui.lineState.forceAll = true
		ui.needsRefresh = true
		return
	}

	// 指定された行のみを更新対象としてマーク
	for _, line := range data.Lines {
		ui.lineState.dirty[line] = true
	}
	ui.needsRefresh = true
}

// handleCursorUpdate はカーソル位置の更新を処理する
func (ui *UI) handleCursorUpdate(pos events.Position) {
	ui.updateCursorPosition(pos)
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
	if !ui.scrollState.smoothScroll {
		// スムーズスクロールが無効な場合は即座に更新
		ui.lastColOffset = data.ColOffset
		ui.needsRefresh = true
		return
	}

	ui.scrollState.targetColOffset = data.ColOffset
	ui.scrollState.currentColOffset = ui.lastColOffset

	// スクロール距離を計算
	distance := ui.scrollState.targetColOffset - ui.scrollState.currentColOffset
	if distance == 0 {
		return
	}

	// スクロールステップごとに更新をキュー
	stepSize := distance / ui.scrollState.scrollSteps
	if stepSize == 0 {
		stepSize = distance / Abs(distance) // 最小1ステップ
	}

	ui.BeginBatchUpdate()
	for step := 1; step <= ui.scrollState.scrollSteps; step++ {
		newOffset := ui.scrollState.currentColOffset + stepSize
		if step == ui.scrollState.scrollSteps {
			newOffset = ui.scrollState.targetColOffset // 最後のステップで確実に目標位置に
		}

		ui.lastColOffset = newOffset
		ui.QueueUpdate(AreaFull, MediumPriority, nil)
	}
	ui.EndBatchUpdate()
}

// SetSmoothScroll はスムーズスクロールの有効/無効を設定する
func (ui *UI) SetSmoothScroll(enabled bool) {
	ui.scrollState.smoothScroll = enabled
}

// SetScrollSteps はスムーズスクロールのステップ数を設定する
func (ui *UI) SetScrollSteps(steps int) {
	if steps < 1 {
		steps = 1
	}
	ui.scrollState.scrollSteps = steps
}

// abs は整数の絶対値を返す
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// markLinesForUpdate は更新が必要な行をマークする
func (ui *UI) markLinesForUpdate(lines []int) {
	// 部分更新のための行管理を実装
	ui.needsRefresh = true // 現時点では全体更新にフォールバック
}

// updateCursorPosition はカーソル位置を更新する
func (ui *UI) updateCursorPosition(pos events.Position) {
	// カーソル位置の更新をトリガー
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

	event := events.NewUIEvent(events.UIRefresh, struct{ FullRefresh bool }{FullRefresh: fullRefresh})
	ui.eventManager.Publish(event)
}

// publishCursorUpdateEvent はカーソル更新イベントを発行する
func (ui *UI) publishCursorUpdateEvent(pos events.Position) {
	if ui.eventManager == nil {
		return
	}

	event := events.NewUIEvent(events.UICursorUpdate, pos)
	ui.eventManager.Publish(event)
}

// GetEventManager はイベントマネージャーを返す
func (ui *UI) GetEventManager() *events.EventManager {
	return ui.eventManager
}

// clearBuffer は出力バッファをクリアする
func (ui *UI) clearBuffer() {
	ui.buffer.Reset()
}

// write は出力をターミナルに書き込む
func (ui *UI) write(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// moveCursor はカーソルを指定位置に移動する
func (ui *UI) moveCursor(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row+1, col+1)
}

// clearLine は現在の行をクリアする
func (ui *UI) clearLine() string {
	return "\x1b[2K"
}

// getScreenPosition はバッファ上の位置から画面上の位置を計算する
func (ui *UI) getScreenPosition(cursor Position, buffer *Buffer, rowOffset, colOffset int) (int, int) {
	// 行番号の調整：エディタ領域内に収める
	screenY := cursor.Row - rowOffset
	if screenY < 0 {
		screenY = 0
	} else if screenY >= ui.screenRows-2 { // ステータスバーとメッセージバーの2行分を考慮
		screenY = ui.screenRows - 3
	}

	// 列位置の調整（文字の表示幅を考慮）
	row := buffer.getRow(cursor.Row)
	var screenX int
	if row != nil {
		// カーソル位置までの表示幅を計算
		screenX = row.OffsetToScreenPosition(cursor.Col) - colOffset
		if screenX < 0 {
			screenX = 0
		} else if screenX >= ui.screenCols {
			screenX = ui.screenCols - 1
		}
	}

	return screenX, screenY
}

// setCursor は画面上のカーソル位置を設定する
func (ui *UI) setCursor(x, y int) error {
	if x < 0 || y < 0 || x >= ui.screenCols || y >= ui.screenRows {
		return fmt.Errorf("invalid cursor position: (%d, %d)", x, y)
	}
	ui.buffer.WriteString(ui.moveCursor(y, x))
	return nil
}

// drawStatusBar はステータスバーを描画する
func (ui *UI) drawStatusBar(buffer *Buffer, filename string) error {
	status := filename
	if status == "" {
		status = "[No Name]"
	}
	if buffer.IsDirty() {
		status += " [+]"
	}
	line := "\x1b[7m" + ui.padLine(status) + "\x1b[m\r\n"
	ui.buffer.WriteString(line)
	return nil
}

// drawRows は編集領域を描画する
func (ui *UI) drawRows(buffer *Buffer, rowOffset, colOffset int) error {
	for y := 0; y < ui.screenRows-2; y++ {
		filerow := y + rowOffset
		ui.buffer.WriteString("\x1b[2K") // 各行をクリア

		// ファイル内の有効な行の場合
		if filerow < buffer.GetLineCount() {
			row := buffer.getRow(filerow)
			if row != nil {
				ui.buffer.WriteString(ui.drawTextRow(row, colOffset))
			}
		} else {
			// ファイルの終端以降は空行を表示
			ui.buffer.WriteString(ui.drawEmptyRow(y, buffer.GetLineCount()))
		}
		ui.buffer.WriteString("\r\n")
	}

	// 更新完了後、状態をリセット
	if ui.lineState.forceAll {
		ui.lineState.forceAll = false
		ui.lineState.dirty = make(map[int]bool)
	}

	return nil
}

// RefreshScreen はエディタの画面を更新する
func (ui *UI) RefreshScreen(buffer *Buffer, filename string, rowOffset, colOffset int) error {
	// 既存のバッファをクリア
	ui.clearBuffer()

	// 画面クリアとカーソルを原点に移動
	ui.buffer.WriteString(ui.clearScreen())
	ui.buffer.WriteString(ui.moveCursorToHome())

	// メインコンテンツの描画
	if err := ui.drawRows(buffer, rowOffset, colOffset); err != nil {
		return err
	}

	// ステータスバーの描画
	if err := ui.drawStatusBar(buffer, filename); err != nil {
		return err
	}

	// メッセージバーの描画
	if err := ui.drawMessageBar(); err != nil {
		return err
	}

	// カーソル位置の設定（画面バッファに追加）
	cursor := buffer.GetCursor()
	screenX, screenY := ui.getScreenPosition(Position{Row: cursor.Y, Col: cursor.X}, buffer, rowOffset, colOffset)
	ui.buffer.WriteString(ui.moveCursor(screenY, screenX))

	// バッファの内容を一括で画面に反映
	return ui.write(ui.buffer.String())
}

// Flush は保留中の更新を即座に画面に反映する
func (ui *UI) Flush() error {
	return ui.write(ui.buffer.String())
}

// drawMessageBar はメッセージバーを描画する
func (ui *UI) drawMessageBar() error {
	// カーソルを最下行に移動
	ui.buffer.WriteString(ui.moveCursor(ui.screenRows-1, 0))

	// 行をクリア
	ui.buffer.WriteString(ui.clearLine())

	// デバッグメッセージがある場合は優先して表示
	if ui.debugMessage != "" {
		ui.buffer.WriteString(ui.debugMessage)
		return nil
	}

	// メッセージを表示（5秒経過したら消去）
	if ui.message != "" && time.Now().Unix()-ui.messageTime < 5 {
		if len(ui.messageArgs) > 0 {
			fmt.Fprintf(&ui.buffer, ui.message, ui.messageArgs...)
		} else {
			ui.buffer.WriteString(ui.message)
		}
	} else {
		ui.message = ""
		ui.messageArgs = make([]interface{}, 0)
	}

	return nil
}

// SetMessage はステータスメッセージを設定する
func (ui *UI) SetMessage(format string, args ...interface{}) {
	ui.message = format
	ui.messageArgs = make([]interface{}, len(args))
	copy(ui.messageArgs, args)
	ui.messageTime = time.Now().Unix()

	// メッセージ更新イベントを発行
	if ui.eventManager != nil {
		event := events.NewUIEvent(events.UIStatusMessage, events.StatusMessageData{
			Message: format,
			Args:    args,
		})
		ui.eventManager.Publish(event)
	}

	// メッセージバーの更新をキューに追加
	ui.QueueUpdate(AreaMessage, HighPriority, nil)
	ui.needsRefresh = true
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
	chars := row.runeSlice
	currentPos := 0

	// colOffsetより前の文字をスキップし、画面幅を超えないように描画
	for i, char := range chars {
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
		builder.WriteRune(char)
		currentPos += width
	}

	// 行末までスペースで埋める
	remaining := ui.screenCols - (currentPos - colOffset)
	if remaining > 0 {
		builder.WriteString(strings.Repeat(" ", remaining))
	}

	return builder.String()
}

// BeginBatchUpdate はバッチ更新モードを開始する
func (ui *UI) BeginBatchUpdate() {
	ui.batchMode = true
}

// EndBatchUpdate はバッチ更新モードを終了し、保留中の更新を処理する
func (ui *UI) EndBatchUpdate() {
	ui.batchMode = false
	ui.processPendingUpdates()
}

// QueueUpdate は更新要求をキューに追加する
func (ui *UI) QueueUpdate(area UpdateArea, priority UpdatePriority, data interface{}) {
	update := UpdateRequest{
		area:     area,
		priority: priority,
		data:     data,
	}

	if ui.batchMode {
		// バッチモード中は同じ領域の更新を統合
		if existing, exists := ui.pendingUpdates[area]; exists {
			if update.priority > existing.priority {
				ui.pendingUpdates[area] = update
			}
		} else {
			ui.pendingUpdates[area] = update
		}
	} else {
		// 即時更新
		ui.processUpdate(update)
	}
}

// processPendingUpdates は保留中の更新を処理する
func (ui *UI) processPendingUpdates() {
	// 優先度順に更新を処理
	for priority := HighPriority; priority >= LowPriority; priority-- {
		for area, update := range ui.pendingUpdates {
			if update.priority == priority {
				ui.processUpdate(update)
				delete(ui.pendingUpdates, area)
			}
		}
	}
}

// processUpdate は単一の更新要求を処理する
func (ui *UI) processUpdate(update UpdateRequest) {
	switch update.area {
	case AreaCursor:
		if pos, ok := update.data.(events.Position); ok {
			ui.updateCursorPosition(pos)
		}
	case AreaLine:
		if data, ok := update.data.(events.EditorUpdateData); ok {
			ui.handlePartialRefresh(data)
		}
	case AreaStatus:
		ui.refreshStatusBar()
	case AreaMessage:
		if data, ok := update.data.(events.StatusMessageData); ok {
			ui.handleStatusMessage(data)
		}
	case AreaFull:
		ui.needsRefresh = true
	}
}

// clearUpdateState は更新状態をリセットする
func (ui *UI) clearUpdateState() {
	ui.lineState.forceAll = false
	ui.lineState.dirty = make(map[int]bool)
}

// SetDebugMessage はデバッグ用のメッセージを設定する
func (ui *UI) SetDebugMessage(msg string) {
	ui.debugMessage = msg
}
