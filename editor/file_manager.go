package editor

import "github.com/wasya-io/go-kilo/editor/events"

// FileManager はファイル操作を管理する構造体
type FileManager struct {
	storage      Storage
	filename     string
	buffer       *Buffer
	eventManager *events.EventManager
}

// NewFileManager は新しいFileManagerインスタンスを作成する
func NewFileManager(buffer *Buffer, eventManager *events.EventManager) *FileManager {
	return &FileManager{
		storage:      NewFileStorage(),
		buffer:       buffer,
		eventManager: eventManager,
	}
}

// OpenFile は指定されたファイルを読み込む
func (fm *FileManager) OpenFile(filename string) error {
	fm.filename = filename
	lines, err := fm.storage.Load(filename)
	if err != nil {
		fileEvent := events.NewFileEvent(events.FileOpen, filename, nil)
		fileEvent.SetError(err)
		fm.eventManager.Publish(fileEvent)
		return err
	}

	fm.buffer.LoadContent(lines)

	// 成功イベントを発行
	fileEvent := events.NewFileEvent(events.FileOpen, filename, lines)
	fm.eventManager.Publish(fileEvent)
	return nil
}

// SaveFile は現在の内容をファイルに保存する
func (fm *FileManager) SaveFile() error {
	if fm.filename == "" {
		return ErrNoFilename
	}

	content := fm.buffer.GetAllContent()
	err := fm.storage.Save(fm.filename, content)
	if err != nil {
		fileEvent := events.NewFileEvent(events.FileSave, fm.filename, content)
		fileEvent.SetError(err)
		fm.eventManager.Publish(fileEvent)
		return err
	}

	fm.buffer.SetDirty(false)

	// 成功イベントを発行
	fileEvent := events.NewFileEvent(events.FileSave, fm.filename, content)
	fm.eventManager.Publish(fileEvent)
	return nil
}

// GetFilename は現在開いているファイル名を返す
func (fm *FileManager) GetFilename() string {
	return fm.filename
}
