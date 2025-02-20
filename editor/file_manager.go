package editor

import (
	"os"
	"strings"

	"github.com/wasya-io/go-kilo/editor/events"
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

	event := events.NewFileEvent(events.FileOpen, filename, content)
	fm.publishEvent(event)
	return nil
}

// SaveFile は現在のバッファの内容をファイルに保存する
func (fm *FileManager) SaveFile() error {
	if fm.filename == "" {
		return ErrNoFilename
	}

	content := fm.buffer.GetAllContent()
	err := fm.writeFile(fm.filename, content)
	if err != nil {
		event := events.NewFileEvent(events.FileSave, fm.filename, content)
		event.SetError(err)
		fm.publishEvent(event)
		return err
	}

	event := events.NewFileEvent(events.FileSave, fm.filename, content)
	fm.publishEvent(event)
	return nil
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
