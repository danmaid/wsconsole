# テスト手順

## クイックスタート

```bash
# 両方のテスト環境起動
make test-launchers

# または個別に
make test-direct              # Direct launcher のみ
make test-systemd-run         # SystemdRun launcher のみ
make test-down                # 停止
```

## テスト環境

| 起動方式 | ポート | URL |
|---------|--------|-----|
| Direct | 6081 | https://localhost:6081/ |
| SystemdRun | 6082 | https://localhost:6082/ |

## 接続方法

### ブラウザから

- Direct: `https://localhost:6081/`
- SystemdRun: `https://localhost:6082/`

自己署名証明書の警告が表示される場合は、「詳細」→「アクセス」をクリック。

### ヘルスチェック

```bash
curl -k https://localhost:6081/healthz   # Direct
curl -k https://localhost:6082/healthz   # SystemdRun
```

### ログ確認

```bash
make test-logs                 # 両方
make test-direct-logs          # Direct のみ
make test-systemd-run-logs     # SystemdRun のみ
```

## ユーザー情報

| ユーザー名 | パスワード |
|-----------|-----------|
| testuser | testpass |
| root | root |

## Debian パッケージテスト

```bash
make deb-test                  # Debian パッケージテスト起動
docker exec -it wsconsole-deb-test bash   # コンテナ内に入る
make deb-test-down             # 停止
```

アクセス: `https://localhost:6083/`

## ユニットテスト

```bash
make test                      # 全テスト実行
```

**注意**: Linux 環境が必要です。

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
