package cursor

import "github.com/wasya-io/go-kilo/app/entity/contents"

type Cursor struct {
	position       position
	latestPosition position
}

func NewCursor() *Cursor {
	return &Cursor{
		position:       newPosition(0, 0),
		latestPosition: newPosition(0, 0),
	}
}

type position struct {
	x, y int
}

// Movement はカーソル移動の種類を表す型
type Movement byte

const (
	CursorUp       Movement = 'A'
	CursorDown     Movement = 'B'
	CursorRight    Movement = 'C'
	CursorLeft     Movement = 'D'
	MouseWheelUp   Movement = 'U' // マウスホイールでの上方向スクロール
	MouseWheelDown Movement = 'V' // マウスホイールでの下方向スクロール
)

func newPosition(x, y int) position {
	return position{x: x, y: y}
}

func (c *Cursor) NewLine() {
	c.latestPosition = c.position
	c.position.x = 0
	c.position.y++
}

func (c *Cursor) ToPosition() contents.Position {
	return contents.Position{
		X: c.position.x,
		Y: c.position.y,
	}
}

func (c *Cursor) SetCursor(x, y int) {
	c.latestPosition = c.position
	c.position = newPosition(x, y)
}

func (c *Cursor) Row() int {
	return c.position.y
}

func (c *Cursor) Col() int {
	return c.position.x
}
