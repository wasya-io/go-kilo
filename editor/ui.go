package editor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core/mathutil"
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

// Cursor はカーソル位置を管理する構造体
type Cursor struct {
	X, Y int
}

// UI は画面表示を管理する構造体
type UI struct {
	screenRows     int
	screenCols     int
	offset         Offset
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
	cursor         Cursor          // Bufferで定義されているCursor型を使用
	lastCursorPos  Cursor
}

type Offset struct {
	Row, Col int
}

// NewUI は新しいUIインスタンスを作成する
func NewUI(rows, cols int, eventManager *events.EventManager) *UI {
	ui := &UI{
		screenRows:     rows,
		screenCols:     cols,
		offset:         Offset{Row: 0, Col: 0},
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
		}, message: "",
		messageArgs:   make([]interface{}, 0),
		messageTime:   0,
		debugMessage:  "",
		cursor:        Cursor{X: 0, Y: 0},
		lastCursorPos: Cursor{X: 0, Y: 0},
	}

	// バッファイベントを購読してUI更新を最適化
	eventManager.Subscribe(events.BufferEventType, ui.handleBufferEvent)

	return ui
}

// HandleNewLine は改行時のカーソル位置を管理する
func (ui *UI) HandleNewLine() {
	ui.lastCursorPos = ui.cursor
	ui.cursor.Y++
	ui.cursor.X = 0
	// カーソル位置の更新を画面更新キューに追加
	ui.QueueUpdate(AreaCursor, HighPriority, nil)
}

// handleBufferEvent はバッファイベントを処理する
func (ui *UI) handleBufferEvent(event events.Event) {
	if bufferEvent, ok := event.(*events.BufferEvent); ok {
		ui.BeginBatchUpdate()
		defer ui.EndBatchUpdate()

		// バッファの変更タイプに応じた更新を行う
		switch bufferEvent.SubType {
		case events.BufferContentChanged:
			if bufferEvent.GetOperation() == events.BufferDeleteChar {
				// 削除操作時の更新は既にDeleteChar処理で行われている
				ui.QueueUpdate(AreaFull, MediumPriority, nil)
			} else {
				// 変更された行のみを更新
				ui.QueueUpdate(AreaLine, MediumPriority, events.EditorUpdateData{
					Lines:    bufferEvent.GetAffectedLines(),
					ForceAll: false,
				})
			}
		case events.BufferStructuralChange:
			// 改行などの構造的変更は全体更新
			ui.QueueUpdate(AreaFull, MediumPriority, nil)
		}

		// 変更があった場合はステータス更新も行う
		if bufferEvent.HasChanges() {
			ui.QueueUpdate(AreaStatus, LowPriority, nil)
		}
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
		stepSize = distance / mathutil.Abs(distance) // 最小1ステップ
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

// UpdateOffsetRow は行オフセットを更新する
func (ui *UI) UpdateOffsetRow(row int) {
	ui.offset.Row = row
}

// UpdateOffsetCol は列オフセットを更新する
func (ui *UI) UpdateOffsetCol(col int) {
	ui.offset.Col = col
}

// GetOffset は現在のオフセットを返す
func (ui *UI) GetOffset() Offset {
	return ui.offset
}

// refreshStatusBar はステータスバーを更新する
func (ui *UI) refreshStatusBar() {
	ui.needsRefresh = true
}

// refreshMessageBar はメッセージバーを更新する
func (ui *UI) refreshMessageBar() {
	ui.needsRefresh = true
}

// publishCursorUpdateEvent はカーソル更新イベントを発行する
func (ui *UI) publishCursorUpdateEvent() {
	if ui.eventManager == nil {
		return
	}

	event := events.NewCursorUpdateEvent(events.CursorPosition{
		X: ui.cursor.X,
		Y: ui.cursor.Y,
	})
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
func (ui *UI) getScreenPosition(cursor events.Position, buffer *contents.Contents, rowOffset, colOffset int) (int, int) {
	// 行番号の調整：エディタ領域内に収める
	screenY := cursor.Y - rowOffset
	if screenY < 0 {
		screenY = 0
	} else if screenY >= ui.screenRows-2 { // ステータスバーとメッセージバーの2行分を考慮
		screenY = ui.screenRows - 3
	}

	// 列位置の調整（文字の表示幅を考慮）
	row := buffer.GetRow(cursor.Y)
	var screenX int
	if row != nil {
		// カーソル位置までの表示幅を計算
		screenX = row.OffsetToScreenPosition(cursor.X) - colOffset
		if screenX < 0 {
			screenX = 0
		} else if screenX >= ui.screenCols {
			screenX = ui.screenCols - 1
		}
	}

	return screenX, screenY
}

// drawStatusBar はステータスバーを描画する
func (ui *UI) drawStatusBar(buffer *contents.Contents, filename string) error {
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
func (ui *UI) drawRows(buffer *contents.Contents, rowOffset, colOffset int) error {
	for y := 0; y < ui.screenRows-2; y++ {
		filerow := y + rowOffset
		ui.buffer.WriteString("\x1b[2K") // 各行をクリア

		// ファイル内の有効な行の場合
		if filerow < buffer.GetLineCount() {
			row := buffer.GetRow(filerow)
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
func (ui *UI) RefreshScreen(buffer *contents.Contents, filename string, rowOffset, colOffset int) error {
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
	cursor := ui.GetCursor() // Bufferからの直接参照をUI内部状態の参照に変更
	screenX, screenY := ui.getScreenPosition(events.Position{Y: cursor.Y, X: cursor.X}, buffer, rowOffset, colOffset)
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
func (ui *UI) drawTextRow(row *contents.Row, colOffset int) string {
	if row == nil {
		return ""
	}

	var builder strings.Builder
	chars := row.GetRunes()
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
	ui.processAreaUpdates(update.area, update.data)
}

// カーソル位置の更新処理を分離
func (ui *UI) processAreaUpdates(area UpdateArea, data interface{}) {
	switch area {
	case AreaCursor:
		// カーソル更新の視覚的な処理のみを行う
		ui.needsRefresh = true
	case AreaLine:
		if data, ok := data.(events.EditorUpdateData); ok {
			ui.handlePartialRefresh(data)
		}
	case AreaStatus:
		ui.refreshStatusBar()
	case AreaMessage:
		if data, ok := data.(events.StatusMessageData); ok {
			ui.handleStatusMessage(data)
		}
	case AreaFull:
		ui.needsRefresh = true
	}
}

// SetDebugMessage はデバッグ用のメッセージを設定する
func (ui *UI) SetDebugMessage(msg string) {
	ui.debugMessage = msg
}

// GetCursor はカーソル位置を返す
func (ui *UI) GetCursor() Cursor {
	return ui.cursor
}

// SetCursor はカーソル位置を設定し、必要な更新をキューに追加する
func (ui *UI) SetCursor(x, y int) {
	if ui.cursor.X == x && ui.cursor.Y == y {
		return // 位置が変わらない場合は更新しない
	}
	ui.lastCursorPos = ui.cursor
	ui.cursor = Cursor{X: x, Y: y}
	// カーソルの変更を通知
	ui.publishCursorUpdateEvent()
	// 画面更新をキューに追加
	ui.QueueUpdate(AreaCursor, HighPriority, nil)
}

// MoveCursor は指定された方向にカーソルを移動し、必要な更新をキューに追加する
func (ui *UI) MoveCursor(movement CursorMovement, buffer *contents.Contents) {
	if buffer == nil || buffer.GetLineCount() == 0 {
		return
	}

	currentRow := buffer.GetRow(ui.cursor.Y)
	if currentRow == nil {
		return
	}

	newPos := ui.calculateNewCursorPosition(movement, buffer, currentRow)
	if newPos != ui.cursor {
		ui.lastCursorPos = ui.cursor
		ui.cursor = newPos
		// カーソルの変更を画面更新キューに追加
		ui.QueueUpdate(AreaCursor, HighPriority, nil)
	}
}

// calculateNewCursorPosition は新しいカーソル位置を計算する（移動処理のロジックを分離）
func (ui *UI) calculateNewCursorPosition(movement CursorMovement, buffer *contents.Contents, currentRow *contents.Row) Cursor {
	newPos := ui.cursor // 現在の位置からコピーを作成

	switch movement {
	case CursorUp:
		if newPos.Y > 0 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y--
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	case CursorDown:
		if newPos.Y < buffer.GetLineCount()-1 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y++
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	case CursorLeft:
		if newPos.X > 0 {
			newPos.X--
		} else if newPos.Y > 0 {
			newPos.Y--
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.GetRuneCount()
			}
		}
	case CursorRight:
		maxX := currentRow.GetRuneCount()
		if newPos.X < maxX {
			newPos.X++
		} else if newPos.Y < buffer.GetLineCount()-1 {
			newPos.Y++
			newPos.X = 0
		}
	case MouseWheelUp:
		targetY := newPos.Y - 3
		if targetY < 0 {
			targetY = 0
		}
		if newPos.Y > 0 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y = targetY
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	case MouseWheelDown:
		targetY := newPos.Y + 3
		if targetY >= buffer.GetLineCount() {
			targetY = buffer.GetLineCount() - 1
		}
		if newPos.Y < buffer.GetLineCount()-1 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y = targetY
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	}

	return newPos
}
