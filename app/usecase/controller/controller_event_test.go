package controller_test

import (
	"testing"
	"time"

	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/controller"
)

// モック用のロガー
type mockLogger struct{}

func (m *mockLogger) Log(category string, message string) {}
func (m *mockLogger) Flush()                              {}
func (m *mockLogger) ReadyWithType(messageType string) core.LogEntry {
	return &mockLogEntry{}
}

// モック用のLogEntry
type mockLogEntry struct{}

func (m *mockLogEntry) WithType() core.LogEntry   { return m }
func (m *mockLogEntry) WithString() core.LogEntry { return m }
func (m *mockLogEntry) WithInt() core.LogEntry    { return m }
func (m *mockLogEntry) Do(values ...interface{})  {}

// モック用のwriter
type mockWriter struct{}

func (m *mockWriter) SetRowOffset(int)                         {}
func (m *mockWriter) SetColOffset(int)                         {}
func (m *mockWriter) ResetBuffer()                             {}
func (m *mockWriter) RenderRow(int, []rune, int, string)       {}
func (m *mockWriter) RenderStatusBar(int, string, string, int) {}
func (m *mockWriter) RenderMessageBar(int, string)             {}
func (m *mockWriter) GetBuffer() []byte                        { return []byte{} }
func (m *mockWriter) ClearScreen() string                      { return "\x1b[2J" }
func (m *mockWriter) MoveCursorToHome() string                 { return "\x1b[H" }
func (m *mockWriter) Write(s string) error                     { return nil }

// モック用のファイルマネージャー
type mockFileManager struct {
	filename     string
	saveRequests int
	saveError    error
	isDirty      bool
	content      []string
}

func newMockFileManager() *mockFileManager {
	return &mockFileManager{
		filename: "test.txt",
		isDirty:  false,
		content:  []string{"Test line 1", "Test line 2"},
	}
}

func (m *mockFileManager) OpenFile(filename string) error {
	m.filename = filename
	return nil
}

func (m *mockFileManager) SaveFile(filename string, content []string) error {
	m.saveRequests++
	m.content = content
	return m.saveError
}

func (m *mockFileManager) SaveCurrentFile() error {
	m.saveRequests++
	return m.saveError
}

func (m *mockFileManager) HandleSaveRequest() error {
	m.saveRequests++
	return m.saveError
}

func (m *mockFileManager) GetFilename() string {
	return m.filename
}

// モック用のインプットプロバイダー
type mockInputProvider struct{}

func (m *mockInputProvider) GetInputEvents() (key.KeyEvent, []key.KeyEvent, error) {
	return key.KeyEvent{}, nil, nil
}

// モック用のRow
type mockRow struct {
	content []rune
}

func (r *mockRow) GetRuneCount() int {
	return len(r.content)
}

func (r *mockRow) OffsetToScreenPosition(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(r.content) {
		return len(r.content)
	}
	return offset
}

// モック用のコンテンツ - 実際のcontents.Contentsに近いインターフェース
type mockContents struct {
	dirty bool
	rows  []*mockRow
}

func newMockContents() *mockContents {
	rows := []*mockRow{
		{content: []rune("Test line 1")},
		{content: []rune("Test line 2")},
	}
	return &mockContents{
		dirty: false,
		rows:  rows,
	}
}

func (m *mockContents) IsDirty() bool {
	return m.dirty
}

func (m *mockContents) SetDirty(dirty bool) {
	m.dirty = dirty
}

func (m *mockContents) GetRow(y int) *mockRow {
	if y >= 0 && y < len(m.rows) {
		return m.rows[y]
	}
	return nil
}

func (m *mockContents) GetLineCount() int {
	return len(m.rows)
}

func (m *mockContents) GetContentLine(y int) []rune {
	if y >= 0 && y < len(m.rows) {
		return m.rows[y].content
	}
	return nil
}

func (m *mockContents) InsertChar(pos contents.Position, ch rune) {
	// テスト用の簡易実装
}

func (m *mockContents) DeleteChar(pos contents.Position) {
	// テスト用の簡易実装
}

func (m *mockContents) InsertNewline(pos contents.Position, indentSize int) {
	// テスト用の簡易実装
}

func (m *mockContents) LoadContent(lines []string) {
	m.rows = make([]*mockRow, len(lines))
	for i, line := range lines {
		m.rows[i] = &mockRow{content: []rune(line)}
	}
	m.dirty = false
}

func (m *mockContents) GetAllLines() []string {
	result := make([]string, len(m.rows))
	for i, row := range m.rows {
		result[i] = string(row.content)
	}
	return result
}

// モック用のカーソル - 実際のcursor.Cursorインターフェースに合わせる
type mockCursor struct {
	x, y int
}

func (c *mockCursor) ToPosition() contents.Position {
	return contents.Position{X: c.x, Y: c.y}
}

func (c *mockCursor) NewLine() {
	c.x = 0
	c.y++
}

func (c *mockCursor) SetPosition(x, y int) {
	c.x = x
	c.y = y
}

// モック用のスクリーン - 実際のscreen.Screenインターフェースに合わせる
type mockScreen struct {
	messages  []string
	cursor    *mockCursor
	rowOffset int
	colOffset int
	rows      int
	cols      int
}

func newMockScreen() *mockScreen {
	return &mockScreen{
		messages:  []string{},
		cursor:    &mockCursor{},
		rowOffset: 0,
		colOffset: 0,
		rows:      25,
		cols:      80,
	}
}

func (m *mockScreen) SetMessage(format string, args ...interface{}) {
	m.messages = append(m.messages, format)
}

func (m *mockScreen) GetOffset() (int, int) {
	return m.colOffset, m.rowOffset
}

func (m *mockScreen) SetRowOffset(offset int) {
	m.rowOffset = offset
}

func (m *mockScreen) SetColOffset(offset int) {
	m.colOffset = offset
}

func (m *mockScreen) GetCursor() *mockCursor {
	return m.cursor
}

func (m *mockScreen) GetRowLines() int {
	return m.rows
}

func (m *mockScreen) GetColLines() int {
	return m.cols
}

func (m *mockScreen) MoveCursor(movement cursor.Movement, contents *contents.Contents) {
	// テスト用の簡易実装
}

func (m *mockScreen) SetCursorPosition(x, y int) {
	m.cursor.SetPosition(x, y)
}

func (m *mockScreen) Redraw(contents *contents.Contents, filename string) error {
	return nil
}

func (m *mockScreen) Flush() error {
	return nil
}

func (m *mockScreen) ClearScreen() string {
	return "\x1b[2J"
}

func (m *mockScreen) MoveCursorToHome() string {
	return "\x1b[H"
}

// テスト用のセットアップ - 実際のオブジェクトを使用しつつ監視用のモックを併用
func setupController() (*controller.Controller, *mockFileManager, *mockContents, *mockScreen, *event.Bus) {
	// モックロガーを作成
	logger := &mockLogger{}
	mockFileManager := newMockFileManager()
	inputProvider := &mockInputProvider{}
	mockScreen := newMockScreen()
	mockContents := newMockContents()

	// イベントバスを作成
	eventBus := event.NewBus()

	// 実際のコンテンツオブジェクトを作成
	c := contents.NewContents(logger)

	// テスト用にコンテンツに初期データをセット
	c.LoadContent([]string{"Test line 1", "Test line 2"})

	// スクリーン関連オブジェクトを作成
	screenRows, screenCols := 25, 80
	cursorObj := cursor.NewCursor()
	writer := &mockWriter{}

	screenObj := screen.NewScreen(
		contents.NewBuilder(),
		writer,
		contents.NewMessage("", nil),
		cursorObj,
		screenRows,
		screenCols,
	)

	// コントローラーを作成 - モックを使用する
	ctrl := controller.NewController(
		screenObj,
		c,
		mockFileManager, // モックFileManagerを使用
		inputProvider,
		logger,
		eventBus,
	)

	return ctrl, mockFileManager, mockContents, mockScreen, eventBus
}

// 保存イベントのテスト
func TestSaveEventHandling(t *testing.T) {
	ctrl, fileManager, _, _, eventBus := setupController()
	defer eventBus.Shutdown()

	// 保存イベントを発行
	ctrl.PublishSaveEvent("test.txt", false)

	// ハンドラーが非同期で処理するのを待つ
	time.Sleep(50 * time.Millisecond)

	// ファイル保存が実行されたことを確認
	if fileManager.saveRequests != 1 {
		t.Errorf("Expected 1 save request, got %d", fileManager.saveRequests)
	}
}

// 終了イベントのテスト (クリーンな場合はすぐに終了)
func TestQuitEventHandlingClean(t *testing.T) {
	ctrl, _, _, _, eventBus := setupController()
	defer eventBus.Shutdown()

	// 終了イベントの処理をモニタリングするためのゴルーチン
	quitReceived := false
	go func() {
		select {
		case <-ctrl.Quit:
			quitReceived = true
		case <-time.After(100 * time.Millisecond):
			// タイムアウト
		}
	}()

	// 終了イベントを発行
	ctrl.PublishQuitEvent(false)

	// 終了シグナルを受け取ったことを確認
	time.Sleep(50 * time.Millisecond)
	if !quitReceived {
		t.Error("Quit channel was not closed")
	}
}

// 終了イベントのテスト (ダーティな場合は警告メッセージ)
func TestQuitEventHandlingDirty(t *testing.T) {
	ctrl, _, _, _, eventBus := setupController()
	defer eventBus.Shutdown()

	// コンテンツを直接ダーティに設定
	ctrl.GetContents().SetDirty(true)

	// テスト用に各テストで新しいゴルーチンを使う
	done := make(chan bool)
	var quitReceived bool

	go func() {
		select {
		case <-ctrl.Quit:
			quitReceived = true
			done <- true
		case <-time.After(100 * time.Millisecond):
			quitReceived = false
			done <- true
		}
	}()

	// 終了イベントを発行
	ctrl.PublishQuitEvent(false)

	// 待つ
	<-done

	// 終了シグナルは受け取らないはず
	if quitReceived {
		t.Error("Quit channel was closed despite dirty state")
	}

	// 2回目の終了イベントのために別のゴルーチン
	go func() {
		select {
		case <-ctrl.Quit:
			quitReceived = true
			done <- true
		case <-time.After(100 * time.Millisecond):
			quitReceived = false
			done <- true
		}
	}()

	// 2回目の終了イベントを発行
	ctrl.PublishQuitEvent(false)

	// 待つ
	<-done

	// 2回目なので終了すべき
	if !quitReceived {
		t.Error("Quit channel was not closed after second quit event")
	}
}

// 強制終了イベントのテスト (ダーティでも終了)
func TestForceQuitEventHandling(t *testing.T) {
	ctrl, _, _, _, eventBus := setupController()
	defer eventBus.Shutdown()

	// コンテンツを直接ダーティに設定
	ctrl.GetContents().SetDirty(true)

	// 終了イベントの処理をモニタリングするためのゴルーチン
	quitReceived := false
	done := make(chan bool)

	go func() {
		select {
		case <-ctrl.Quit:
			quitReceived = true
			done <- true
		case <-time.After(100 * time.Millisecond):
			quitReceived = false
			done <- true
		}
	}()

	// 強制終了イベントを発行
	ctrl.PublishQuitEvent(true)

	// 待つ
	<-done

	// 終了シグナルを受け取ったことを確認
	if !quitReceived {
		t.Error("Quit channel was not closed despite force flag")
	}
}
