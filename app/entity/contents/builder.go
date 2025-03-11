package contents

import (
	"fmt"
	"strings"
)

type Builder struct {
	buffer strings.Builder
}

func NewBuilder() *Builder {
	return &Builder{
		buffer: strings.Builder{},
	}
}

func (b *Builder) Clear() {
	b.buffer.Reset()
}

func (b *Builder) Write(s string) {
	b.buffer.WriteString(s)
}

// moveCursor はカーソルを指定位置に移動する
func (b *Builder) MoveCursor(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row+1, col+1)
}

func (b *Builder) Build() string {
	return b.buffer.String()
}
