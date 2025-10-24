package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"crypt-info-v2/pkg/linebot"
)

var lineBotHandler *linebot.Handler

func init() {
	var err error
	lineBotHandler, err = linebot.NewHandler()
	if err != nil {
		fmt.Printf("LINE Bot初期化エラー: %v\n", err)
	}
}

// Handler メインハンドラー（Webhook受信）
func Handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")

	switch path {
	case "":
		// LINE Webhook
		if lineBotHandler != nil {
			lineBotHandler.HandleWebhook(w, r)
		} else {
			http.Error(w, "LINE Bot初期化エラー", http.StatusInternalServerError)
		}
	case "/cron":
		// 定期実行エンドポイント
		handleCron(w, r)
	case "/health":
		// ヘルスチェック
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	default:
		http.NotFound(w, r)
	}
}

// handleCron 定期実行ハンドラー
func handleCron(w http.ResponseWriter, r *http.Request) {
	// Vercel Cron Jobsの認証チェック
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "vercel-cron/1.0" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// メソッドチェック（Vercel Cron JobsはGETリクエストを送信）
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 定期実行処理
	if err := executeScheduledTask(); err != nil {
		http.Error(w, "Scheduled task failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Scheduled task completed")
}

// executeScheduledTask 定期実行タスク
func executeScheduledTask() error {
	if lineBotHandler == nil {
		return fmt.Errorf("LINE Bot not initialized")
	}

	// 残高情報を取得
	balance, err := lineBotHandler.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance: %v", err)
	}

	// メッセージをフォーマット
	message := lineBotHandler.FormatBalanceMessage(balance)
	message = "📅 定期レポート\n\n" + message

	// プッシュメッセージを送信
	userID := os.Getenv("LINE_USER_ID")
	if userID == "" {
		return fmt.Errorf("LINE_USER_ID not configured")
	}

	if err := lineBotHandler.SendPushMessage(userID, message); err != nil {
		return fmt.Errorf("failed to send push message: %v", err)
	}

	return nil
}
