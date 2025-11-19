# Go言語プロジェクト用 GitHub Copilotインストラクション

## 生成する回答の原則

- Think English, Talk Japanese.
- 実装後はコンパイルエラーをチェックし、コンパイルエラーの残った状態で動作確認、テストの実行を要求しないこと

## 基本原則

- **シンプルさ**: Go言語は明快さと単純さを重視します
- **明示的なエラー処理**: `error`型を返し、適切に処理する
- **並行処理**: ゴルーチンとチャネルを効果的に使用する
- **インターフェースの小型化**: 小さなインターフェースを設計する
- **標準ライブラリの活用**: 可能な限り標準ライブラリを使用する

## プロジェクト構造

```
project/
├── app/                # ソースのルート
│   ├── boundary/       # ファイルやキーボードなどの実体と結びつく処理、クリーンアーキテクチャにおけるインフラ層
│   ├── config/         # .envファイルからconfig構造体を生成する処理
│   ├── entity/         # boundaryへのインターフェイス、単体で完結するビジネスロジック
│   └── usecase/        # 複数のオブジェクトを組み合わせて実装するビジネスロジック
│       ├── command/    # コマンドの生成処理を格納
│       ├── controller/ # 入出力を仲介するコントローラ機能
│       ├── editor/     # エディタ本体の機能
│       └── parser/     # 入力をイベントに変換するパーサ処理
├── docs/               # ドキュメント
├── editor.go           # Editorの生成とDIを処理
├── main.go             # メインアプリケーションのエントリポイント
└── go.mod              # Go モジュールファイル
```

## 命名規則

- **パッケージ名**: 小文字の単一語（例: `http`, `json`, `auth`）
- **インターフェース**: 動詞+er（例: `Reader`, `Writer`, `Handler`）
- **変数名**: キャメルケース（例: `userID`, `timeStamp`）
- **定数**: キャメルケース（例: `maxLength`, `defaultTimeout`）
- **エクスポートされる名前**: 大文字で始める（例: `Client`, `NewReader`）
- **非公開の名前**: 小文字で始める（例: `conn`, `buffer`）

## エラー処理

```go
// 良い例
f, err := os.Open(filename)
if err != nil {
    return nil, fmt.Errorf("ファイルを開けません: %w", err)
}

// 悪い例 - エラーを無視している
f, _ := os.Open(filename)
```

- エラーをラップする際は `fmt.Errorf("...: %w", err)` を使用する
- エラーが発生したら早期リターンする
- カスタムエラー型を定義する場合は `errors.Is()` と `errors.As()` に対応する

## コメントとドキュメンテーション

```go
// User は認証されたユーザーを表します。
// このモデルはデータベースとAPIの両方で使用されます。
type User struct {
    ID        string    `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NewUser は新しいUserインスタンスを作成します。
// emailが空の場合はエラーを返します。
func NewUser(email string) (*User, error) {
    // 実装...
}
```

- エクスポートされる全ての型、関数、メソッド、パッケージにはコメントを書く
- コメントは「何を」ではなく「なぜ」を説明する
- パッケージコメントは `package` キーワードの直前に記述する

## テスト

```go
func TestCalculate(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
        wantErr  bool
    }{
        {"正常なケース", 5, 10, false},
        {"エラーケース", -1, 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Calculate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.expected {
                t.Errorf("Calculate() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

- テーブル駆動テストを使用する
- サブテストを使用して大きなテストを整理する
- ヘルパー関数には `t.Helper()` を使用する
- モックには `gomock` を検討する

## 依存関係管理

```
# go.mod
module github.com/username/project

go 1.21

require (
    github.com/pkg/errors v0.9.1
    golang.org/x/sync v0.5.0
)
```

- `go mod tidy` を定期的に実行する
- 依存関係は最小限に保つ
- ベンダリングが必要な場合は `go mod vendor` を使用する
- バージョンを明示的に指定する（例: `go get github.com/pkg/errors@v0.9.1`）

## 並行処理パターン

```go
// 良い例 - コンテキストの使用
func ProcessData(ctx context.Context, data []string) error {
    results := make(chan string, len(data))
    errs := make(chan error, 1)
    
    for _, item := range data {
        go func(item string) {
            select {
            case <-ctx.Done():
                return
            default:
                result, err := process(item)
                if err != nil {
                    select {
                    case errs <- err:
                    default:
                    }
                    return
                }
                results <- result
            }
        }(item)
    }
    
    // 結果の収集...
}
```

- コンテキストを使用してキャンセルとタイムアウトを伝播する
- ゴルーチンのリークを防ぐ（必ず終了する方法を確保する）
- チャネルのサイズを適切に設定する
- `sync.WaitGroup` を使用してゴルーチンの完了を待つ
- `sync/errgroup` パッケージを検討する

## コードスタイル

- `gofmt` または `goimports` で自動フォーマットする
- 行の長さは一般的に80-100文字以内に保つ
- 必要以上に複雑にしない
- DRY（Don't Repeat Yourself）原則に従う

## パフォーマンスの考慮

- 必要な場合のみポインタを使用する
- スライスとマップのサイズを事前に割り当てる
- ヒープアロケーションを最小限に抑える
- `sync.Pool` を使用して一時オブジェクトを再利用する

## 推奨ツール

- **gofmt/goimports**: コードフォーマット
- **golangci-lint**: 包括的なLintツール
- **go test**: テスト実行
- **go vet**: コードの問題検出
- **gocritic**: 高度なコード分析
- **pprof**: プロファイリング

## デバッグとロギング

```go
// 構造化ログの例
log.WithFields(log.Fields{
    "user_id": userID,
    "action":  "login",
    "status":  "success",
}).Info("ユーザーがログインしました")

// エラーログの例
if err != nil {
    log.WithError(err).Error("データベース接続に失敗しました")
}
```

- 構造化ログを使用する（`zap`, `logrus` など）
- ログレベルを適切に設定する
- 機密情報をログに記録しない
- 本番環境と開発環境で異なるログ設定を使用する

## ベストプラクティスのチェックリスト

- [ ] エラーが適切に処理されている
- [ ] コンテキストが伝播されている
- [ ] レースコンディションがない（`go race` で検証）
- [ ] テストカバレッジが十分ある
- [ ] コメントが最新かつ明確である
- [ ] リソースがすべて適切にクローズされている
- [ ] セキュリティの脆弱性がない
- [ ] コードが読みやすく、メンテナンスしやすい
