# ローカルテスト用のREADME

## ローカルテスト方法

### 1. 環境変数の設定

1. `.env.sample`をコピーして`.env`ファイルを作成:
```bash
cp .env.sample .env
```

2. `.env`ファイルを編集して実際の値を設定:
```bash
# LINE Bot設定
LINE_CHANNEL_SECRET=実際のチャンネルシークレット
LINE_CHANNEL_ACCESS_TOKEN=実際のアクセストークン
LINE_USER_ID=実際のユーザーID

# Coincheck API設定
COINCHECK_API_KEY=実際のAPIキー
COINCHECK_API_SECRET=実際のシークレットキー

# 定期実行設定
CRON_SECRET=ランダムな文字列

# ローカル開発用
PORT=8080
```

### 2. 依存関係のインストール

```bash
# Go 1.25.3を使用する場合
export PATH=$HOME/go/bin:$PATH
export GOTOOLCHAIN=go1.25.3
go mod tidy
```

### 3. ローカルサーバーの起動

```bash
# Go 1.25.3を使用する場合
export PATH=$HOME/go/bin:$PATH
export GOTOOLCHAIN=go1.25.3
go run main.go
```

サーバーが起動すると以下のURLが利用可能になります:
- `http://localhost:8080/api` - LINE Webhook
- `http://localhost:8080/api/cron` - 定期実行テスト
- `http://localhost:8080/api/health` - ヘルスチェック

### 4. LINE Webhookのテスト

#### ngrokを使用したテスト（推奨）

1. ngrokをインストール:
```bash
# macOS
brew install ngrok

# または直接ダウンロード
# https://ngrok.com/download
```

2. ngrokでトンネルを作成:
```bash
ngrok http 8080
```

3. ngrokが表示するHTTPS URL（例: `https://abc123.ngrok.io`）をLINE Developers ConsoleのWebhook URLに設定:
```
https://abc123.ngrok.io/api
```

4. LINE Botにメッセージを送信してテスト

#### curlを使用したテスト

```bash
# ヘルスチェック
curl http://localhost:8080/api/health

# 定期実行テスト（認証が必要）
curl -X POST \
  -H "Authorization: Bearer your_cron_secret" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/cron
```

### 5. LINE User IDの取得方法

1. LINE Botに友達追加
2. Botにメッセージを送信
3. VercelのFunction LogsまたはローカルログでUser IDを確認
4. `.env`ファイルの`LINE_USER_ID`に設定

### 6. Coincheck APIのテスト

```bash
# 残高取得テスト（認証が必要）
curl -X GET \
  -H "ACCESS-KEY: your_api_key" \
  -H "ACCESS-NONCE: $(date +%s)" \
  -H "ACCESS-SIGNATURE: your_signature" \
  https://coincheck.com/api/accounts/balance
```

### 7. デバッグ方法

#### ログの確認
```bash
# 詳細ログを有効にする場合
export DEBUG=true
go run main.go
```

#### エラーの確認
- LINE Botのエラー: Vercel Function Logs
- Coincheck APIのエラー: APIレスポンスを確認
- ローカルテスト: ターミナルのログ出力を確認

### 8. よくある問題

1. **環境変数が読み込まれない**
   - `.env`ファイルが正しい場所にあるか確認
   - ファイル名が`.env`（先頭にドット）になっているか確認

2. **LINE Botが応答しない**
   - Webhook URLが正しく設定されているか確認
   - ngrokが起動しているか確認
   - Channel SecretとAccess Tokenが正しいか確認

3. **Coincheck APIエラー**
   - API KeyとSecret Keyが正しいか確認
   - API権限が適切に設定されているか確認

4. **定期実行が動作しない**
   - CRON_SECRETが一致しているか確認
   - Authorizationヘッダーが正しく設定されているか確認
