# wsconsole

`systemd-run` を使ってログインセッションを起動する WebSocket ターミナルゲートウェイ

## 概要

`wsconsole` は Go で実装された HTTP/WebSocket サーバーで、Web ベースのターミナルアクセスを提供します。WebSocket 接続ごとに `systemd-run --uid=0 --pty` 経由で `/bin/login` セッションを新規起動し、完全に機能する PTY（擬似端末）を作成して WebSocket 接続と双方向にブリッジします。

### 特長

- **PTY 管理**: 完全な PTY サポートでログインセッションを起動
- **端末サイズ変更**: TIOCSWINSZ と SIGWINCH を処理し、動的な端末サイズ変更に対応
- **バイナリ透過性**: CR/LF 変換なしで全バイト（0x00-0xFF）を処理
- **WebSocket プロトコル**: バイナリ透過モード（デフォルト）と JSON メッセージモード
- **ヘルスモニタリング**: ヘルスチェック用の `/healthz` エンドポイント搭載
- **Ping/Pong**: 設定可能なタイムアウトによる自動接続維持
- **グレースフルシャットダウン**: PTY とプロセスリソースの適切なクリーンアップ
- **静的フロントエンド**: シンプルな Web ベースターミナルクライアントを同梱

## アーキテクチャ

```
┌─────────────┐         WebSocket          ┌──────────────┐
│   Browser   │◄──── Binary/JSON ─────────►│  wsconsole   │
│  (Client)   │   (transparent + resize)   │   Server     │
└─────────────┘                             └──────┬───────┘
                                                   │
                                                   │ PTY I/O
                                                   ▼
                                            ┌──────────────┐
                                            │ systemd-run  │
                                            │  /bin/login  │
                                            └──────────────┘
```

### ディレクトリ構成

```
wsconsole/
├── cmd/
│   └── wsconsole/
│       └── main.go              # エントリポイント、HTTP サーバー
├── internal/
│   ├── pty/
│   │   └── pty.go              # PTY 制御（リサイズ、I/O）
│   ├── systemd/
│   │   └── runner.go           # systemd-run ラッパー
│   └── ws/
│       └── handler.go          # WebSocket ハンドラー、メッセージプロトコル
├── deploy/
│   ├── systemd/
│   │   └── wsconsole.service  # systemd ユニットファイル
│   ├── polkit/
│   │   └── 10-wsconsole.rules # polkit ルール
│   └── static/
│       └── index.html          # シンプルな Web ターミナルフロントエンド
├── packaging/
│   └── deb/
│       ├── DEBIAN/
│       │   ├── control         # パッケージメタデータ
│       │   ├── postinst        # インストール後スクリプト
│       │   └── prerm           # 削除前スクリプト
│       └── README.md
├── TESTING.md                   # テスト手順
├── go.mod
├── go.sum
└── README.md
```

## 必要要件

- **Go**: 1.21 以降
- **systemd**: `systemd-run` コマンド用
- **polkit**: 権限管理用
- **Linux**: PTY 操作は Linux 専用

## Windows でのビルド/テスト

wsconsole は Linux 専用です。Windows では **Linux 向けにクロスビルド**し、**テストは Docker で実行**してください。

### クロスビルド（PowerShell）

```powershell
./build.ps1 -Arch amd64
./build.ps1 -Arch arm64
./build.ps1 -Arch armv7
```

NanoPi R1（Friendly Core）のような armv7 環境は `armv7` を使用してください。

### Docker でテスト

```bash
make test-docker
make test-launchers
make deb-test
```

## 開発環境

### VS Code Dev Container（推奨）

このプロジェクトは VS Code の Dev Container に対応しています。Dev Container 内では Go 開発環境と Docker in Docker (DinD) が利用可能です。

### クイックスタート（推奨）

Dev Container は root として実行されるため、direct launcher で直接起動できます：

```bash
make dev
```

これで以下が利用可能になります：
- WebSocket: ws://localhost:8080/ws
- Web UI: http://localhost:8080/
- Health: http://localhost:8080/healthz

**ログイン情報:**

Dev Container 内でテストユーザーを作成してください：

```bash
# テストユーザー作成（推奨）
make setup-users
```

利用可能なユーザー（パスワード設定後）:
- `testuser` / `testpass` (推奨)
- `vscode` / `vscode`
- `root` / `root`

**⚠️ 注意事項:**

Dev Container では開発用のカスタムプロンプト（`root ➜ ~`）が表示されます。標準的な Unix プロンプト（root は `#`、一般ユーザーは `$`）を確認したい場合は、テストコンテナを使用してください：

```bash
# テストコンテナで動作確認（標準プロンプト）
make test-launchers
# Direct launcher: http://localhost:8081/
# SystemdRun launcher: http://localhost:8082/
```

### systemd-run のテスト

systemd-run の動作確認は、専用コンテナで実施します：

```bash
make systemd-start
```

**便利なコマンド:**
```bash
# Dev Container 内で直接起動（direct launcher）
make dev

# systemd コンテナで起動（systemd-run launcher）
make systemd-start

# サービスログを表示
make systemd-logs

# コンテナ内でコマンド実行
docker exec -it wsconsole-systemd bash

# systemd コンテナを停止・削除
make systemd-down
```

## インストール

### ソースからビルド

1. リポジトリをクローン:
   ```bash
   git clone https://github.com/danmaid/wsconsole.git
   cd wsconsole
   ```

2. 依存関係をダウンロード:
   ```bash
   go mod download
   ```

3. バイナリをビルド:
   ```bash
   go build -o wsconsole ./cmd/wsconsole
   ```

4. サーバーを起動:
   ```bash
   ./wsconsole
   ```

### systemd を使用（本番環境）

1. ファイルをシステムの場所にコピー:
   ```bash
   sudo cp wsconsole /usr/local/bin/
   sudo mkdir -p /usr/local/share/wsconsole/static
   sudo cp deploy/static/index.html /usr/local/share/wsconsole/static/
   sudo cp deploy/systemd/wsconsole.service /etc/systemd/system/
   sudo cp deploy/polkit/10-wsconsole.rules /etc/polkit-1/rules.d/
   ```

2. サービスユーザーを作成:
   ```bash
   sudo groupadd -f wsconsole
   sudo useradd -r -g wsconsole -s /usr/sbin/nologin -M wsconsole
   ```

3. サービスを有効化して起動:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable wsconsole
   sudo systemctl start wsconsole
   ```

4. ステータスを確認:
   ```bash
   sudo systemctl status wsconsole
   ```

### Debian パッケージ

Debian/Ubuntu システム（Friendly Core など）用の .deb パッケージをビルドできます。

#### パッケージのビルドとテスト

```bash
# debパッケージをビルドしてテスト環境で実行
make deb-test

# これにより以下が実行されます:
# 1. バイナリのビルド
# 2. debパッケージの作成 (wsconsole_$(VERSION)_$(DEB_ARCH).deb)
# 3. Debian Bookwormベースのテストコンテナ起動
# 4. パッケージのインストールと起動
```

詳細は [TESTING.md](TESTING.md) を参照してください。

#### 本番環境へのデプロイ

```bash
# 1. debパッケージをビルド
make deb

# 2. 対象システムに転送
scp wsconsole_1.0.0_amd64.deb user@targethost:/tmp/

# 3. 対象システムでインストール
ssh user@targethost
sudo dpkg -i /tmp/wsconsole_1.0.0_amd64.deb

# 4. サービス確認
sudo systemctl status wsconsole.service
```

パッケージには以下が含まれます:
- バイナリ: `/usr/local/bin/wsconsole`
- 静的ファイル: `/usr/local/share/wsconsole/static/`
- systemd ユニット: `/etc/systemd/system/wsconsole.service`
- polkit ルール: `/etc/polkit-1/rules.d/10-wsconsole.rules`

## CI・リリース

GitHub Actions でテスト・ビルド・リリースを自動化しています。

- CI: push/PR でテスト・lint・ビルド
- リリース: `vX.Y.Z` のタグを作成すると、deb と zip を生成して GitHub Release に添付

ワークフロー:
- [.github/workflows/ci.yml](.github/workflows/ci.yml)
- [.github/workflows/release.yml](.github/workflows/release.yml)

## 使用方法

### コマンドラインオプション

```bash
./wsconsole [オプション]

オプション:
  -addr string
        HTTP サービスアドレス（デフォルト: ":8080"）
  -static string
        静的ファイルディレクトリ（デフォルト: "./deploy/static"）
  -log string
        ログレベル: debug, info, warn, error（デフォルト: "info"）
```

### エンドポイント

- `GET /` - 静的な Web ターミナルインターフェース
- `GET /ws` - ターミナル接続用 WebSocket エンドポイント（バイナリ透過モード）
- `GET /ws?mode=json` - JSON メッセージモードの WebSocket エンドポイント
- `GET /healthz` - ヘルスチェックエンドポイント（200 OK を返す）

### WebSocket プロトコル

#### バイナリ透過モード（デフォルト）

**クライアント → サーバー**
- バイナリフレーム: raw UTF-8 バイトを直接 PTY に送信
- テキストフレーム（JSON）: リサイズメッセージ用
  ```json
  {
    "type": "resize",
    "cols": 80,
    "rows": 24
  }
  ```

**サーバー → クライアント**
- バイナリフレーム: PTY からの raw 出力（0x00-0xFF 完全透過）

#### JSON メッセージモード（`?mode=json`）

**クライアント → サーバー**

データ送信（未実装、バイナリモード使用を推奨）:
```json
{
  "type": "data",
  "payload": "<base64-エンコードされたバイナリデータ>"
}
```

リサイズメッセージ（端末サイズ変更）:
```json
{
  "type": "resize",
  "cols": 80,
  "rows": 24
}
```

**サーバー → クライアント**

データ受信:
```json
{
  "type": "data",
  "payload": "<base64-エンコードされたバイナリデータ>"
}
```

### テスト

#### ユニットテストの実行

```bash
go test ./...
```

**注意**: テストには Linux 環境が必要です（`//go:build linux` タグ）。

#### 起動戦略の統合テスト

両方の起動戦略（direct と systemd-run）を検証するための Docker テスト環境を用意しています：

```bash
# 両方のテストコンテナをビルドして起動
make test-launchers

# 個別にテスト
# テスト1: Direct 起動 (UID=0) - ポート 8081
# テスト2: SystemdRun 起動 - ポート 8082

# ログを確認
docker compose -f docker-compose.test.yml logs -f test-direct
docker compose -f docker-compose.test.yml logs -f test-systemd-run

# 停止
docker compose -f docker-compose.test.yml down
```

**テスト用クレデンシャル:**
- ユーザー名: `testuser`
- パスワード: `testpass`

**テストクライアント:**

```bash
# Direct 起動のテスト (ポート 8081)
make test-client LAUNCHER=direct

# SystemdRun 起動のテスト (ポート 8082)
make test-client LAUNCHER=systemd-run
```

**注意**: `websocat` または `wscat` が必要です。

#### 手動テスト

1. サーバーを起動:
   ```bash
   go run ./cmd/wsconsole
   ```

2. ヘルスチェック:
   ```bash
   curl http://localhost:8080/healthz
   ```

3. Web インターフェースを開く:
   ```
   http://localhost:8080/
   ```

4. ブラウザから WebSocket 経由で接続、または WebSocket クライアントを使用

## セキュリティ上の考慮事項

- **root アクセス**: このサービスは `systemd-run --uid=0` を起動し、root ログインアクセスを許可します
- **polkit 設定**: polkit ルールが適切に設定され、アクセスが制限されていることを確認してください
- **ネットワーク公開**: 本番環境では TLS 付きリバースプロキシ（nginx、caddy）を使用してください
- **認証**: WebSocket アップグレード前に認証ミドルウェアの追加を検討してください
- **レート制限**: 悪用を防ぐためにレート制限を実装してください

## 開発

### テスト実行

```bash
go test ./...
```

**注意**: テストには Linux 環境が必要です（`//go:build linux` タグ）。

### 本番環境用ビルド

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o wsconsole ./cmd/wsconsole
```

### ロギング

アプリケーションは Go 1.21 の構造化ログ（`log/slog`）を使用します。`-log` フラグでログレベルを設定:

```bash
./wsconsole -log debug
```

ログは JSON 形式で出力され、ログアグリゲーターによる解析が容易です。

## トラブルシューティング

### PTY の問題

- **エラー: "failed to open PTY"**: `/dev/ptmx` がアクセス可能か確認してください
- **端末が文字化け**: バイナリ透過性が維持されているか確認してください（CR/LF 変換なし）

### systemd-run の問題

- **エラー: "failed to start systemd-run"**: systemd が実行中で、ユーザーに必要な権限があることを確認してください
- **Permission denied**: `/etc/polkit-1/rules.d/` の polkit ルールを確認してください

### WebSocket 接続の問題

- **Connection refused**: サーバーが実行中で、ファイアウォールがポート 8080 を許可しているか確認してください
- **Connection closed immediately**: PTY 起動中のエラーについてサーバーログを確認してください

## 技術仕様

### WebSocket 動作

- **Ping 間隔**: 27秒（pongWait の 9/10）
- **Pong タイムアウト**: 30秒
- **アイドルタイムアウト**: 5分
- **最大メッセージサイズ**: 512KB
- **PTY バッファサイズ**: 64KB チャンク

### エラーハンドリング

- PTY EOF → WebSocket を `CloseNormalClosure` で正常終了
- WebSocket Close/Error → PTY プロセスを `Kill()` してクリーンアップ
- 適切な defer 順序でリソースリーク防止

## ライセンス

MIT License - 詳細は LICENSE ファイルを参照してください。

## コントリビューション

コントリビューションを歓迎します！Issue を開くか、プルリクエストを送信してください。

## 作者

- Your Name (@yourname)

## 謝辞

- WebSocket サポートに [gorilla/websocket](https://github.com/gorilla/websocket) を使用
- PTY 管理に [creack/pty](https://github.com/creack/pty) を使用
- PTY 制御に [golang.org/x/sys/unix](https://golang.org/x/sys) を使用
