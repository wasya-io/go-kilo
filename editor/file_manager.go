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

	// バッファのダーティフラグをクリア
	if fm.buffer != nil {
		fm.buffer.SetDirty(false)
	}

	// ファイル保存イベントを発行
	if fm.eventManager != nil {
		event := events.NewFileEvent(events.FileSave, filename, content)
		fm.publishEvent(event)
	}

	return nil
}

// SaveCurrentFile は現在のファイルに保存する
func (fm *FileManager) SaveCurrentFile() error {
	if fm.buffer == nil {
		return ErrNoBuffer
	}
	if fm.filename == "" {
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

// publishEvent はファイルイベントを発行する
func (fm *FileManager) publishEvent(event *events.FileEvent) {
	if fm.eventManager != nil {
		fm.eventManager.Publish(event)
	}
}

// HandleSaveRequest はSystemEventのSaveリクエストを処理する
func (fm *FileManager) HandleSaveRequest(event *events.SaveEvent) error {
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
		// エラー時のFileEvent発行
		fileEvent := events.NewFileEvent(events.FileSave, filename, nil)
		fileEvent.SetError(err)
		fm.publishEvent(fileEvent)
		return err
	}

	// 成功時のFileEvent発行
	fileEvent := events.NewFileEvent(events.FileSave, filename, fm.buffer.GetAllLines())
	fm.publishEvent(fileEvent)
	return nil
}
