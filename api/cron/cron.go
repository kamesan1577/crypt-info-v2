package cron

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"crypt-info-v2/pkg/linebot"
)

// LogEntry APIログエントリ
type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	Method    string                 `json:"method,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// logMessage API構造化ログを出力
func logMessage(level, message string, data map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
		Data:      data,
	}

	logJSON, err := json.Marshal(entry)
	if err != nil {
		log.Printf("APIログ出力エラー: %v", err)
		return
	}

	log.Printf("%s", string(logJSON))
}

// logError APIエラーログを出力
func logError(message string, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if err != nil {
		data["error"] = err.Error()
	}
	logMessage("ERROR", message, data)
}

// logInfo API情報ログを出力
func logInfo(message string, data map[string]interface{}) {
	logMessage("INFO", message, data)
}

var lineBotHandler *linebot.Handler

func init() {
	logInfo("Cron API初期化開始", nil)

	var err error
	lineBotHandler, err = linebot.NewHandler()
	if err != nil {
		logError("LINE Bot初期化エラー", err, nil)
		fmt.Printf("LINE Bot初期化エラー: %v\n", err)
	} else {
		logInfo("Cron API初期化完了", nil)
	}
}

// GET 定期実行エンドポイント（Vercel Functions形式）
func GET(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("cron_%d", time.Now().UnixNano())

	logInfo("定期実行リクエスト受信", map[string]interface{}{
		"request_id":  requestID,
		"method":      r.Method,
		"user_agent":  r.Header.Get("User-Agent"),
		"remote_addr": r.RemoteAddr,
	})

	// Vercel Cron Jobsの認証チェック
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "vercel-cron/1.0" {
		logError("Vercel Cron認証失敗", nil, map[string]interface{}{
			"request_id": requestID,
			"user_agent": userAgent,
		})
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 定期実行処理
	if err := executeScheduledTask(requestID); err != nil {
		logError("定期実行タスク失敗", err, map[string]interface{}{
			"request_id": requestID,
		})
		http.Error(w, "Scheduled task failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	logInfo("定期実行タスク完了", map[string]interface{}{
		"request_id":  requestID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Scheduled task completed")
}

// executeScheduledTask 定期実行タスク
func executeScheduledTask(requestID string) error {
	logInfo("定期実行タスク開始", map[string]interface{}{
		"request_id": requestID,
	})

	if lineBotHandler == nil {
		err := fmt.Errorf("LINE Bot not initialized")
		logError("LINE Bot未初期化", err, map[string]interface{}{
			"request_id": requestID,
		})
		return err
	}

	// 残高情報を取得
	logInfo("残高情報取得開始", map[string]interface{}{
		"request_id": requestID,
	})

	balance, err := lineBotHandler.GetBalance()
	if err != nil {
		logError("残高取得失敗", err, map[string]interface{}{
			"request_id": requestID,
		})
		return fmt.Errorf("failed to get balance: %v", err)
	}

	logInfo("残高情報取得完了", map[string]interface{}{
		"request_id":  requestID,
		"jpy_balance": balance.Data.JPY,
		"btc_balance": balance.Data.BTC,
	})

	// メッセージをフォーマット
	message := lineBotHandler.FormatBalanceMessage(balance)
	message = "📅 定期レポート\n\n" + message

	// プッシュメッセージを送信
	userID := os.Getenv("LINE_USER_ID")
	if userID == "" {
		err := fmt.Errorf("LINE_USER_ID not configured")
		logError("LINE_USER_ID未設定", err, map[string]interface{}{
			"request_id": requestID,
		})
		return err
	}

	logInfo("プッシュメッセージ送信開始", map[string]interface{}{
		"request_id":     requestID,
		"user_id":        userID,
		"message_length": len(message),
	})

	if err := lineBotHandler.SendPushMessage(userID, message); err != nil {
		logError("プッシュメッセージ送信失敗", err, map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID,
		})
		return fmt.Errorf("failed to send push message: %v", err)
	}

	logInfo("定期実行タスク完了", map[string]interface{}{
		"request_id": requestID,
		"user_id":    userID,
	})

	return nil
}
