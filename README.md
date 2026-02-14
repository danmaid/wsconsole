# wsconsole

Web ベースのターミナルゲートウェイ。WebSocket 経由で `/bin/login` セッションへのアクセスを提供します。

## 特長

- **PTY 管理** - 完全な疑似端末サポート
- **動的リサイズ** - 端末サイズ変動に対応
- **バイナリ透過** - 全バイト（0x00-0xFF）を正確に転送
- **複数プロトコル** - バイナリモード・JSONモード両対応
- **ヘルスチェック** - `/healthz` エンドポイント搭載
- **HTTPS デフォルト** - セキュアがデフォルト（自動証明書生成）
- **パスプレフィックス対応** - リバースプロキシ互換

## クイックスタート

### ビルド・実行

```bash
# ビルド
make build

# 実行
./wsconsole
# → https://localhost:6001/
```

### Dev Container（推奨）

```bash
make dev
# → https://localhost:6001/
```

### テスト

```bash
make test-launchers
# Direct launcher: https://localhost:6081/
# SystemdRun launcher: https://localhost:6082/
```

## ドキュメント

| ドキュメント | 説明 |
|-----------|------|
| [SETUP.md](SETUP.md) | セットアップ・ポート・HTTPS設定 |
| [TESTING.md](TESTING.md) | テスト手順 |
| [USERS.md](USERS.md) | テストユーザー情報 |

## 必要要件

- **Go**: 1.21 以降
- **systemd**: `systemd-run` コマンド
- **Linux**: PTY 操作は Linux 専用

## デフォルトポート

| 用途 | ポート |
|------|--------|
| メインサーバー | 6001 |
| テスト (Direct) | 6081 |
| テスト (SystemdRun) | 6082 |
| Debian テスト | 6083 |

すべてデフォルトで HTTPS（自動生成証明書）です。詳細は [SETUP.md](SETUP.md) を参照。

## デフォルトユーザー

| ユーザー | パスワード |
|---------|-----------|
| testuser | testpass |
| vscode | vscode |
| root | root |

詳細は [USERS.md](USERS.md) を参照。

## 主なコマンド

```bash
make build              # ビルド
make dev                # Dev Container で実行
make test-launchers     # テスト環境起動
make test-down          # テスト環境停止
make deb-test           # Debian パッケージテスト
```

全コマンドは `make help` で確認できます。

## ライセンス

MIT License
