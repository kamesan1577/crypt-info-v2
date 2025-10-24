package health

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
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

// logInfo API情報ログを出力
func logInfo(message string, data map[string]interface{}) {
	logMessage("INFO", message, data)
}

func init() {
	logInfo("Health API初期化完了", nil)
}

// GET ヘルスチェックエンドポイント（Vercel Functions形式）
func GET(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("health_%d", time.Now().UnixNano())

	logInfo("ヘルスチェックリクエスト受信", map[string]interface{}{
		"request_id":  requestID,
		"method":      r.Method,
		"user_agent":  r.Header.Get("User-Agent"),
		"remote_addr": r.RemoteAddr,
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")

	logInfo("ヘルスチェック完了", map[string]interface{}{
		"request_id":  requestID,
		"duration_ms": time.Since(startTime).Milliseconds(),
	})
}
