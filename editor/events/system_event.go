package events

import "time"

// SystemEventSubType はシステムイベントのサブタイプを表す
type SystemEventSubType string

const (
	SystemSave   SystemEventSubType = "save"
	SystemQuit   SystemEventSubType = "quit"
	SystemStatus SystemEventSubType = "status"
)

// SystemEvent はシステムイベントのインターフェースを定義
type SystemEvent interface {
	Event
	GetSystemType() SystemEventSubType
}

// BaseSystemEvent はシステムイベントの基本実装
type BaseSystemEvent struct {
	BaseEvent
	SubType SystemEventSubType
}

func (e *BaseSystemEvent) GetSystemType() SystemEventSubType {
	return e.SubType
}

// SaveEvent はファイル保存イベントを表す
type SaveEvent struct {
	BaseSystemEvent
	Filename string
	Force    bool
}

// NewSaveEvent は新しいSaveEventを作成する
func NewSaveEvent(filename string, force bool) *SaveEvent {
	return &SaveEvent{
		BaseSystemEvent: BaseSystemEvent{
			BaseEvent: BaseEvent{
				Type:     SystemEventType,
				Time:     time.Now(),
				Priority: HighPriority,
			},
			SubType: SystemSave,
		},
		Filename: filename,
		Force:    force,
	}
}

// QuitEvent は終了イベントを表す
type QuitEvent struct {
	BaseSystemEvent
	Force      bool
	SaveNeeded bool
}

// NewQuitEvent は新しいQuitEventを作成する
func NewQuitEvent(saveNeeded bool, force bool) *QuitEvent {
	return &QuitEvent{
		BaseSystemEvent: BaseSystemEvent{
			BaseEvent: BaseEvent{
				Type:     SystemEventType,
				Time:     time.Now(),
				Priority: HighPriority,
			},
			SubType: SystemQuit,
		},
		Force:      force,
		SaveNeeded: saveNeeded,
	}
}

// StatusEvent はステータスメッセージイベントを表す
type StatusEvent struct {
	BaseSystemEvent
	Message string
	Args    []interface{}
}

// NewStatusEvent は新しいStatusEventを作成する
func NewStatusEvent(message string, args ...interface{}) *StatusEvent {
	return &StatusEvent{
		BaseSystemEvent: BaseSystemEvent{
			BaseEvent: BaseEvent{
				Type:     SystemEventType,
				Time:     time.Now(),
				Priority: MediumPriority,
			},
			SubType: SystemStatus,
		},
		Message: message,
		Args:    args,
	}
}

// SystemEventHandler はシステムイベントを処理するハンドラのインターフェース
type SystemEventHandler interface {
	HandleSaveEvent(event *SaveEvent) error
	HandleQuitEvent(event *QuitEvent) error
	HandleStatusEvent(event *StatusEvent) error
}

// DefaultSystemEventHandler はSystemEventHandlerの基本実装
type DefaultSystemEventHandler struct {
	editor interface {
		SetStatusMessage(format string, args ...interface{})
		Quit()
		IsDirty() bool
		IsQuitWarningShown() bool
		SetQuitWarningShown(bool)
	}
	fileManager interface {
		HandleSaveRequest(*SaveEvent) error
	}
}

// NewDefaultSystemEventHandler は新しいDefaultSystemEventHandlerを作成する
func NewDefaultSystemEventHandler(editor interface {
	SetStatusMessage(format string, args ...interface{})
	Quit()
	IsDirty() bool
	IsQuitWarningShown() bool
	SetQuitWarningShown(bool)
}, fileManager interface {
	HandleSaveRequest(*SaveEvent) error
}) *DefaultSystemEventHandler {
	return &DefaultSystemEventHandler{
		editor:      editor,
		fileManager: fileManager,
	}
}

// HandleSaveEvent はファイル保存イベントを処理する
func (h *DefaultSystemEventHandler) HandleSaveEvent(event *SaveEvent) error {
	// FileManagerに処理を委譲（ファイル名はFileManagerが管理）
	if err := h.fileManager.HandleSaveRequest(event); err != nil {
		return err
	}
	return nil
}

// HandleQuitEvent は終了イベントを処理する
func (h *DefaultSystemEventHandler) HandleQuitEvent(event *QuitEvent) error {
	if !event.Force && h.editor.IsDirty() && !h.editor.IsQuitWarningShown() {
		h.editor.SetQuitWarningShown(true)
		// 警告メッセージを直接設定（イベント発行なし）
		h.editor.SetStatusMessage("Warning! File has unsaved changes. Press Ctrl-Q or Ctrl-C again to quit.")
		return nil
	}
	h.editor.Quit()
	return nil
}

// HandleStatusEvent はステータスメッセージイベントを処理する
func (h *DefaultSystemEventHandler) HandleStatusEvent(event *StatusEvent) error {
	// ステータスメッセージを直接設定（イベント発行なし）
	h.editor.SetStatusMessage(event.Message, event.Args...)
	return nil
}
