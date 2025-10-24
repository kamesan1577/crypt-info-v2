package linebot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"crypt-info-v2/pkg/coincheck"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

// LogLevel ログレベル
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry ログエントリ
type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Handler LINE Bot処理ハンドラー
type Handler struct {
	bot           *linebot.Client
	coincheck     *coincheck.Client
	channelSecret string
}

// logMessage 構造化ログを出力
func logMessage(level LogLevel, message string, data map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
		Data:      data,
	}

	logJSON, err := json.Marshal(entry)
	if err != nil {
		log.Printf("ログ出力エラー: %v", err)
		return
	}

	log.Printf("%s", string(logJSON))
}

// logError エラーログを出力
func logError(message string, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if err != nil {
		data["error"] = err.Error()
	}
	logMessage(LogLevelError, message, data)
}

// logInfo 情報ログを出力
func logInfo(message string, data map[string]interface{}) {
	logMessage(LogLevelInfo, message, data)
}

// logDebug デバッグログを出力
func logDebug(message string, data map[string]interface{}) {
	logMessage(LogLevelDebug, message, data)
}

// NewHandler 新しいLINE Botハンドラーを作成
func NewHandler() (*Handler, error) {
	logInfo("LINE Botハンドラーの初期化を開始", nil)

	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	coincheckAPIKey := os.Getenv("COINCHECK_API_KEY")
	coincheckAPISecret := os.Getenv("COINCHECK_API_SECRET")

	if channelSecret == "" || channelToken == "" {
		logError("LINE環境変数が設定されていません", nil, map[string]interface{}{
			"channel_secret_set": channelSecret != "",
			"channel_token_set":  channelToken != "",
		})
		return nil, fmt.Errorf("LINE環境変数が設定されていません")
	}

	if coincheckAPIKey == "" || coincheckAPISecret == "" {
		logError("Coincheck環境変数が設定されていません", nil, map[string]interface{}{
			"api_key_set":    coincheckAPIKey != "",
			"api_secret_set": coincheckAPISecret != "",
		})
		return nil, fmt.Errorf("Coincheck環境変数が設定されていません")
	}

	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		logError("LINE Bot初期化失敗", err, nil)
		return nil, fmt.Errorf("LINE Bot初期化失敗: %v", err)
	}

	coincheckClient := coincheck.NewClient(coincheckAPIKey, coincheckAPISecret)

	logInfo("LINE Botハンドラーの初期化完了", nil)

	return &Handler{
		bot:           bot,
		coincheck:     coincheckClient,
		channelSecret: channelSecret,
	}, nil
}

// VerifySignature 署名を検証
func (h *Handler) VerifySignature(body []byte, signature string) bool {
	hash := hmac.New(sha256.New, []byte(h.channelSecret))
	hash.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature == expectedSignature
}

// HandleWebhook Webhookリクエストを処理
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	logInfo("Webhookリクエスト受信", map[string]interface{}{
		"request_id":  requestID,
		"method":      r.Method,
		"user_agent":  r.Header.Get("User-Agent"),
		"remote_addr": r.RemoteAddr,
	})

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logError("リクエストボディ読み込み失敗", err, map[string]interface{}{
			"request_id": requestID,
		})
		http.Error(w, "リクエストボディ読み込み失敗", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Line-Signature")
	if !h.VerifySignature(body, signature) {
		logError("署名検証失敗", nil, map[string]interface{}{
			"request_id": requestID,
			"signature":  signature,
		})
		http.Error(w, "署名検証失敗", http.StatusUnauthorized)
		return
	}

	events, err := h.bot.ParseRequest(r)
	if err != nil {
		logError("イベント解析失敗", err, map[string]interface{}{
			"request_id": requestID,
			"body_size":  len(body),
		})
		http.Error(w, "イベント解析失敗", http.StatusBadRequest)
		return
	}

	logInfo("イベント解析完了", map[string]interface{}{
		"request_id":  requestID,
		"event_count": len(events),
	})

	processedCount := 0
	errorCount := 0

	for i, event := range events {
		eventStartTime := time.Now()
		if err := h.handleEvent(event, requestID); err != nil {
			logError("イベント処理エラー", err, map[string]interface{}{
				"request_id":  requestID,
				"event_index": i,
				"event_type":  event.Type,
				"user_id":     event.Source.UserID,
			})
			errorCount++
		} else {
			processedCount++
		}

		eventDuration := time.Since(eventStartTime).Milliseconds()
		logDebug("イベント処理完了", map[string]interface{}{
			"request_id":  requestID,
			"event_index": i,
			"event_type":  event.Type,
			"duration_ms": eventDuration,
		})
	}

	totalDuration := time.Since(startTime).Milliseconds()
	logInfo("Webhookリクエスト処理完了", map[string]interface{}{
		"request_id":        requestID,
		"total_duration_ms": totalDuration,
		"processed_count":   processedCount,
		"error_count":       errorCount,
	})

	w.WriteHeader(http.StatusOK)
}

// handleEvent イベントを処理
func (h *Handler) handleEvent(event *linebot.Event, requestID string) error {
	logDebug("イベント処理開始", map[string]interface{}{
		"request_id":  requestID,
		"event_type":  event.Type,
		"user_id":     event.Source.UserID,
		"reply_token": event.ReplyToken,
	})

	switch event.Type {
	case linebot.EventTypeMessage:
		return h.handleMessage(event, requestID)
	default:
		logDebug("未対応のイベントタイプ", map[string]interface{}{
			"request_id": requestID,
			"event_type": event.Type,
		})
		return nil
	}
}

// handleMessage メッセージイベントを処理
func (h *Handler) handleMessage(event *linebot.Event, requestID string) error {
	logDebug("メッセージイベント処理開始", map[string]interface{}{
		"request_id": requestID,
		"user_id":    event.Source.UserID,
	})

	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		return h.handleTextMessage(event, message, requestID)
	default:
		logDebug("未対応のメッセージタイプ", map[string]interface{}{
			"request_id":   requestID,
			"message_type": fmt.Sprintf("%T", event.Message),
		})
		return nil
	}
}

// handleTextMessage テキストメッセージを処理
func (h *Handler) handleTextMessage(event *linebot.Event, message *linebot.TextMessage, requestID string) error {
	text := strings.TrimSpace(message.Text)

	logInfo("テキストメッセージ受信", map[string]interface{}{
		"request_id":     requestID,
		"user_id":        event.Source.UserID,
		"message_text":   text,
		"message_length": len(text),
	})

	var replyMessage string
	var err error

	switch {
	case text == "残高" || text == "balance" || text == "残高確認":
		logInfo("残高確認コマンド実行", map[string]interface{}{
			"request_id": requestID,
			"user_id":    event.Source.UserID,
		})
		replyMessage, err = h.getBalanceMessage()
	case text == "資産" || text == "assets" || text == "資産確認":
		logInfo("資産確認コマンド実行", map[string]interface{}{
			"request_id": requestID,
			"user_id":    event.Source.UserID,
		})
		replyMessage, err = h.getAssetsMessage()
	case text == "ヘルプ" || text == "help" || text == "?":
		logInfo("ヘルプコマンド実行", map[string]interface{}{
			"request_id": requestID,
			"user_id":    event.Source.UserID,
		})
		replyMessage = h.getHelpMessage()
	default:
		logInfo("未認識コマンド", map[string]interface{}{
			"request_id": requestID,
			"user_id":    event.Source.UserID,
			"command":    text,
		})
		replyMessage = "コマンドが認識されませんでした。\n「ヘルプ」と送信すると利用可能なコマンドが表示されます。"
	}

	if err != nil {
		logError("メッセージ処理エラー", err, map[string]interface{}{
			"request_id": requestID,
			"user_id":    event.Source.UserID,
			"command":    text,
		})
		replyMessage = "エラーが発生しました: " + err.Error()
	}

	_, err = h.bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
	if err != nil {
		logError("LINE返信メッセージ送信失敗", err, map[string]interface{}{
			"request_id":  requestID,
			"user_id":     event.Source.UserID,
			"reply_token": event.ReplyToken,
		})
		return err
	}

	logInfo("LINE返信メッセージ送信完了", map[string]interface{}{
		"request_id":   requestID,
		"user_id":      event.Source.UserID,
		"reply_length": len(replyMessage),
	})

	return nil
}

// getBalanceMessage 残高メッセージを取得
func (h *Handler) getBalanceMessage() (string, error) {
	logDebug("Coincheck残高取得開始", nil)

	balance, err := h.coincheck.GetBalance()
	if err != nil {
		logError("Coincheck残高取得失敗", err, nil)
		return "", fmt.Errorf("残高取得失敗: %v", err)
	}

	logInfo("Coincheck残高取得完了", map[string]interface{}{
		"jpy_balance": balance.Data.JPY,
		"btc_balance": balance.Data.BTC,
	})

	return h.coincheck.FormatBalanceMessage(balance), nil
}

// getAssetsMessage 資産メッセージを取得（残高と同じ）
func (h *Handler) getAssetsMessage() (string, error) {
	return h.getBalanceMessage()
}

// getHelpMessage ヘルプメッセージを取得
func (h *Handler) getHelpMessage() string {
	logDebug("ヘルプメッセージ生成", nil)
	return `🤖 Coincheck LINE Bot ヘルプ

利用可能なコマンド:
• 残高 / balance / 残高確認
  → 口座残高を表示

• 資産 / assets / 資産確認
  → 保有暗号通貨の一覧を表示

• ヘルプ / help / ?
  → このヘルプを表示

毎週土曜日朝6時に自動で残高情報が送信されます。`
}

// SendPushMessage プッシュメッセージを送信（定期実行用）
func (h *Handler) SendPushMessage(userID, message string) error {
	logInfo("プッシュメッセージ送信開始", map[string]interface{}{
		"user_id":        userID,
		"message_length": len(message),
	})

	_, err := h.bot.PushMessage(userID, linebot.NewTextMessage(message)).Do()
	if err != nil {
		logError("プッシュメッセージ送信失敗", err, map[string]interface{}{
			"user_id": userID,
		})
		return err
	}

	logInfo("プッシュメッセージ送信完了", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

// GetBalance 残高を取得（定期実行用）
func (h *Handler) GetBalance() (*coincheck.BalanceResponse, error) {
	return h.coincheck.GetBalance()
}

// FormatBalanceMessage 残高メッセージをフォーマット（定期実行用）
func (h *Handler) FormatBalanceMessage(balance *coincheck.BalanceResponse) string {
	return h.coincheck.FormatBalanceMessage(balance)
}
