package events

// SubEventType はイベントのサブタイプを表す
type SubEventType string

const (
	// FileOpen はファイルを開く操作
	FileOpen SubEventType = "open"
	// FileSave はファイルを保存する操作
	FileSave SubEventType = "save"
)

// FileEvent はファイル操作イベントを表す
type FileEvent struct {
	BaseEvent
	SubType  SubEventType
	Filename string
	Content  []string
	Error    error
}

// NewFileEvent は新しいFileEventを作成する
func NewFileEvent(subType SubEventType, filename string, content []string) *FileEvent {
	return &FileEvent{
		BaseEvent: BaseEvent{Type: FileEventType},
		SubType:   subType,
		Filename:  filename,
		Content:   content,
	}
}

// SetError はイベントにエラーを設定する
func (e *FileEvent) SetError(err error) {
	e.Error = err
}

// GetContent はファイルの内容を取得する
func (e *FileEvent) GetContent() []string {
	return e.Content
}

// GetFilename はファイル名を取得する
func (e *FileEvent) GetFilename() string {
	return e.Filename
}

// GetError はエラーを取得する
func (e *FileEvent) GetError() error {
	return e.Error
}

// HasError はエラーがあるかどうかを返す
func (e *FileEvent) HasError() bool {
	return e.Error != nil
}
