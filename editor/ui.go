package editor

import (
	"fmt"
	"os"
	"strings"
)

// UI は画面表示を管理する構造体
type UI struct {
	screenRows    int
	screenCols    int
	message       string
	messageArgs   []interface{}
	lastColOffset int // 前回のcolOffsetを保存
}

// NewUI は新しいUIインスタンスを作成する
func NewUI(rows, cols int) *UI {
	return &UI{
		screenRows:    rows,
		screenCols:    cols,
		lastColOffset: 0,
	}
}

// RefreshScreen は画面を更新する
func (ui *UI) RefreshScreen(buffer *Buffer, filename string, rowOffset, colOffset int) error {
	var builder strings.Builder

	// カーソルを非表示にする
	builder.WriteString("\x1b[?25l")

	// 現在のカーソル位置を保存
	builder.WriteString("\x1b[s")

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
		runePos := row.OffsetToScreenPosition(x)
		screenX = runePos - colOffset + 1
	}

	// カーソル位置の範囲チェック
	if screenX < 1 {
		screenX = 1
	}
	if screenY < 1 {
		screenY = 1
	}
	if screenX > ui.screenCols {
		screenX = ui.screenCols
	}
	if screenY > ui.screenRows-2 {
		screenY = ui.screenRows - 2
	}

	// カーソルを新しい位置に移動（画面の端を考慮）
	if screenY >= 1 && screenY <= ui.screenRows-2 {
		builder.WriteString(fmt.Sprintf("\x1b[%d;%dH", screenY, screenX))
	}

	// カーソルを再表示
	builder.WriteString("\x1b[?25h")

	// 全ての変更を一度に出力
	_, err := os.Stdout.WriteString(builder.String())
	return err
}

// SetMessage はステータスメッセージを設定する
func (ui *UI) SetMessage(format string, args ...interface{}) {
	ui.message = format
	ui.messageArgs = args
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
	runes := []rune(row.GetContent())
	visualPos := 0 // 画面上の表示位置

	// 各文字の表示位置を計算しながら描画
	for i := 0; i < len(runes); i++ {
		width := row.GetRuneWidth(i)

		// colOffsetより前の文字はスキップ
		if visualPos < colOffset {
			visualPos += width
			continue
		}

		// 画面幅を超える場合は描画終了
		if visualPos-colOffset >= ui.screenCols {
			break
		}

		// 文字を描画
		builder.WriteRune(runes[i])
		visualPos += width
	}

	return builder.String()
}
