package reader

import (
	"fmt"
	"os"
)

type StandardKeyReader struct{}

type KeyReader interface {
	Read() ([]byte, int, error)
}

func NewStandardKeyReader() *StandardKeyReader {
	return &StandardKeyReader{}
}

func (kr *StandardKeyReader) Read() ([]byte, int, error) {
	// 標準入力から読み取り
	buf := make([]byte, 32)
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return nil, n, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return nil, n, fmt.Errorf("no input")
	}

	return buf, n, nil
}
