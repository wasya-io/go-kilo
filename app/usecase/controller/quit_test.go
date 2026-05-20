package controller

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_filemanager "github.com/wasya-io/go-kilo/app/boundary/filemanager/mock"
	mock_input "github.com/wasya-io/go-kilo/app/boundary/provider/input/mock"
	mock_writer "github.com/wasya-io/go-kilo/app/boundary/writer/mock"
	"github.com/wasya-io/go-kilo/app/entity/contents"
	mock_contents "github.com/wasya-io/go-kilo/app/entity/contents/mock"
	mock_core "github.com/wasya-io/go-kilo/app/entity/core/mock"
	"github.com/wasya-io/go-kilo/app/entity/cursor"
	"github.com/wasya-io/go-kilo/app/entity/event"
	"github.com/wasya-io/go-kilo/app/entity/key"
	"github.com/wasya-io/go-kilo/app/entity/screen"
)

func TestController_QuitSequence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)
	// Important: Use asynchronous event bus to simulate real behavior
	mockEventBus := event.NewBus()
	// mockEventBus.SetSynchronous(true) // Start with default (async in implementation?)
	// Actually NewBus() returns async by default.
	// But let's be explicit if we can, or just remove SetSynchronous(true)
	mockEventBus.SetSynchronous(false)

	// Mock expectations for screen refresh (happens on warning) and general logging
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Write(gomock.Any()).AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// Setup contents and make it dirty
	c := contents.NewContents(mockLogger)
	c.LoadContent([]string{"original"})
	c.InsertChar(contents.Position{X: 8, Y: 0}, '.')
	// Make sure it's considered dirty
	assert.True(t, c.IsDirty())

	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, nil, mockEventBus)
	controller.SetRefreshDelay(0)

	// 1st Ctrl+X: Should trigger Warning and NOT Quit
	// We expect GetInputEvents to be called once for first Ctrl+X
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlX}, nil, nil)

	// Execute first process
	err := controller.Process()
	assert.NoError(t, err)

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	// INJECT PHANTOM EVENT (Rune = 0)
	// This simulates the suspected bug cause
	// Keeping this to ensure robustness, but focusing on the race condition now.
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventChar, Rune: 0}, nil, nil)
	err = controller.Process()
	assert.NoError(t, err)

	// REMOVED SLEEP to expose race condition.
	// The Quit channel MUST be closed before Process returns to prevent the main loop
	// from entering another blocking read.

	// Verify Quit channel is NOT closed after phantom event
	select {
	case <-controller.Quit:
		t.Fatal("Quit channel closed after phantom event (should be ignored)")
	default:
		// OK
	}

	// 2nd Ctrl+X: Should trigger Quit
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlX}, nil, nil)

	// Execute second process
	err = controller.Process()
	assert.NoError(t, err)

	// Verify Quit channel IS closed IMMEDIATELY (Race Condition Check)
	// If PublishQuitEvent is async, this will fail.
	select {
	case <-controller.Quit:
		// OK
	default:
		t.Fatal("Quit channel NOT closed immediately after second Ctrl+X (Race Condition!)")
	}
}
