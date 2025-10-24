# Coincheck LINE Bot 仕様書

## 概要
Coincheck APIを利用して自分の口座情報を取得し、LINEメッセージで確認できるBotです。毎週土曜日朝6時に自動で残高レポートが送信されます。

## 機能

### 1. LINE Bot機能
- **Webhook受信**: LINE Messaging APIからのメッセージを受信
- **コマンド処理**: ユーザーからのテキストメッセージに応答
- **署名検証**: セキュリティのための署名検証

### 2. Coincheck API連携
- **認証**: HMAC-SHA256署名による認証
- **残高取得**: JPY残高と暗号通貨残高を取得
- **価格情報**: 各暗号通貨の現在価格を取得

### 3. 定期実行機能
- **スケジュール**: 毎週土曜日朝6時（JST）
- **自動送信**: Vercel Cron Jobsによる自動実行
- **ブロードキャスト通知**: 友達登録ユーザー全員に残高レポートを自動送信
- **グループ通知**: 参加しているグループにも残高レポートを自動送信
- **個別通知**: 環境変数で指定された個別IDにも追加送信（オプション）

## 利用可能なコマンド

| コマンド                        | 説明                     |
| ------------------------------- | ------------------------ |
| `残高` / `balance` / `残高確認` | 口座残高を表示           |
| `資産` / `assets` / `資産確認`  | 保有暗号通貨の一覧を表示 |
| `ヘルプ` / `help` / `?`         | ヘルプメッセージを表示   |

## API仕様

### エンドポイント

#### POST /api
LINE Webhook受信用エンドポイント

**リクエストヘッダー:**
- `X-Line-Signature`: LINE署名
- `Content-Type`: application/json

**レスポンス:**
- `200 OK`: 処理成功
- `400 Bad Request`: リクエストエラー
- `401 Unauthorized`: 署名検証失敗

#### POST /api/cron
定期実行用エンドポイント（認証必須）

**リクエストヘッダー:**
- `Authorization`: Bearer {CRON_SECRET}
- `Content-Type`: application/json

**レスポンス:**
- `200 OK`: 処理成功
- `401 Unauthorized`: 認証失敗
- `500 Internal Server Error`: 処理エラー

#### GET /api/health
ヘルスチェック用エンドポイント

**レスポンス:**
- `200 OK`: サービス正常

## 環境変数

### 必須環境変数
- `LINE_CHANNEL_SECRET`: LINE Botのチャネルシークレット
- `LINE_CHANNEL_ACCESS_TOKEN`: LINE Botのチャネルアクセストークン
- `COINCHECK_API_KEY`: Coincheck APIキー
- `COINCHECK_API_SECRET`: Coincheck APIシークレット

### オプション環境変数（定期実行時の追加送信用）
- `LINE_USER_ID`: 個別ユーザーID（ブロードキャストに加えて追加送信）
- `LINE_GROUP_ID`: 個別グループID（ブロードキャストに加えて追加送信）
- `LINE_MULTIPLE_IDS`: 複数のIDをカンマ区切りで指定（例: "user1,group1,user2"）

**注意**: オプション環境変数が設定されていない場合でも、定期実行時は友達登録ユーザー全員と参加しているグループに自動でメッセージが送信されます。

## 定期実行の動作

定期実行時（毎週土曜日朝6時）は以下の順序でメッセージが送信されます：

1. **ブロードキャスト送信**: 友達登録ユーザー全員に送信
2. **グループ送信**: Botが参加している全グループに送信
3. **個別送信**: 環境変数で指定された個別IDに追加送信（設定されている場合のみ）

各送信は独立しており、一部が失敗しても他の送信は継続されます。

## セットアップ手順

### 1. 必要なアカウント・サービス
- [LINE Developers](https://developers.line.biz/) アカウント
- [Coincheck](https://coincheck.com/) アカウント
- [Vercel](https://vercel.com/) アカウント
- [GitHub](https://github.com/) アカウント

### 2. LINE Bot設定
1. LINE Developers Consoleで新しいチャンネルを作成
2. Messaging APIを有効化
3. Webhook URLを設定: `https://your-domain.vercel.app/api`
4. Channel SecretとChannel Access Tokenを取得

### 3. Coincheck API設定
1. Coincheckにログイン
2. API設定ページでAPI KeyとSecret Keyを生成
3. 必要に応じて権限を設定（残高取得権限など）

### 4. Vercelデプロイ
1. GitHubリポジトリをVercelにインポート
2. 環境変数を設定:
   ```
   LINE_CHANNEL_SECRET=your_channel_secret
   LINE_CHANNEL_ACCESS_TOKEN=your_channel_access_token
   COINCHECK_API_KEY=your_api_key
   COINCHECK_API_SECRET=your_api_secret
   LINE_USER_ID=your_line_user_id
   CRON_SECRET=your_random_secret_string
   ```
3. デプロイ実行

### 5. GitHub Actions設定
1. GitHubリポジトリのSecretsに以下を追加:
   ```
   CRON_SECRET=your_random_secret_string
   VERCEL_URL=https://your-domain.vercel.app
   ```
2. Actionsが有効化されていることを確認

## 環境変数

| 変数名                      | 説明                                 | 必須 |
| --------------------------- | ------------------------------------ | ---- |
| `LINE_CHANNEL_SECRET`       | LINE Bot Channel Secret              | ✓    |
| `LINE_CHANNEL_ACCESS_TOKEN` | LINE Bot Channel Access Token        | ✓    |
| `COINCHECK_API_KEY`         | Coincheck API Key                    | ✓    |
| `COINCHECK_API_SECRET`      | Coincheck API Secret                 | ✓    |
| `LINE_USER_ID`              | プッシュ通知送信先のLINE User ID     | ✓    |
| `CRON_SECRET`               | 定期実行エンドポイントの認証トークン | ✓    |

## セキュリティ

- LINE Webhookの署名検証
- Coincheck APIのHMAC-SHA256認証
- 定期実行エンドポイントのBearer認証
- 環境変数による機密情報の管理

## トラブルシューティング

### よくある問題

1. **LINE Botが応答しない**
   - Webhook URLが正しく設定されているか確認
   - Channel SecretとAccess Tokenが正しいか確認

2. **Coincheck APIエラー**
   - API KeyとSecret Keyが正しいか確認
   - API権限が適切に設定されているか確認

3. **定期実行が動作しない**
   - GitHub Actionsが有効化されているか確認
   - Secretsが正しく設定されているか確認
   - CRON_SECRETが一致しているか確認

### ログ確認
VercelのFunction Logsでエラーログを確認できます。

