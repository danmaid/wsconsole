# テストユーザー

## デフォルトユーザー

wsconsole に接続するとログインプロンプトが表示されます。以下のユーザーを使用してください。

| ユーザー名 | パスワード | 用途 |
|-----------|-----------|------|
| testuser | testpass | テスト用（推奨） |
| vscode | vscode | VS Code Dev Container |
| root | root | 管理者 |

## 新規ユーザー作成

```bash
# コンテナ内で実行
docker exec -it wsconsole-test-direct bash
useradd -m -s /bin/bash newuser
echo "newuser:password" | chpasswd
```

## プロンプト表示

- **root**: `#` で終了
- **一般ユーザー**: `$` で終了
