package event_test

import (
	"errors"
	"testing"

	"github.com/wasya-io/go-kilo/app/entity/event"
)

func TestNewEvent(t *testing.T) {
	// イベントの作成をテスト
	payload := "test payload"
	evt := event.NewEvent(event.TypeInput, payload)

	if evt.Type != event.TypeInput {
		t.Errorf("Expected event type %s, got %s", event.TypeInput, evt.Type)
	}

	if p, ok := evt.Payload.(string); !ok || p != payload {
		t.Errorf("Expected payload %v, got %v", payload, evt.Payload)
	}
}

func TestNewSaveEvent(t *testing.T) {
	// 保存イベントの作成をテスト
	filename := "test.txt"
	force := true
	evt := event.NewSaveEvent(filename, force)

	if evt.Type != event.TypeSave {
		t.Errorf("Expected event type %s, got %s", event.TypeSave, evt.Type)
	}

	if p, ok := evt.Payload.(event.SaveEvent); !ok {
		t.Errorf("Expected SaveEvent payload, got %T", evt.Payload)
	} else {
		if p.Filename != filename {
			t.Errorf("Expected filename %s, got %s", filename, p.Filename)
		}
		if p.Force != force {
			t.Errorf("Expected force %v, got %v", force, p.Force)
		}
	}
}

func TestNewQuitEvent(t *testing.T) {
	// 終了イベントの作成をテスト
	force := true
	evt := event.NewQuitEvent(force)

	if evt.Type != event.TypeQuit {
		t.Errorf("Expected event type %s, got %s", event.TypeQuit, evt.Type)
	}

	if p, ok := evt.Payload.(event.QuitEvent); !ok {
		t.Errorf("Expected QuitEvent payload, got %T", evt.Payload)
	} else {
		if p.Force != force {
			t.Errorf("Expected force %v, got %v", force, p.Force)
		}
	}
}

func TestNewResponseEvent(t *testing.T) {
	// 応答イベントの作成をテスト
	success := true
	message := "success message"
	err := errors.New("test error")
	evt := event.NewResponseEvent(success, message, err)

	if evt.Type != event.TypeResponse {
		t.Errorf("Expected event type %s, got %s", event.TypeResponse, evt.Type)
	}

	if p, ok := evt.Payload.(event.ResponseEvent); !ok {
		t.Errorf("Expected ResponseEvent payload, got %T", evt.Payload)
	} else {
		if p.Success != success {
			t.Errorf("Expected success %v, got %v", success, p.Success)
		}
		if p.Message != message {
			t.Errorf("Expected message %s, got %s", message, p.Message)
		}
		if p.Error != err {
			t.Errorf("Expected error %v, got %v", err, p.Error)
		}
	}
}
