package cursor

import "github.com/wasya-io/go-kilo/app/entity/contents"

type Cursor interface {
	NewLine()
	ToPosition() contents.Position
	SetCursor(x, y int)
	Row() int
	Col() int
}

type StandardCursor struct {
	position       position
	latestPosition position
}

func NewCursor() *StandardCursor {
	return &StandardCursor{
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

func (c *StandardCursor) NewLine() {
	c.latestPosition = c.position
	c.position.x = 0
	c.position.y++
}

func (c *StandardCursor) ToPosition() contents.Position {
	return contents.Position{
		X: c.position.x,
		Y: c.position.y,
	}
}

func (c *StandardCursor) SetCursor(x, y int) {
	c.latestPosition = c.position
	c.position = newPosition(x, y)
}

func (c *StandardCursor) Row() int {
	return c.position.y
}

func (c *StandardCursor) Col() int {
	return c.position.x
}
