# セットアップガイド

## 概要

wsconsole のセットアップ、ポート設定、HTTPS設定を説明します。

## ポート設定

### デフォルトポート

| 用途 | ポート | プロトコル | 説明 |
|------|--------|----------|------|
| メインサーバー | 6001 | HTTPS | 自動生成証明書 |
| テスト (Direct) | 6081 | HTTPS | root 起動 |
| テスト (SystemdRun) | 6082 | HTTPS | systemd-run 起動 |
| Debian テスト | 6083 | HTTPS | パッケージテスト |

### ポート指定

```bash
# カスタムポート
./wsconsole -addr ":6100"

# HTTP モード（リバースプロキシ背後）
./wsconsole -tls=false -addr ":6001"
```

## HTTPS/TLS 設定

### デフォルト動作

wsconsole はデフォルトで **HTTPS で起動** します。

```bash
./wsconsole
# → https://localhost:6001/
# 証明書: /tmp/wsconsole/cert.pem, key.pem（自動生成）
```

自己署名証明書により、ブラウザに警告が表示されますが、開発環境では無視できます。

### HTTP モード（リバースプロキシ対応）

リバースプロキシで TLS 終端を処理する場合：

```bash
./wsconsole -tls=false -addr ":6001"
# → http://localhost:6001/ （内部のみ）
```

Nginx 設定例：

```nginx
server {
    listen 443 ssl;
    server_name example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:6001/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

起動コマンド：
```bash
./wsconsole -tls=false -addr ":6001"
```

### カスタム証明書

Let's Encrypt などから取得した証明書を使用：

```bash
./wsconsole -cert /etc/wsconsole/cert.pem -key /etc/wsconsole/key.pem
```

### パスプレフィックス対応

リバースプロキシでパスを変更する場合：

```bash
./wsconsole -path-prefix /api/wsconsole
# → https://localhost:6001/api/wsconsole/ws
```

## アクセス方法

### ブラウザ

```
https://localhost:6001/
```

自己署名証明書の警告が表示される場合：
1. 「詳細設定」をクリック
2. 「localhost にアクセス（危険）」をクリック

### ヘルスチェック

```bash
curl -k https://localhost:6001/healthz
```

### WebSocket （JavaScript）

```javascript
// プロトコルを自動検出
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsUrl = `${protocol}//localhost:6001/ws`;
```

## コマンドラインフラグ

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-addr` | `:6001` | リッスンアドレス |
| `-tls` | `true` | HTTPS を有効化 |
| `-cert` | (自動生成) | 証明書ファイルパス |
| `-key` | (自動生成) | 秘密鍵ファイルパス |
| `-path-prefix` | なし | パスプレフィックス（リバプロ用） |
| `-launcher` | `auto` | 起動戦略: auto, direct, systemd-run |
| `-static` | `./deploy/static` | 静的ファイルディレクトリ |
| `-log` | `info` | ログレベル: debug, info, warn, error |

## Docker での実行

### 自動生成証明書

```bash
docker run -p 6001:6001 wsconsole:latest
```

### 外部証明書をマウント

```bash
docker run -p 6001:6001 \
  -v /path/to/cert.pem:/etc/wsconsole/cert.pem:ro \
  -v /path/to/key.pem:/etc/wsconsole/key.pem:ro \
  wsconsole:latest \
  wsconsole -cert /etc/wsconsole/cert.pem -key /etc/wsconsole/key.pem
```

## トラブルシューティング

### ポートが使用中

```bash
# 別のポートを使用
./wsconsole -addr ":6002"

# またはプロセスを確認
lsof -i :6001
```

### 証明書エラー

自動生成証明書の警告は開発環境では無視できます。本番環境では正式な証明書を使用してください。

### 接続できない

```bash
# ファイアウォール確認
sudo ufw allow 6001

# ポート確認
netstat -tlnp | grep 6001

# ログ確認
./wsconsole -log debug
```

## Systemd サービス

```ini
[Unit]
Description=wsconsole WebSocket Terminal Gateway
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/wsconsole -cert /etc/wsconsole/cert.pem -key /etc/wsconsole/key.pem
Restart=on-failure
User=wsconsole

[Install]
WantedBy=multi-user.target
```

## 詳細ドキュメント

- [README.md](README.md) - プロジェクト概要
- [TESTING.md](TESTING.md) - テスト手順
- [USERS.md](USERS.md) - ユーザー情報
