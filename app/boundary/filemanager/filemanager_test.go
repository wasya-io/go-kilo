package filemanager

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestFileManager_OpenFile_FileNotExists(t *testing.T) {
	// モックコントローラーの作成
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// モックFileManagerの作成
	mock := NewMockFileManager(ctrl)

	// 存在しないファイルを開こうとしたときの振る舞いを設定
	nonExistentFile := "non_existent_file.txt"
	mock.EXPECT().
		OpenFile(nonExistentFile).
		Return(os.ErrNotExist)

	// テストの実行
	err := mock.OpenFile(nonExistentFile)
	if err != os.ErrNotExist {
		t.Errorf("Expected error os.ErrNotExist, got %v", err)
	}
}