package editor

import "errors"

// エラー定義
var (
	ErrNoBuffer   = errors.New("no buffer available")
	ErrNoFilename = errors.New("no filename specified")
)
