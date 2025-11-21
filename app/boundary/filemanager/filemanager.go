package filemanager

import (
	"errors"
	"os"
	"strings"

	"github.com/wasya-io/go-kilo/app/entity/contents"
)

// StandardFileManager はファイル操作を管理する構造体
type StandardFileManager struct {
	buffer   *contents.Contents
	filename string
}

type FileManager interface {
	OpenFile(filename string) error
	SaveFile(filename string, content []string) error
	SaveCurrentFile() error
	GetFilename() string
	HandleSaveRequest() error
}

// エラー定義
var (
	ErrNoBuffer   = errors.New("no buffer available")
	ErrNoFilename = errors.New("no filename specified")
)

// NewFileManager は新しいFileManagerを作成する
func NewFileManager(buffer *contents.Contents) *StandardFileManager {
	return &StandardFileManager{
		buffer: buffer,
	}
}

// OpenFile は指定されたファイルを開く
func (fm *StandardFileManager) OpenFile(filename string) error {
	content, err := fm.readFile(filename)
	if err != nil {
		return err
	}
	fm.filename = filename
	fm.buffer.LoadContent(content)

	return nil
}

// SaveFile はバッファの内容をファイルに保存する
func (fm *StandardFileManager) SaveFile(filename string, content []string) error {
	if filename == "" {
		return ErrNoFilename
	}

	// ファイルに書き込む
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 内容を書き込む
	for i, line := range content {
		if i > 0 {
			file.WriteString("\n")
		}
		file.WriteString(line)
	}

	// バッファのダーティフラグをクリア
	if fm.buffer != nil {
		fm.buffer.SetDirty(false)
	}

	// 保存に成功したら、管理しているファイル名を更新する
	fm.filename = filename

	return nil
}

// SaveCurrentFile は現在のファイルに保存する
func (fm *StandardFileManager) SaveCurrentFile() error {
	if fm.buffer == nil {
		return ErrNoBuffer
	}
	if fm.filename == "" {
		return ErrNoFilename
	}
	return fm.SaveFile(fm.filename, fm.buffer.GetAllLines())
}

// GetFilename は現在開いているファイル名を返す
func (fm *StandardFileManager) GetFilename() string {
	return fm.filename
}

// readFile はファイルを読み込む
func (fm *StandardFileManager) readFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	return lines, nil
}

// HandleSaveRequest はSystemEventのSaveリクエストを処理する
func (fm *StandardFileManager) HandleSaveRequest() error {
	// 保存前の状態確認
	if fm.buffer == nil {
		return ErrNoBuffer
	}

	// ファイル名の検証（FileManagerが管理するファイル名を優先）
	filename := fm.filename
	if filename == "" {
		return ErrNoFilename
	}

	// 保存処理を実行
	if err := fm.SaveCurrentFile(); err != nil {
		return err
	}

	return nil
}
