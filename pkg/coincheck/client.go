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
	Data    map[string]interface{} `json:"data,omitempty"`
	// 直接フィールドとしても残高データを保持
	JPY   string `json:"jpy,omitempty"`
	BTC   string `json:"btc,omitempty"`
	ETH   string `json:"eth,omitempty"`
	ETC   string `json:"etc,omitempty"`
	LSK   string `json:"lsk,omitempty"`
	FCT   string `json:"fct,omitempty"`
	XRP   string `json:"xrp,omitempty"`
	XEM   string `json:"xem,omitempty"`
	LTC   string `json:"ltc,omitempty"`
	BCH   string `json:"bch,omitempty"`
	MONA  string `json:"mona,omitempty"`
	XLM   string `json:"xlm,omitempty"`
	QTUM  string `json:"qtum,omitempty"`
	DASH  string `json:"dash,omitempty"`
	ZEC   string `json:"zec,omitempty"`
	BAT   string `json:"bat,omitempty"`
	IOST  string `json:"iost,omitempty"`
	ENJ   string `json:"enj,omitempty"`
	OMG   string `json:"omg,omitempty"`
	PLT   string `json:"plt,omitempty"`
	XTZ   string `json:"xtz,omitempty"`
	ATOM  string `json:"atom,omitempty"`
	MKR   string `json:"mkr,omitempty"`
	LINK  string `json:"link,omitempty"`
	COMP  string `json:"comp,omitempty"`
	YFI   string `json:"yfi,omitempty"`
	UNI   string `json:"uni,omitempty"`
	AAVE  string `json:"aave,omitempty"`
	SNX   string `json:"snx,omitempty"`
	CRV   string `json:"crv,omitempty"`
	MATIC string `json:"matic,omitempty"`
	SOL   string `json:"sol,omitempty"`
	AVAX  string `json:"avax,omitempty"`
	DOT   string `json:"dot,omitempty"`
	ADA   string `json:"ada,omitempty"`
	SHIB  string `json:"shib,omitempty"`
	DOGE  string `json:"doge,omitempty"`
	TRX   string `json:"trx,omitempty"`
	NEAR  string `json:"near,omitempty"`
	FTM   string `json:"ftm,omitempty"`
	ALGO  string `json:"algo,omitempty"`
	MANA  string `json:"mana,omitempty"`
	SAND  string `json:"sand,omitempty"`
	AXS   string `json:"axs,omitempty"`
	CHZ   string `json:"chz,omitempty"`
	FLOW  string `json:"flow,omitempty"`
	ICP   string `json:"icp,omitempty"`
	VET   string `json:"vet,omitempty"`
	FIL   string `json:"fil,omitempty"`
	THETA string `json:"theta,omitempty"`
	EOS   string `json:"eos,omitempty"`
	KLAY  string `json:"klay,omitempty"`
	HBAR  string `json:"hbar,omitempty"`
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

	logInfo("残高取得完了", map[string]interface{}{
		"jpy_balance": balance.JPY,
		"btc_balance": balance.BTC,
		"eth_balance": balance.ETH,
		"raw_data": map[string]interface{}{
			"jpy": balance.JPY,
			"btc": balance.BTC,
			"eth": balance.ETH,
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
	jpyBalance := strings.TrimSpace(balance.JPY)
	jpyAmount := 0.0
	if jpyBalance != "" && jpyBalance != "0" {
		if parsed, err := strconv.ParseFloat(jpyBalance, 64); err == nil {
			jpyAmount = parsed
		}
		message += fmt.Sprintf("💴 JPY: %s円\n", jpyBalance)
	}

	// 暗号通貨残高（0以外のみ表示）
	cryptoAssets := map[string]string{
		"BTC":   balance.BTC,
		"ETH":   balance.ETH,
		"ETC":   balance.ETC,
		"LSK":   balance.LSK,
		"FCT":   balance.FCT,
		"XRP":   balance.XRP,
		"XEM":   balance.XEM,
		"LTC":   balance.LTC,
		"BCH":   balance.BCH,
		"MONA":  balance.MONA,
		"XLM":   balance.XLM,
		"QTUM":  balance.QTUM,
		"DASH":  balance.DASH,
		"ZEC":   balance.ZEC,
		"BAT":   balance.BAT,
		"IOST":  balance.IOST,
		"ENJ":   balance.ENJ,
		"OMG":   balance.OMG,
		"PLT":   balance.PLT,
		"XTZ":   balance.XTZ,
		"ATOM":  balance.ATOM,
		"MKR":   balance.MKR,
		"LINK":  balance.LINK,
		"COMP":  balance.COMP,
		"YFI":   balance.YFI,
		"UNI":   balance.UNI,
		"AAVE":  balance.AAVE,
		"SNX":   balance.SNX,
		"CRV":   balance.CRV,
		"MATIC": balance.MATIC,
		"SOL":   balance.SOL,
		"AVAX":  balance.AVAX,
		"DOT":   balance.DOT,
		"ADA":   balance.ADA,
		"SHIB":  balance.SHIB,
		"DOGE":  balance.DOGE,
		"TRX":   balance.TRX,
		"NEAR":  balance.NEAR,
		"FTM":   balance.FTM,
		"ALGO":  balance.ALGO,
		"MANA":  balance.MANA,
		"SAND":  balance.SAND,
		"AXS":   balance.AXS,
		"CHZ":   balance.CHZ,
		"FLOW":  balance.FLOW,
		"ICP":   balance.ICP,
		"VET":   balance.VET,
		"FIL":   balance.FIL,
		"THETA": balance.THETA,
		"EOS":   balance.EOS,
		"KLAY":  balance.KLAY,
		"HBAR":  balance.HBAR,
	}

	// Coincheckでサポートされている通貨ペアのマッピング
	supportedPairs := map[string]string{
		"BTC":   "btc_jpy",
		"ETH":   "eth_jpy",
		"ETC":   "etc_jpy",
		"LSK":   "lsk_jpy",
		"FCT":   "fct_jpy",
		"XRP":   "xrp_jpy",
		"XEM":   "xem_jpy",
		"LTC":   "ltc_jpy",
		"BCH":   "bch_jpy",
		"MONA":  "mona_jpy",
		"XLM":   "xlm_jpy",
		"QTUM":  "qtum_jpy",
		"DASH":  "dash_jpy",
		"ZEC":   "zec_jpy",
		"BAT":   "bat_jpy",
		"IOST":  "iost_jpy",
		"ENJ":   "enj_jpy",
		"OMG":   "omg_jpy",
		"PLT":   "plt_jpy",
		"XTZ":   "xtz_jpy",
		"ATOM":  "atom_jpy",
		"MKR":   "mkr_jpy",
		"LINK":  "link_jpy",
		"COMP":  "comp_jpy",
		"YFI":   "yfi_jpy",
		"UNI":   "uni_jpy",
		"AAVE":  "aave_jpy",
		"SNX":   "snx_jpy",
		"CRV":   "crv_jpy",
		"MATIC": "matic_jpy",
		"SOL":   "sol_jpy",
		"AVAX":  "avax_jpy",
		"DOT":   "dot_jpy",
		"ADA":   "ada_jpy",
		"SHIB":  "shib_jpy",
		"DOGE":  "doge_jpy",
		"TRX":   "trx_jpy",
		"NEAR":  "near_jpy",
		"FTM":   "ftm_jpy",
		"ALGO":  "algo_jpy",
		"MANA":  "mana_jpy",
		"SAND":  "sand_jpy",
		"AXS":   "axs_jpy",
		"CHZ":   "chz_jpy",
		"FLOW":  "flow_jpy",
		"ICP":   "icp_jpy",
		"VET":   "vet_jpy",
		"FIL":   "fil_jpy",
		"THETA": "theta_jpy",
		"EOS":   "eos_jpy",
		"KLAY":  "klay_jpy",
		"HBAR":  "hbar_jpy",
	}

	totalCryptoValue := 0.0

	for symbol, amount := range cryptoAssets {
		trimmedAmount := strings.TrimSpace(amount)
		if trimmedAmount != "" && trimmedAmount != "0" {
			// 金額を数値に変換
			if parsedAmount, err := strconv.ParseFloat(trimmedAmount, 64); err == nil && parsedAmount > 0 {
				message += fmt.Sprintf("🪙 %s: %s", symbol, trimmedAmount)

				// 価格情報を取得して円換算
				if pair, exists := supportedPairs[symbol]; exists {
					if ticker, err := c.GetTicker(pair); err == nil {
						cryptoValue := parsedAmount * ticker.Last
						totalCryptoValue += cryptoValue
						message += fmt.Sprintf(" (約%.0f円)", cryptoValue)
					}
				}
				message += "\n"
			}
		}
	}

	// 合計金額を表示
	totalValue := jpyAmount + totalCryptoValue
	if totalValue > 0 {
		message += fmt.Sprintf("\n📊 合計: 約%.0f円", totalValue)
	} else {
		message += "\n📊 現在の残高: 0円"
	}

	// 日本時間で表示
	jst := time.FixedZone("JST", 9*60*60)
	message += "\n📅 " + time.Now().In(jst).Format("2006年1月2日 15:04 JST")

	return message
}
