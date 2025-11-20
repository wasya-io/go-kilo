package controller_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/wasya-io/go-kilo/app/boundary/filemanager"
	"github.com/wasya-io/go-kilo/app/boundary/provider/input"
	"github.com/wasya-io/go-kilo/app/boundary/writer"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	"github.com/wasya-io/go-kilo/app/entity/core"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/screen"
	"github.com/wasya-io/go-kilo/app/usecase/controller"
)

// テスト用のセットアップ
func setupController(t *testing.T) (*controller.Controller, *filemanager.MockFileManager, *writer.MockScreenWriter, *event.Bus) {
	ctrl := gomock.NewController(t)

	// モックの作成
	mockLogger := core.NewMockLogger(ctrl)
	mockFileManager := filemanager.NewMockFileManager(ctrl)
	mockInputProvider := input.NewMockProvider(ctrl)
	mockWriter := writer.NewMockScreenWriter(ctrl)

	// ロガーのスタブ設定
	mockLogEntry := core.NewMockLogEntry(ctrl)
	mockLogEntry.EXPECT().WithType().Return(mockLogEntry).AnyTimes()
	mockLogEntry.EXPECT().WithString().Return(mockLogEntry).AnyTimes()
	mockLogEntry.EXPECT().WithInt().Return(mockLogEntry).AnyTimes()
	mockLogEntry.EXPECT().Do(gomock.Any()).AnyTimes()

	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Flush().AnyTimes()
	mockLogger.EXPECT().ReadyWithType(gomock.Any()).Return(mockLogEntry).AnyTimes()

	// Writerのスタブ設定は各テストで行う

	// 実際のコンテンツオブジェクトを作成
	c := contents.NewContents(mockLogger)
	c.LoadContent([]string{"Test line 1", "Test line 2"})

	// スクリーン関連オブジェクトを作成
	screenRows, screenCols := 25, 80
	cursorObj := cursor.NewCursor()

	screenObj := screen.NewScreen(
		contents.NewBuilder(),
		mockWriter,
		contents.NewMessage("", nil),
		cursorObj,
		screenRows,
		screenCols,
	)

	// イベントバスを作成
	eventBus := event.NewBus()

	// コントローラーを作成
	controller := controller.NewController(
		screenObj,
		c,
		mockFileManager,
		mockInputProvider,
		mockLogger,
		eventBus,
	)

	return controller, mockFileManager, mockWriter, eventBus
}

// 保存イベントのテスト
func TestSaveEventHandling(t *testing.T) {
	ctrl, mockFM, mockWriter, eventBus := setupController(t)
	defer eventBus.Shutdown()

	// 期待値の設定
	mockFM.EXPECT().HandleSaveRequest().Return(nil)
	mockFM.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	// 画面更新が行われる
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// 保存イベントを発行
	ctrl.PublishSaveEvent("test.txt", false)

	// ハンドラーが非同期で処理するのを待つ
	time.Sleep(50 * time.Millisecond)
}

// 保存イベントのエラーハンドリングテスト
func TestSaveEventHandlingError(t *testing.T) {
	ctrl, mockFM, mockWriter, eventBus := setupController(t)
	defer eventBus.Shutdown()

	// 期待値の設定
	// 保存リクエストがエラーを返す
	mockFM.EXPECT().HandleSaveRequest().Return(fmt.Errorf("save failed"))
	mockFM.EXPECT().GetFilename().Return("test.txt").AnyTimes()

	// エラーメッセージが表示されるため、画面更新が行われることを確認
	// 最低1回はWriteが呼ばれるはず
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).MinTimes(1)

	// 保存イベントを発行
	ctrl.PublishSaveEvent("test.txt", false)

	// ハンドラーが非同期で処理するのを待つ
	time.Sleep(50 * time.Millisecond)
}

// 終了イベントのテスト (クリーンな場合はすぐに終了)
func TestQuitEventHandlingClean(t *testing.T) {
	ctrl, _, _, eventBus := setupController(t)
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
	ctrl, mockFM, mockWriter, eventBus := setupController(t)
	defer eventBus.Shutdown()

	// GetFilenameが呼ばれる可能性があるためスタブ設定
	mockFM.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	// 警告表示などで画面更新が行われる
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

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

// 終了イベントのテスト (ダーティ時の警告表示確認)
func TestQuitEventHandlingDirtyWarning(t *testing.T) {
	ctrl, mockFM, mockWriter, eventBus := setupController(t)
	defer eventBus.Shutdown()

	// GetFilenameが呼ばれる可能性があるためスタブ設定
	mockFM.EXPECT().GetFilename().Return("test.txt").AnyTimes()

	// コンテンツを直接ダーティに設定
	ctrl.GetContents().SetDirty(true)

	// 警告メッセージ表示のためにWriteが呼ばれることを確認
	// 最低1回はWriteが呼ばれるはず
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).MinTimes(1)

	// 終了イベントを発行
	ctrl.PublishQuitEvent(false)

	// 非同期処理待ち
	time.Sleep(50 * time.Millisecond)

	// Quitチャネルが閉じられていないことを確認
	select {
	case <-ctrl.Quit:
		t.Error("Quit channel should not be closed")
	default:
		// OK
	}
}

// 強制終了イベントのテスト (ダーティでも終了)
func TestForceQuitEventHandling(t *testing.T) {
	ctrl, _, _, eventBus := setupController(t)
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
