package events

// FileSubEventType はファイル操作のサブタイプを表す
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
		BaseEvent: NewBaseEvent(FileEventType),
		SubType:   subType,
		Filename:  filename,
		Content:   content,
	}
}

// SetError はイベントにエラーを設定する
func (e *FileEvent) SetError(err error) {
	e.Error = err
}
