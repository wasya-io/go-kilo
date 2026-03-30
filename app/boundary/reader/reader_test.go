package reader

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wasya-io/go-kilo/app/boundary/logger"
)

func TestStandardKeyReader_ReadLongJapaneseInput(t *testing.T) {
	l := logger.New(true)

	// 25文字の日本語（75バイト）を入力データとする
	inputStr := "アイウエオアイウエオアイウエオアイウエオアイウエオ"
	in := bytes.NewReader([]byte(inputStr))

	kr := NewStandardKeyReaderWithInput(l, in)

	buf, n, err := kr.Read()
	assert.NoError(t, err)

	// バッファサイズが拡張されたため、75バイトすべて1回のReadで取得できるはず
	assert.Equal(t, 75, n, "Should read all 75 bytes of the input at once")
	assert.Equal(t, inputStr, string(buf[:n]), "The read string should exactly match the input")
}
