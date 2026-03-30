package controller

import (
	"sync"
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

func TestController_QuitWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFileManager := mock_filemanager.NewMockFileManager(ctrl)
	mockInputProvider := mock_input.NewMockProvider(ctrl)
	mockLogger := mock_core.NewMockLogger(ctrl)
	mockWriter := mock_writer.NewMockScreenWriter(ctrl)
	mockBuilder := mock_contents.NewMockBuilder(ctrl)

	// Use ASYNC bus to simulate real behavior
	mockEventBus := event.NewBus()
	// mockEventBus.SetSynchronous(false) // Default is false

	c := contents.NewContents(mockLogger)
	// Allow logs during setup
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()

	c.LoadContent([]string{"dirty content"})
	// Make content dirty
	c.InsertChar(contents.Position{X: 0, Y: 0}, 'X')

	cur := cursor.NewCursor()
	msg := contents.NewMessage("")
	scr := screen.NewScreen(mockBuilder, mockWriter, msg, cur, 24, 80)

	controller := NewController(scr, c, mockFileManager, mockInputProvider, mockLogger, mockEventBus)
	controller.SetRefreshDelay(0)

	// Capture builder writes to verify warning message
	var capturedOutput string
	var outputMutex sync.Mutex
	mockBuilder.EXPECT().Write(gomock.Any()).DoAndReturn(func(s string) {
		outputMutex.Lock()
		defer outputMutex.Unlock()
		capturedOutput += s
	}).AnyTimes()

	// RefreshScreen mocks
	mockFileManager.EXPECT().GetFilename().Return("test.txt").AnyTimes()
	mockBuilder.EXPECT().Clear().AnyTimes()
	mockBuilder.EXPECT().Build().Return("").AnyTimes()
	mockWriter.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()

	// 1. Send Ctrl+X
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlX}, nil, nil)

	err := controller.Process()
	assert.NoError(t, err)

	// Wait for async handler
	time.Sleep(100 * time.Millisecond)

	// Verify warning message is present in output
	outputMutex.Lock()
	if !assert.Contains(t, capturedOutput, "Warning! File has unsaved changes") {
		t.Logf("Captured output: %s", capturedOutput)
	}
	outputMutex.Unlock()

	// Reset output for next step
	outputMutex.Lock()
	capturedOutput = ""
	outputMutex.Unlock()

	// Check Quit channel
	select {
	case <-controller.Quit:
		t.Fatal("Quit channel should NOT be closed after first Ctrl+X")
	default:
		// OK
	}

	// 2. Send Ctrl+X again
	mockInputProvider.EXPECT().GetInputEvents().Return(key.KeyEvent{Type: key.KeyEventControl, Key: key.KeyCtrlX}, nil, nil)

	err = controller.Process()
	assert.NoError(t, err)

	// Wait for async handler
	time.Sleep(100 * time.Millisecond)

	// Verify Quit channel is closed
	select {
	case <-controller.Quit:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("Quit channel SHOULD be closed after second Ctrl+X")
	}
}
