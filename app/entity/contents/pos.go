package contents

// Position はバッファ内の位置を表す（編集操作の位置を示すために使用）
type Position struct {
	X, Y int
}

func NewPosition(x, y int) Position {
	return Position{X: x, Y: y}
}
