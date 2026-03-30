package reader

import (
	"fmt"
	"io"
	"os"

	"github.com/wasya-io/go-kilo/app/entity/core"
)

type StandardKeyReader struct {
	logger core.Logger
	in     io.Reader
}

type KeyReader interface {
	Read() ([]byte, int, error)
}

func NewStandardKeyReader(logger core.Logger) *StandardKeyReader {
	return &StandardKeyReader{
		logger: logger,
		in:     os.Stdin,
	}
}

// NewStandardKeyReaderWithInput はテスト用の初期化関数です
func NewStandardKeyReaderWithInput(logger core.Logger, in io.Reader) *StandardKeyReader {
	return &StandardKeyReader{
		logger: logger,
		in:     in,
	}
}

func (kr *StandardKeyReader) Read() ([]byte, int, error) {
	// 指定された io.Reader (通常は os.Stdin) から読み取り
	buf := make([]byte, 4096)
	n, err := kr.in.Read(buf[:])
	if err != nil {
		return nil, n, fmt.Errorf("input error: %v", err)
	}
	if n == 0 {
		return nil, n, fmt.Errorf("no input")
	}

	return buf, n, nil
}
