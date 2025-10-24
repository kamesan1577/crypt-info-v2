package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	logInfo("API初期化開始", nil)

	var err error
	lineBotHandler, err = linebot.NewHandler()
	if err != nil {
		logError("LINE Bot初期化エラー", err, nil)
		fmt.Printf("LINE Bot初期化エラー: %v\n", err)
	} else {
		// Webhook接続確認
		if err := lineBotHandler.VerifyWebhookConnection(); err != nil {
			logError("Webhook接続確認エラー", err, nil)
			fmt.Printf("Webhook接続確認エラー: %v\n", err)
		}
		logInfo("API初期化完了", nil)
	}
}

// Handler メインハンドラー（Webhook受信）
func Handler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("api_%d", time.Now().UnixNano())
	path := strings.TrimPrefix(r.URL.Path, "/api")

	logInfo("APIリクエスト受信", map[string]interface{}{
		"request_id":  requestID,
		"method":      r.Method,
		"path":        path,
		"user_agent":  r.Header.Get("User-Agent"),
		"remote_addr": r.RemoteAddr,
	})

	var statusCode int
	var responseMessage string

	switch path {
	case "":
		// ルートパス - ヘルスチェックとして使用
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Crypt Info API is running")
		statusCode = http.StatusOK
		responseMessage = "API is running"
	case "/webhook":
		// LINE Webhook
		if lineBotHandler != nil {
			lineBotHandler.HandleWebhook(w, r)
			statusCode = http.StatusOK
			responseMessage = "Webhook processed"
		} else {
			logError("LINE Bot初期化エラー", nil, map[string]interface{}{
				"request_id": requestID,
			})
			http.Error(w, "LINE Bot初期化エラー", http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
			responseMessage = "LINE Bot initialization error"
		}
	case "/cron":
		// 定期実行エンドポイント
		statusCode, responseMessage = handleCron(w, r, requestID)
	case "/health":
		// ヘルスチェック
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
		statusCode = http.StatusOK
		responseMessage = "Health check OK"
	default:
		logError("未対応のパス", nil, map[string]interface{}{
			"request_id": requestID,
			"path":       path,
		})
		http.NotFound(w, r)
		statusCode = http.StatusNotFound
		responseMessage = "Not found"
	}

	duration := time.Since(startTime).Milliseconds()
	logInfo("APIリクエスト処理完了", map[string]interface{}{
		"request_id":  requestID,
		"method":      r.Method,
		"path":        path,
		"status":      statusCode,
		"duration_ms": duration,
		"response":    responseMessage,
	})
}

// handleCron 定期実行ハンドラー
func handleCron(w http.ResponseWriter, r *http.Request, requestID string) (int, string) {
	logInfo("定期実行リクエスト受信", map[string]interface{}{
		"request_id": requestID,
		"user_agent": r.Header.Get("User-Agent"),
	})

	// Vercel Cron Jobsの認証チェック
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "vercel-cron/1.0" {
		logError("Vercel Cron認証失敗", nil, map[string]interface{}{
			"request_id": requestID,
			"user_agent": userAgent,
		})
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return http.StatusUnauthorized, "Unauthorized"
	}

	// メソッドチェック（Vercel Cron JobsはGETリクエストを送信）
	if r.Method != "GET" {
		logError("不正なメソッド", nil, map[string]interface{}{
			"request_id": requestID,
			"method":     r.Method,
		})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return http.StatusMethodNotAllowed, "Method not allowed"
	}

	// 定期実行処理
	if err := executeScheduledTask(requestID); err != nil {
		logError("定期実行タスク失敗", err, map[string]interface{}{
			"request_id": requestID,
		})
		http.Error(w, "Scheduled task failed: "+err.Error(), http.StatusInternalServerError)
		return http.StatusInternalServerError, "Scheduled task failed"
	}

	logInfo("定期実行タスク完了", map[string]interface{}{
		"request_id": requestID,
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Scheduled task completed")
	return http.StatusOK, "Scheduled task completed"
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
		"jpy_balance": balance.JPY,
		"btc_balance": balance.BTC,
	})

	// メッセージをフォーマット
	message := lineBotHandler.FormatBalanceMessage(balance)
	message = "📅 定期レポート\n\n" + message

	// プッシュメッセージを送信
	userID := os.Getenv("LINE_USER_ID")
	groupID := os.Getenv("LINE_GROUP_ID")
	multipleIDs := os.Getenv("LINE_MULTIPLE_IDS")

	var sendTargets []string
	var targetNames []string

	// ユーザーIDが設定されている場合
	if userID != "" {
		sendTargets = append(sendTargets, userID)
		targetNames = append(targetNames, "ユーザー")
	}

	// グループIDが設定されている場合
	if groupID != "" {
		sendTargets = append(sendTargets, groupID)
		targetNames = append(targetNames, "グループ")
	}

	// 複数IDが設定されている場合（カンマ区切り）
	if multipleIDs != "" {
		ids := strings.Split(multipleIDs, ",")
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				sendTargets = append(sendTargets, id)
				targetNames = append(targetNames, "複数ID")
			}
		}
	}

	// 送信先が設定されていない場合
	if len(sendTargets) == 0 {
		err := fmt.Errorf("送信先が設定されていません（LINE_USER_ID、LINE_GROUP_ID、LINE_MULTIPLE_IDSのいずれかが必要）")
		logError("送信先未設定", err, map[string]interface{}{
			"request_id": requestID,
		})
		return err
	}

	logInfo("プッシュメッセージ送信開始", map[string]interface{}{
		"request_id":     requestID,
		"target_count":   len(sendTargets),
		"target_names":   targetNames,
		"message_length": len(message),
	})

	// 各送信先に個別にプッシュメッセージを送信
	var errors []string
	successCount := 0

	for i, target := range sendTargets {
		if err := lineBotHandler.SendPushMessage(target, message); err != nil {
			logError("プッシュメッセージ送信失敗", err, map[string]interface{}{
				"request_id":   requestID,
				"target":       target,
				"target_index": i,
			})
			errors = append(errors, fmt.Sprintf("target %s: %v", target, err))
		} else {
			successCount++
			logInfo("プッシュメッセージ送信成功", map[string]interface{}{
				"request_id":   requestID,
				"target":       target,
				"target_index": i,
			})
		}
	}

	// 全ての送信が失敗した場合はエラーを返す
	if successCount == 0 {
		return fmt.Errorf("全ての送信先への送信に失敗しました: %s", strings.Join(errors, "; "))
	}

	// 一部の送信が失敗した場合は警告ログを出力
	if len(errors) > 0 {
		logError("一部の送信先への送信に失敗", nil, map[string]interface{}{
			"request_id":    requestID,
			"success_count": successCount,
			"error_count":   len(errors),
			"errors":        errors,
		})
	}

	logInfo("定期実行タスク完了", map[string]interface{}{
		"request_id":   requestID,
		"target_count": len(sendTargets),
		"target_names": targetNames,
	})

	return nil
}
