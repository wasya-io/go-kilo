package editor

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// UI はエディタのユーザーインターフェース関連の処理を担当する
type UI struct {
	screenRows  int
	screenCols  int
	message     string
	messageTime time.Time
}

// NewUI は新しいUI構造体を作成する
func NewUI(rows, cols int) *UI {
	return &UI{
		screenRows: rows,
		screenCols: cols,
	}
}

// RenderScreen はエディタの画面全体を描画する
func (ui *UI) RenderScreen(rows []*Row, filename string, dirty bool, cx, cy, rowOffset, colOffset int) string {
	var b strings.Builder

	b.WriteString(ui.hideCursor())
	b.WriteString(ui.moveCursorToHome())

	// テキストエリアの描画
	ui.renderTextArea(&b, rows, rowOffset, colOffset)

	// ステータスバーの描画
	ui.renderStatusBar(&b, filename, dirty, cy+1, len(rows))

	// メッセージ行の描画
	ui.renderMessageBar(&b)

	// カーソル位置の設定
	screenX := ui.getScreenX(rows, cy, cx, colOffset)
	filerow := cy - rowOffset + 1
	b.WriteString(ui.moveCursor(filerow, screenX))
	b.WriteString(ui.showCursor())

	return b.String()
}

// renderTextArea はテキストエリアを描画する
func (ui *UI) renderTextArea(b *strings.Builder, rows []*Row, rowOffset, colOffset int) {
	for y := 0; y < ui.screenRows-2; y++ {
		filerow := y + rowOffset
		if filerow < len(rows) {
			row := rows[filerow]
			runes := []rune(row.GetContent())
			startIdx := row.ScreenPositionToOffset(colOffset)

			if startIdx >= 0 {
				currentWidth := row.positions[startIdx] - colOffset
				for i := startIdx; i < len(runes); i++ {
					if currentWidth >= ui.screenCols {
						break
					}
					b.WriteString(string(runes[i]))
					currentWidth += row.widths[i]
				}
			}
		} else {
			b.WriteString("~")
		}

		b.WriteString(ui.clearLine())
		b.WriteString("\r\n")
	}
}

// renderStatusBar はステータスバーを描画する
func (ui *UI) renderStatusBar(b *strings.Builder, filename string, dirty bool, currentLine, totalLines int) {
	b.WriteString(ui.invertColors()) // 反転表示
	status := ""
	if filename == "" {
		status = "[No Name]"
	} else {
		status = filename
	}
	if dirty {
		status += " [+]"
	}
	rstatus := fmt.Sprintf("%d/%d", currentLine, totalLines)
	if len(status) > ui.screenCols {
		status = status[:ui.screenCols]
	}
	b.WriteString(status)
	for i := len(status); i < ui.screenCols-len(rstatus); i++ {
		b.WriteString(" ")
	}
	b.WriteString(rstatus)
	b.WriteString(ui.resetColors()) // 反転表示解除
	b.WriteString("\r\n")
}

// renderMessageBar はメッセージバーを描画する
func (ui *UI) renderMessageBar(b *strings.Builder) {
	b.WriteString(ui.clearLine())
	if time.Since(ui.messageTime) < 5*time.Second {
		b.WriteString(ui.message)
	}
}

// SetMessage はステータスメッセージを設定する
func (ui *UI) SetMessage(format string, args ...interface{}) {
	ui.message = fmt.Sprintf(format, args...)
	ui.messageTime = time.Now()
}

// getScreenX はカーソルの画面上のX座標を計算する
func (ui *UI) getScreenX(rows []*Row, cy, cx, colOffset int) int {
	if cy < len(rows) {
		return rows[cy].OffsetToScreenPosition(cx) - colOffset + 1
	}
	return 1
}

// scroll は必要に応じてスクロール位置を更新する
func (ui *UI) scroll(cx, cy int, buffer *Buffer, rowOffset, colOffset *int) {
	// 垂直スクロール
	if cy < *rowOffset {
		*rowOffset = cy
	}
	if cy >= *rowOffset+ui.screenRows-2 {
		*rowOffset = cy - (ui.screenRows - 3)
	}

	// 水平スクロール
	screenX := 0
	if cy < buffer.GetLineCount() {
		row := buffer.GetContent(cy)
		if len(row) > 0 {
			screenX = len([]rune(row[:cx]))
		}
	}

	if screenX < *colOffset {
		*colOffset = screenX
	}
	if screenX >= *colOffset+ui.screenCols {
		*colOffset = screenX - ui.screenCols + 1
	}
}

// RefreshScreen は画面を更新する
func (ui *UI) RefreshScreen(buffer *Buffer, filename string, rowOffset, colOffset int) error {
	cx, cy := buffer.GetCursor()
	ui.scroll(cx, cy, buffer, &rowOffset, &colOffset)

	lines := make([]*Row, buffer.GetLineCount())
	for i := 0; i < buffer.GetLineCount(); i++ {
		content := buffer.GetContent(i)
		lines[i] = NewRow(content)
	}

	output := ui.RenderScreen(lines, filename, buffer.IsDirty(), cx, cy, rowOffset, colOffset)
	_, err := os.Stdout.WriteString(output)
	return err
}

// ANSI エスケープシーケンス関連のメソッド
func (ui *UI) hideCursor() string   { return "\x1b[?25l" }
func (ui *UI) showCursor() string   { return "\x1b[?25h" }
func (ui *UI) clearScreen() string  { return "\x1b[2J" }
func (ui *UI) clearLine() string    { return "\x1b[K" }
func (ui *UI) invertColors() string { return "\x1b[7m" }
func (ui *UI) resetColors() string  { return "\x1b[m" }
func (ui *UI) moveCursor(y, x int) string {
	return fmt.Sprintf("\x1b[%d;%dH", y, x)
}
func (ui *UI) moveCursorToHome() string { return "\x1b[H" }
