package event

import "github.com/wasya-io/go-kilo/app/entity/cursor"

// EventType はイベントの種類を表す型です。
type EventType string

// 定義済みイベントタイプ
const (
	TypeSave     EventType = "save"     // 保存イベント
	TypeQuit     EventType = "quit"     // 終了イベント
	TypeInput    EventType = "input"    // 入力イベント
	TypeRefresh  EventType = "refresh"  // 画面更新イベント
	TypeCursor   EventType = "cursor"   // カーソルイベント
	TypeBuffer   EventType = "buffer"   // バッファイベント
	TypeCommand  EventType = "command"  // コマンド実行イベント
	TypeResponse EventType = "response" // 応答イベント
)

// Event はアプリケーション内で発生するイベントを表します。
type Event struct {
	Type    EventType   // イベントの種類
	Payload interface{} // イベントデータ
}

// SaveEvent は保存イベントのペイロードを表します。
type SaveEvent struct {
	Filename string // 保存するファイル名
	Force    bool   // 強制保存するかどうか
}

// QuitEvent は終了イベントのペイロードを表します。
type QuitEvent struct {
	Force bool // 強制終了するかどうか
}

// CursorEvent はカーソルイベントのペイロードを表します。
type CursorEvent struct {
	Action cursor.Movement // カーソル移動アクション
}

// BufferAction はバッファ操作の種類を表します。
type BufferAction int

const (
	BufferInsert BufferAction = iota
	BufferDelete
	BufferNewline
)

// BufferEvent はバッファイベントのペイロードを表します。
type BufferEvent struct {
	Action BufferAction
	Rune   rune
}

// ResponseEvent はコマンド応答イベントのペイロードを表します。
type ResponseEvent struct {
	Success bool   // 成功したかどうか
	Message string // メッセージ
	Error   error  // エラー情報
}

// NewEvent は新しいイベントを作成します。
func NewEvent(eventType EventType, payload interface{}) Event {
	return Event{
		Type:    eventType,
		Payload: payload,
	}
}

// NewSaveEvent は新しい保存イベントを作成します。
func NewSaveEvent(filename string, force bool) Event {
	return NewEvent(TypeSave, SaveEvent{
		Filename: filename,
		Force:    force,
	})
}

// NewQuitEvent は新しい終了イベントを作成します。
func NewQuitEvent(force bool) Event {
	return NewEvent(TypeQuit, QuitEvent{
		Force: force,
	})
}

// NewCursorEvent は新しいカーソルイベントを作成します。
func NewCursorEvent(action cursor.Movement) Event {
	return NewEvent(TypeCursor, CursorEvent{
		Action: action,
	})
}

// NewBufferEvent は新しいバッファイベントを作成します。
func NewBufferEvent(action BufferAction, r rune) Event {
	return NewEvent(TypeBuffer, BufferEvent{
		Action: action,
		Rune:   r,
	})
}

// NewResponseEvent は新しい応答イベントを作成します。
func NewResponseEvent(success bool, message string, err error) Event {
	return NewEvent(TypeResponse, ResponseEvent{
		Success: success,
		Message: message,
		Error:   err,
	})
}
