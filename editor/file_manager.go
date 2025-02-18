package editor

// FileManager はファイル操作を管理する構造体
type FileManager struct {
	storage  Storage
	filename string
	buffer   *Buffer
}

// NewFileManager は新しいFileManagerインスタンスを作成する
func NewFileManager(buffer *Buffer) *FileManager {
	return &FileManager{
		storage: NewFileStorage(),
		buffer:  buffer,
	}
}

// OpenFile は指定されたファイルを読み込む
func (fm *FileManager) OpenFile(filename string) error {
	fm.filename = filename
	lines, err := fm.storage.Load(filename)
	if err != nil {
		return err
	}
	fm.buffer.LoadContent(lines)
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
		return err
	}

	fm.buffer.SetDirty(false)
	return nil
}

// GetFilename は現在開いているファイル名を返す
func (fm *FileManager) GetFilename() string {
	return fm.filename
}
