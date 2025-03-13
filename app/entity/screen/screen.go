package screen

import (
	"fmt"
	"strings"
	"time"

	"github.com/wasya-io/go-kilo/app/boundary/writer"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
)

const (
	// エスケープシーケンス
	escape             = "\x1b" // ESC
	clearSequence      = "[2J"  // 画面クリア
	clearLineSequence  = "[K"   // 行クリア
	cursorHomeSequence = "[H"   // カーソルを原点に移動
)

type Screen struct {
	scrollOffset position
	rowLines     int
	colLines     int
	builder      contents.Builder
	writer       writer.ScreenWriter
	message      contents.Message
	debugMessage contents.DebugMessage
	cursor       cursor.Cursor
}

type position struct {
	x, y int
}

func NewScreen(
	builder contents.Builder,
	writer writer.ScreenWriter,
	message contents.Message,
	cursor cursor.Cursor,
	rows, cols int,
) *Screen {
	return &Screen{
		scrollOffset: position{x: 0, y: 0},
		rowLines:     rows,
		colLines:     cols,
		builder:      builder,
		writer:       writer,
		message:      message,
		debugMessage: "",
		cursor:       cursor,
	}
}

func (s *Screen) SetRowOffset(y int) {
	s.scrollOffset.y = y
}

func (s *Screen) SetColOffset(x int) {
	s.scrollOffset.x = x
}

func (s *Screen) SetCursor(cursor cursor.Cursor) {
	s.cursor = cursor
}

func (s *Screen) SetCursorPosition(x, y int) {
	s.cursor.SetCursor(x, y)
}

func (s *Screen) GetOffset() (int, int) {
	return s.scrollOffset.x, s.scrollOffset.y
}

func (s *Screen) GetCursor() cursor.Cursor {
	return s.cursor
}

func (s *Screen) GetRowLines() int {
	return s.rowLines
}

func (s *Screen) GetColLines() int {
	return s.colLines
}

// func (s *Screen) MoveCursor(m cursor.Movement, buffer *contents.Contents) {
// 	s.builder.MoveCursor(m, buffer)
// }

// Redraw は画面を再描画する
func (s *Screen) Redraw(buffer *contents.Contents, filename string) error {
	// 既存のバッファをクリア
	s.builder.Clear()

	// 画面クリアとカーソルを原点に移動
	s.builder.Write(escape + clearSequence)
	s.builder.Write(escape + cursorHomeSequence)

	// メインコンテンツの描画
	if err := s.drawRows(buffer, s.scrollOffset.y, s.scrollOffset.x); err != nil {
		return err
	}

	// ステータスバーの描画
	if err := s.drawStatusBar(buffer, filename); err != nil {
		return err
	}

	// メッセージバーの描画
	if err := s.drawMessageBar(); err != nil {
		return err
	}

	// カーソル位置の設定（画面バッファに追加）
	// cursor := ui.GetCursor() // Bufferからの直接参照をUI内部状態の参照に変更
	pos := s.cursor.ToPosition()
	screenX, screenY := s.getScreenPosition(pos.X, pos.Y, buffer, s.scrollOffset.y, s.scrollOffset.x)
	s.builder.Write(fmt.Sprintf("\x1b[%d;%dH", screenY+1, screenX+1))

	// バッファの内容を一括で画面に反映
	return s.writer.Write(s.builder.Build())
}

// Flush は画面バッファを画面に反映する
func (s *Screen) Flush() error {
	return s.writer.Write(s.builder.Build())
}

// SetMessage はステータスメッセージを設定する
func (s *Screen) SetMessage(format string, args ...interface{}) {
	s.message.SetMessage(format, args...)
}

// MoveCursor は指定された方向にカーソルを移動し、必要な更新をキューに追加する
func (s *Screen) MoveCursor(movement cursor.Movement, buffer *contents.Contents) {
	if buffer == nil || buffer.GetLineCount() == 0 {
		return
	}

	pos := s.cursor.ToPosition()
	currentRow := buffer.GetRow(pos.Y)
	if currentRow == nil {
		return
	}

	newCursor := s.calculateNewCursorPosition(movement, buffer, currentRow)
	newPos := newCursor.ToPosition()

	// カーソル位置の更新
	s.cursor.SetCursor(newPos.X, newPos.Y)
}

// calculateNewCursorPosition は新しいカーソル位置を計算する（移動処理のロジックを分離）
func (s *Screen) calculateNewCursorPosition(movement cursor.Movement, buffer *contents.Contents, currentRow *contents.Row) *cursor.StandardCursor {
	newPos := s.cursor.ToPosition() // 現在の位置からコピーを作成

	switch movement {
	case cursor.CursorUp:
		if newPos.Y > 0 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y--
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	case cursor.CursorDown:
		if newPos.Y < buffer.GetLineCount()-1 {
			currentVisualX := currentRow.OffsetToScreenPosition(newPos.X)
			newPos.Y++
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.ScreenPositionToOffset(currentVisualX)
			}
		}
	case cursor.CursorLeft:
		if newPos.X > 0 {
			newPos.X--
		} else if newPos.Y > 0 {
			newPos.Y--
			targetRow := buffer.GetRow(newPos.Y)
			if targetRow != nil {
				newPos.X = targetRow.GetRuneCount()
			}
		}
	case cursor.CursorRight:
		maxX := currentRow.GetRuneCount()
		if newPos.X < maxX {
			newPos.X++
		} else if newPos.Y < buffer.GetLineCount()-1 {
			newPos.Y++
			newPos.X = 0
		}
	case cursor.MouseWheelUp:
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
	case cursor.MouseWheelDown:
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
	newCursor := cursor.NewCursor()
	newCursor.SetCursor(newPos.X, newPos.Y)

	return newCursor
}

// drawMessageBar はメッセージバーを描画する
func (s *Screen) drawMessageBar() error {
	// カーソルを最下行に移動
	s.builder.Write(fmt.Sprintf("\x1b[%d;%dH", s.rowLines-1, 0))

	// 行をクリア
	s.builder.Write(escape + clearLineSequence)

	// デバッグメッセージがある場合は優先して表示
	if s.debugMessage != "" {
		s.builder.Write(string(s.debugMessage))
		return nil
	}

	// メッセージを表示（5秒経過したら消去）
	if s.message.Get() != "" && time.Now().Unix()-s.message.GetTime() < 5 {
		s.builder.Write(s.message.String())
		// if len(s.message.Args) > 0 {
		// 	fmt.Fprintf(&ui.buffer, ui.message, ui.messageArgs...)
		// } else {
		// 	ui.buffer.WriteString(ui.message)
		// }
	} else {
		s.message.Clear()
		// ui.message = ""
		// ui.messageArgs = make([]interface{}, 0)
	}

	return nil
}

// getScreenPosition はバッファ上の位置から画面上の位置を計算する
func (s *Screen) getScreenPosition(x, y int, buffer *contents.Contents, rowOffset, colOffset int) (int, int) {
	// 行番号の調整：エディタ領域内に収める
	screenY := y - rowOffset

	// 列位置の調整（文字の表示幅を考慮）
	row := buffer.GetRow(y)
	var screenX int
	if row != nil {
		// カーソル位置までの表示幅を計算
		screenX = row.OffsetToScreenPosition(x) - colOffset
	}

	return screenX, screenY
}

// drawStatusBar はステータスバーを描画する
func (s *Screen) drawStatusBar(buffer *contents.Contents, filename string) error {
	status := filename
	if status == "" {
		status = "[No Name]"
	}
	if buffer.IsDirty() {
		status += " [+]"
	}
	line := "\x1b[7m" + s.padLine(status) + "\x1b[m\r\n"
	s.builder.Write(line)
	return nil
}

// drawRows は編集領域を描画する
func (s *Screen) drawRows(buffer *contents.Contents, rowOffset, colOffset int) error {
	for y := 0; y < s.rowLines-2; y++ {
		filerow := y + rowOffset
		s.builder.Write("\x1b[2K") // 各行をクリア

		// ファイル内の有効な行の場合
		if filerow < buffer.GetLineCount() {
			row := buffer.GetRow(filerow)
			if row != nil {
				s.builder.Write(s.drawTextRow(row, colOffset))
			}
		} else {
			// ファイルの終端以降は空行を表示
			s.builder.Write(s.drawEmptyRow(y, buffer.GetLineCount()))
		}
		s.builder.Write("\r\n")
	}

	return nil
}

// drawEmptyRow は空行（チルダ）またはウェルカムメッセージを描画
func (s *Screen) drawEmptyRow(y int, totalLines int) string {
	if totalLines == 0 && y == s.rowLines/3 {
		return s.drawWelcomeMessage()
	}
	return "~"
}

// drawWelcomeMessage はウェルカムメッセージを描画
func (s *Screen) drawWelcomeMessage() string {
	// TODO: getにする
	welcome := "Kilo editor -- version 1.0"
	if len(welcome) > s.colLines {
		welcome = welcome[:s.colLines]
	}
	padding := (s.colLines - len(welcome)) / 2
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
func (s *Screen) drawTextRow(row *contents.Row, colOffset int) string {
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
		if currentPos-colOffset >= s.colLines {
			break
		}

		// 文字を描画
		builder.WriteRune(char)
		currentPos += width
	}

	// 行末までスペースで埋める
	remaining := s.colLines - (currentPos - colOffset)
	if remaining > 0 {
		builder.WriteString(strings.Repeat(" ", remaining))
	}

	return builder.String()
}

// clearScreen は画面をクリアする
func (s *Screen) ClearScreen() string {
	return escape + clearSequence
}

// moveCursorToHome はカーソルを原点に移動する
func (s *Screen) MoveCursorToHome() string {
	return escape + cursorHomeSequence
}

// padLine は行を画面幅に合わせてパディングする
func (s *Screen) padLine(line string) string {
	if len(line) > s.colLines {
		return line[:s.colLines]
	}
	return line + strings.Repeat(" ", s.colLines-len(line))
}
