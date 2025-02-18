package editor

import "errors"

var (
	// ErrNoFilename はファイル名が設定されていない場合のエラー
	ErrNoFilename = errors.New("no filename specified")
)
