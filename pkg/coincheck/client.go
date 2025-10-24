package coincheck

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// LogEntry Coincheckログエントリ
type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	Method    string                 `json:"method,omitempty"`
	URL       string                 `json:"url,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// logMessage Coincheck構造化ログを出力
func logMessage(level, message string, data map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
		Data:      data,
	}

	logJSON, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Coincheckログ出力エラー: %v", err)
		return
	}

	log.Printf("%s", string(logJSON))
}

// logError Coincheckエラーログを出力
func logError(message string, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if err != nil {
		data["error"] = err.Error()
	}
	logMessage("ERROR", message, data)
}

// logInfo Coincheck情報ログを出力
func logInfo(message string, data map[string]interface{}) {
	logMessage("INFO", message, data)
}

// logDebug Coincheckデバッグログを出力
func logDebug(message string, data map[string]interface{}) {
	logMessage("DEBUG", message, data)
}

// Client Coincheck API クライアント
type Client struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

// BalanceResponse 残高レスポンス
type BalanceResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

// TickerResponse ティッカーレスポンス（価格情報）
type TickerResponse struct {
	Last      float64 `json:"last"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
}

// NewClient 新しいCoincheckクライアントを作成
func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		APIKey:    apiKey,
		APISecret: apiSecret,
		BaseURL:   "https://coincheck.com",
	}
}

// createSignature HMAC-SHA256署名を作成
func (c *Client) createSignature(nonce, url, body string) string {
	message := nonce + url + body
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// makeRequest 認証付きリクエストを実行
func (c *Client) makeRequest(method, endpoint, body string) (*http.Response, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("coincheck_%d", time.Now().UnixNano())
	url := c.BaseURL + endpoint
	nonce := strconv.FormatInt(time.Now().Unix(), 10)

	logDebug("Coincheck APIリクエスト開始", map[string]interface{}{
		"request_id": requestID,
		"method":     method,
		"url":        url,
		"nonce":      nonce,
	})

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		logError("HTTPリクエスト作成失敗", err, map[string]interface{}{
			"request_id": requestID,
			"method":     method,
			"url":        url,
		})
		return nil, err
	}

	if body != "" {
		req.Body = io.NopCloser(strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	signature := c.createSignature(nonce, url, body)

	req.Header.Set("ACCESS-KEY", c.APIKey)
	req.Header.Set("ACCESS-NONCE", nonce)
	req.Header.Set("ACCESS-SIGNATURE", signature)

	logDebug("Coincheck API認証ヘッダー設定", map[string]interface{}{
		"request_id": requestID,
		"access_key": c.APIKey,
		"nonce":      nonce,
		"signature":  signature,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)

	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		logError("Coincheck APIリクエスト失敗", err, map[string]interface{}{
			"request_id":  requestID,
			"method":      method,
			"url":         url,
			"duration_ms": duration,
		})
		return nil, err
	}

	logInfo("Coincheck APIリクエスト完了", map[string]interface{}{
		"request_id":  requestID,
		"method":      method,
		"url":         url,
		"status":      resp.StatusCode,
		"duration_ms": duration,
	})

	return resp, nil
}

// GetBalance 口座残高を取得
func (c *Client) GetBalance() (*BalanceResponse, error) {
	logInfo("残高取得リクエスト開始", nil)

	resp, err := c.makeRequest("GET", "/api/accounts/balance", "")
	if err != nil {
		logError("残高取得リクエスト失敗", err, nil)
		return nil, fmt.Errorf("残高取得リクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError("残高取得失敗", nil, map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("残高取得失敗: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("レスポンス読み込み失敗", err, nil)
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	var balance BalanceResponse
	if err := json.Unmarshal(body, &balance); err != nil {
		logError("JSON解析失敗", err, map[string]interface{}{
			"body_size": len(body),
		})
		return nil, fmt.Errorf("JSON解析失敗: %v", err)
	}

	// デバッグ: 実際のAPIレスポンスをログ出力
	logInfo("Coincheck APIレスポンス詳細", map[string]interface{}{
		"raw_response":  string(body),
		"success":       balance.Success,
		"response_size": len(body),
	})

	if !balance.Success {
		logError("API呼び出し失敗", nil, map[string]interface{}{
			"response_body": string(body),
		})
		return nil, fmt.Errorf("API呼び出し失敗")
	}

	// 主要な残高を取得
	jpyBalance := ""
	btcBalance := ""
	ethBalance := ""

	if jpyVal, exists := balance.Data["jpy"]; exists {
		if jpyStr, ok := jpyVal.(string); ok {
			jpyBalance = jpyStr
		}
	}
	if btcVal, exists := balance.Data["btc"]; exists {
		if btcStr, ok := btcVal.(string); ok {
			btcBalance = btcStr
		}
	}
	if ethVal, exists := balance.Data["eth"]; exists {
		if ethStr, ok := ethVal.(string); ok {
			ethBalance = ethStr
		}
	}

	logInfo("残高取得完了", map[string]interface{}{
		"jpy_balance": jpyBalance,
		"btc_balance": btcBalance,
		"eth_balance": ethBalance,
		"raw_data": map[string]interface{}{
			"jpy": jpyBalance,
			"btc": btcBalance,
			"eth": ethBalance,
		},
	})

	return &balance, nil
}

// GetTicker 指定通貨の価格情報を取得
func (c *Client) GetTicker(pair string) (*TickerResponse, error) {
	logInfo("価格情報取得リクエスト開始", map[string]interface{}{
		"pair": pair,
	})

	url := fmt.Sprintf("%s/api/ticker?pair=%s", c.BaseURL, pair)

	resp, err := http.Get(url)
	if err != nil {
		logError("価格取得リクエスト失敗", err, map[string]interface{}{
			"pair": pair,
			"url":  url,
		})
		return nil, fmt.Errorf("価格取得リクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logError("価格取得失敗", nil, map[string]interface{}{
			"pair":        pair,
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("価格取得失敗: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("レスポンス読み込み失敗", err, map[string]interface{}{
			"pair": pair,
		})
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	var ticker TickerResponse
	if err := json.Unmarshal(body, &ticker); err != nil {
		logError("JSON解析失敗", err, map[string]interface{}{
			"pair":      pair,
			"body_size": len(body),
		})
		return nil, fmt.Errorf("JSON解析失敗: %v", err)
	}

	logInfo("価格情報取得完了", map[string]interface{}{
		"pair": pair,
		"last": ticker.Last,
	})

	return &ticker, nil
}

// FormatBalanceMessage 残高情報をLINEメッセージ用にフォーマット
func (c *Client) FormatBalanceMessage(balance *BalanceResponse) string {
	message := "💰 Coincheck 口座残高\n\n"

	// JPY残高を取得
	jpyBalance := ""
	if jpyVal, exists := balance.Data["jpy"]; exists {
		if jpyStr, ok := jpyVal.(string); ok {
			jpyBalance = strings.TrimSpace(jpyStr)
		}
	}

	if jpyBalance != "" && jpyBalance != "0" {
		message += fmt.Sprintf("💴 JPY: %s円\n", jpyBalance)
	} else if jpyBalance == "" {
		// 空文字列の場合は0として扱う
		message += "💴 JPY: 0円\n"
	}

	// 暗号通貨残高（0以外のみ表示）
	cryptoAssets := []string{
		"btc", "eth", "etc", "lsk", "fct", "xrp", "xem", "ltc", "bch", "mona",
		"xlm", "qtum", "dash", "zec", "bat", "iost", "enj", "omg", "plt", "xtz",
		"atom", "mkr", "link", "comp", "yfi", "uni", "aave", "snx", "crv", "matic",
		"sol", "avax", "dot", "ada", "shib", "doge", "trx", "near", "ftm", "algo",
		"mana", "sand", "axs", "chz", "flow", "icp", "vet", "fil", "theta", "eos",
		"klay", "hbar",
	}

	hasCryptoBalance := false
	for _, symbol := range cryptoAssets {
		if val, exists := balance.Data[symbol]; exists {
			if amountStr, ok := val.(string); ok {
				trimmedAmount := strings.TrimSpace(amountStr)
				if trimmedAmount != "" && trimmedAmount != "0" {
					message += fmt.Sprintf("🪙 %s: %s\n", strings.ToUpper(symbol), trimmedAmount)
					hasCryptoBalance = true
				}
			}
		}
	}

	// 残高がすべて0または空の場合の処理を改善
	if jpyBalance == "" || jpyBalance == "0" {
		if !hasCryptoBalance {
			message += "\n📊 現在の残高: 0円"
		}
	}

	// 日本時間で表示
	jst := time.FixedZone("JST", 9*60*60)
	message += "\n📅 " + time.Now().In(jst).Format("2006年1月2日 15:04 JST")

	return message
}
