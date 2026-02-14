# wsconsole のテストユーザー

## 利用可能なユーザー

wsconsole に http://localhost:8080/ で接続するとログインプロンプトが表示されます。以下の認証情報を利用してください。

### テストユーザー（推奨）
- **ユーザー名**: `testuser`
- **パスワード**: `testpass`
- **シェル**: `/bin/bash`
- **ホーム**: `/home/testuser`

### VS Code ユーザー
- **ユーザー名**: `vscode`
- **パスワード**: `vscode`
- **シェル**: `/bin/bash`
- **ホーム**: `/home/vscode`

### root ユーザー
- **ユーザー名**: `root`
- **パスワード**: `root`
- **シェル**: `/bin/bash`
- **ホーム**: `/root`

## 追加ユーザーの作成

新しいテストユーザーを作成する場合:

```bash
# ホームディレクトリと bash シェルを指定して作成
sudo useradd -m -s /bin/bash username

# パスワード設定
echo "username:password" | sudo chpasswd

# 対話的に設定する場合
sudo passwd username
```

## ログイン確認

コマンドラインでログインを確認:

```bash
# テストユーザーでログイン
su - testuser

# /bin/login を直接利用
login testuser
```

## 注意事項

- これらは開発用の認証情報です
- 本番環境では使用しないでください
- 認証は `/bin/login` が処理します
- 失敗したログインはログに記録されます

## プロンプト表示

ログイン後のプロンプトはユーザーによって異なります:

- **root ユーザー**: `root@hostname:/path#`（末尾が `#`）
- **一般ユーザー**: `username@hostname:/path$`（末尾が `$`）

これは Unix/Linux の標準的な慣習で、特権ユーザー（root）と一般ユーザーを区別します。

### Dev Container の注意点

Dev Container では開発用のカスタムプロンプト（例: `root ➜ ~`）が表示されます。標準プロンプトを確認したい場合はテストコンテナを利用してください。

```bash
make test-launchers
# http://localhost:8081/ または http://localhost:8082/ に接続
```
