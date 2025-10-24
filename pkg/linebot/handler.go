package linebot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"crypt-info-v2/pkg/coincheck"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

// Handler LINE Bot処理ハンドラー
type Handler struct {
	bot           *linebot.Client
	coincheck     *coincheck.Client
	channelSecret string
}

// NewHandler 新しいLINE Botハンドラーを作成
func NewHandler() (*Handler, error) {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	coincheckAPIKey := os.Getenv("COINCHECK_API_KEY")
	coincheckAPISecret := os.Getenv("COINCHECK_API_SECRET")

	if channelSecret == "" || channelToken == "" {
		return nil, fmt.Errorf("LINE環境変数が設定されていません")
	}

	if coincheckAPIKey == "" || coincheckAPISecret == "" {
		return nil, fmt.Errorf("Coincheck環境変数が設定されていません")
	}

	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		return nil, fmt.Errorf("LINE Bot初期化失敗: %v", err)
	}

	coincheckClient := coincheck.NewClient(coincheckAPIKey, coincheckAPISecret)

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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエストボディ読み込み失敗", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Line-Signature")
	if !h.VerifySignature(body, signature) {
		http.Error(w, "署名検証失敗", http.StatusUnauthorized)
		return
	}

	events, err := h.bot.ParseRequest(r)
	if err != nil {
		http.Error(w, "イベント解析失敗", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		if err := h.handleEvent(event); err != nil {
			fmt.Printf("イベント処理エラー: %v\n", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleEvent イベントを処理
func (h *Handler) handleEvent(event *linebot.Event) error {
	switch event.Type {
	case linebot.EventTypeMessage:
		return h.handleMessage(event)
	default:
		return nil
	}
}

// handleMessage メッセージイベントを処理
func (h *Handler) handleMessage(event *linebot.Event) error {
	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		return h.handleTextMessage(event, message)
	default:
		return nil
	}
}

// handleTextMessage テキストメッセージを処理
func (h *Handler) handleTextMessage(event *linebot.Event, message *linebot.TextMessage) error {
	text := strings.TrimSpace(message.Text)

	var replyMessage string
	var err error

	switch {
	case text == "残高" || text == "balance" || text == "残高確認":
		replyMessage, err = h.getBalanceMessage()
	case text == "資産" || text == "assets" || text == "資産確認":
		replyMessage, err = h.getAssetsMessage()
	case text == "ヘルプ" || text == "help" || text == "?":
		replyMessage = h.getHelpMessage()
	default:
		replyMessage = "コマンドが認識されませんでした。\n「ヘルプ」と送信すると利用可能なコマンドが表示されます。"
	}

	if err != nil {
		replyMessage = "エラーが発生しました: " + err.Error()
	}

	_, err = h.bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
	return err
}

// getBalanceMessage 残高メッセージを取得
func (h *Handler) getBalanceMessage() (string, error) {
	balance, err := h.coincheck.GetBalance()
	if err != nil {
		return "", fmt.Errorf("残高取得失敗: %v", err)
	}

	return h.coincheck.FormatBalanceMessage(balance), nil
}

// getAssetsMessage 資産メッセージを取得（残高と同じ）
func (h *Handler) getAssetsMessage() (string, error) {
	return h.getBalanceMessage()
}

// getHelpMessage ヘルプメッセージを取得
func (h *Handler) getHelpMessage() string {
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
	_, err := h.bot.PushMessage(userID, linebot.NewTextMessage(message)).Do()
	return err
}

// GetBalance 残高を取得（定期実行用）
func (h *Handler) GetBalance() (*coincheck.BalanceResponse, error) {
	return h.coincheck.GetBalance()
}

// FormatBalanceMessage 残高メッセージをフォーマット（定期実行用）
func (h *Handler) FormatBalanceMessage(balance *coincheck.BalanceResponse) string {
	return h.coincheck.FormatBalanceMessage(balance)
}
