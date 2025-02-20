package editor

import (
	"errors"
	"os"
	"strings"

	"github.com/wasya-io/go-kilo/editor/events"
)

// エラー定義
var (
	ErrNoBuffer = errors.New("no buffer available")
)

// FileManager はファイル操作を管理する構造体
type FileManager struct {
	buffer       *Buffer
	filename     string
	eventManager *events.EventManager
}

// NewFileManager は新しいFileManagerを作成する
func NewFileManager(buffer *Buffer, eventManager *events.EventManager) *FileManager {
	return &FileManager{
		buffer:       buffer,
		eventManager: eventManager,
	}
}

// OpenFile は指定されたファイルを開く
func (fm *FileManager) OpenFile(filename string) error {
	content, err := fm.readFile(filename)
	if err != nil {
		event := events.NewFileEvent(events.FileOpen, filename, nil)
		event.SetError(err)
		fm.publishEvent(event)
		return err
	}

	fm.filename = filename
	fm.buffer.LoadContent(content)
	fm.buffer.Filename = filename // バッファのファイル名も更新

	event := events.NewFileEvent(events.FileOpen, filename, content)
	fm.publishEvent(event)
	return nil
}

// SaveFile はバッファの内容をファイルに保存する
func (fm *FileManager) SaveFile(filename string, content []string) error {
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

	// バッファのファイル名を更新
	if fm.buffer != nil {
		fm.buffer.Filename = filename
		fm.buffer.SetDirty(false) // ダーティフラグをクリア
	}

	// ファイル保存イベントを発行
	if fm.eventManager != nil {
		event := events.NewFileEvent(events.FileSave, filename, content)
		fm.eventManager.Publish(event)
	}

	return nil
}

// SaveCurrentFile は現在のファイルに保存する
func (fm *FileManager) SaveCurrentFile() error {
	if fm.buffer == nil {
		return ErrNoBuffer
	}
	if fm.filename == "" { // FileManagerのfilenameフィールドを使用
		return ErrNoFilename
	}
	return fm.SaveFile(fm.filename, fm.buffer.GetAllLines())
}

// GetFilename は現在開いているファイル名を返す
func (fm *FileManager) GetFilename() string {
	return fm.filename
}

// readFile はファイルを読み込む
func (fm *FileManager) readFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	return lines, nil
}

// writeFile はファイルに書き込む
func (fm *FileManager) writeFile(filename string, lines []string) error {
	data := []byte(strings.Join(lines, "\n"))
	return os.WriteFile(filename, data, 0644)
}

// publishEvent はファイルイベントを発行する
func (fm *FileManager) publishEvent(event *events.FileEvent) {
	if fm.eventManager != nil {
		fm.eventManager.Publish(event)
	}
}
