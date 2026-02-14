# 起動戦略のテスト

このドキュメントでは、両方の起動戦略をテストするための Docker 環境の使い方を説明します。

## 利用可能な戦略

### 1. Direct 起動（`-launcher direct`）
- **要件**: UID=0 (root) で実行
- **方式**: `/bin/login` を直接 fork
- **用途**: root で起動する本番環境
- **テストポート**: 8081

### 2. SystemdRun 起動（`-launcher systemd-run`）
- **要件**: systemd-run が利用可能
- **方式**: `systemd-run --uid=0 /bin/login` で権限昇格
- **用途**: 一般ユーザーからの実行、systemd 管理
- **テストポート**: 8082

## クイックスタート

### すべてのテストを起動

```bash
make test-launchers
```

### 個別にテスト

```bash
# Direct 起動のみ
make test-direct

# SystemdRun 起動のみ
make test-systemd-run

# 停止
make test-down
```

## ログの確認

```bash
# すべてのログ
make test-logs

# Direct 起動のログ
make test-direct-logs

# SystemdRun 起動のログ
make test-systemd-run-logs
```

## 接続テスト

### WebSocket クライアントを使用

```bash
# Direct 起動 (port 8081)
make test-client LAUNCHER=direct

# SystemdRun 起動 (port 8082)
make test-client LAUNCHER=systemd-run
```

**注意**: `websocat` または `wscat` が必要です。

```bash
# websocat のインストール
cargo install websocat

# wscat のインストール
npm install -g wscat
```

### ブラウザから接続

1. Direct 起動: http://localhost:8081/
2. SystemdRun 起動: http://localhost:8082/

### curl でヘルスチェック

```bash
curl http://localhost:8081/healthz  # Direct
curl http://localhost:8082/healthz  # SystemdRun
```

## テスト用クレデンシャル

両方のコンテナには同じテストユーザーが設定されています：

- **ユーザー名**: `testuser`
- **パスワード**: `testpass`

## プロンプト表示

テストコンテナでは、標準的な Unix プロンプトが正しく表示されます：

- **Root ユーザー**: `root@container:/path#` (末尾が `#`)
- **一般ユーザー**: `testuser@container:/path$` (末尾が `$`)

**注意**: Dev Container では開発用のカスタムプロンプト（`root ➜ ~`）が表示されますが、テストコンテナや本番環境では標準的な Unix プロンプトが表示されます。

## トラブルシューティング

### コンテナの状態確認

```bash
docker ps --filter "name=terminal-gateway"
```

### コンテナ内でシェルを起動

```bash
# Direct 起動
docker compose -f docker-compose.test.yml exec test-direct bash

# SystemdRun 起動
docker compose -f docker-compose.test.yml exec test-systemd-run bash
```

### systemd サービスの確認 (SystemdRun only)

```bash
docker compose -f docker-compose.test.yml exec test-systemd-run systemctl status wsconsole-test
```

### コンテナの再起動

```bash
docker compose -f docker-compose.test.yml restart test-direct
docker compose -f docker-compose.test.yml restart test-systemd-run
```

### クリーンアップして再ビルド

```bash
docker compose -f docker-compose.test.yml down
docker compose -f docker-compose.test.yml up -d --build
```

## 自動選択のテスト

デフォルトの `auto` 戦略では、環境に応じて自動的に最適な方法が選択されます：

1. UID=0 → Direct
2. UID!=0 && systemd-run 利用可能 → SystemdRun
3. それ以外 → エラー

これは各コンテナで自動的にテストされています。

---

# Debian パッケージのテスト

このセクションでは、Friendly Core 向けの deb パッケージをテスト環境でビルド・インストール・動作確認する方法を説明します。

## クイックスタート

```bash
make deb-test
```

このコマンドは以下の処理を自動で行います：
1. Goバイナリのビルド（Linux/amd64）
2. debパッケージの作成
3. Debian Bookwormベースのテストコンテナの起動
4. debパッケージのインストール
5. systemdサービスとしてwsconsoleを起動

## 主要なコマンド

### deb パッケージのビルドのみ

```bash
make deb
```

- `wsconsole_$(VERSION)_$(DEB_ARCH).deb` ファイルが生成されます
- `VERSION` と `DEB_ARCH` は Makefile で指定可能です

例:

```bash
make deb VERSION=1.2.3 DEB_ARCH=arm64
```

### テスト環境の起動

```bash
make deb-test
```

### テスト環境へのシェルアクセス

```bash
make deb-test-shell
```

### ログの確認

```bash
make deb-test-logs
```

### サービスの状態確認

```bash
make deb-test-status
```

### デプロイ検証（自動）

```bash
make deb-test-verify
```

### テスト環境の停止

```bash
make deb-test-down
```

## アクセスURL

- **Web UI**: http://localhost:8083/
- **WebSocket**: ws://localhost:8083/ws
- **Health Check**: http://localhost:8083/healthz

## テストユーザー

- **testuser** / testpass
- **admin** / admin
- **root** / root

## パッケージ内容

```
/usr/local/bin/wsconsole              # バイナリ
/usr/local/share/wsconsole/static/    # 静的ファイル（Web UI）
/etc/systemd/system/wsconsole.service # systemdユニットファイル
/etc/polkit-1/rules.d/10-wsconsole.rules # polkitルール
```

インストール時の動作：
- `wsconsole` ユーザーとグループの作成
- systemdサービスの有効化と起動
- systemd daemon の再読み込み

**重要**: サービスは `--launcher=systemd-run` で起動するため、各WebSocket接続は systemd-run を介して /bin/login セッションを起動します。

## 本番環境へのデプロイ

```bash
VERSION=1.0.0
make deb VERSION=$$VERSION DEB_ARCH=amd64
scp wsconsole_$${VERSION}_amd64.deb user@friendlycore:/tmp/
ssh user@friendlycore
sudo dpkg -i /tmp/wsconsole_$${VERSION}_amd64.deb
sudo systemctl status wsconsole.service
```
