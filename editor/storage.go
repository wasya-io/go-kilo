package editor

import (
	"os"
	"strings"
)

// Storage はファイル操作のインターフェースを定義します
type Storage interface {
	Load(filename string) ([]string, error)
	Save(filename string, content []string) error
}

// FileStorage は実際のファイルシステムを使用した Storage の実装です
type FileStorage struct{}

// NewFileStorage は新しい FileStorage インスタンスを作成します
func NewFileStorage() *FileStorage {
	return &FileStorage{}
}

// Load はファイルから内容を読み込みます
func (fs *FileStorage) Load(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(content), "\n"), "\n"), nil
}

// Save はコンテンツをファイルに保存します
func (fs *FileStorage) Save(filename string, content []string) error {
	return os.WriteFile(filename, []byte(strings.Join(content, "\n")), 0644)
}
