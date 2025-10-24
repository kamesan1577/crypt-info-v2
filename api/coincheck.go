package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CoincheckClient Coincheck API クライアント
type CoincheckClient struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

// BalanceResponse 残高レスポンス
type BalanceResponse struct {
	Success bool `json:"success"`
	Data    struct {
		JPY   string `json:"jpy"`
		BTC   string `json:"btc"`
		ETH   string `json:"eth"`
		ETC   string `json:"etc"`
		LSK   string `json:"lsk"`
		FCT   string `json:"fct"`
		XRP   string `json:"xrp"`
		XEM   string `json:"xem"`
		LTC   string `json:"ltc"`
		BCH   string `json:"bch"`
		MONA  string `json:"mona"`
		XLM   string `json:"xlm"`
		QTUM  string `json:"qtum"`
		DASH  string `json:"dash"`
		ZEC   string `json:"zec"`
		BAT   string `json:"bat"`
		IOST  string `json:"iost"`
		ENJ   string `json:"enj"`
		OMG   string `json:"omg"`
		PLT   string `json:"plt"`
		XTZ   string `json:"xtz"`
		ATOM  string `json:"atom"`
		MKR   string `json:"mkr"`
		LINK  string `json:"link"`
		COMP  string `json:"comp"`
		YFI   string `json:"yfi"`
		UNI   string `json:"uni"`
		AAVE  string `json:"aave"`
		SNX   string `json:"snx"`
		CRV   string `json:"crv"`
		MATIC string `json:"matic"`
		SOL   string `json:"sol"`
		AVAX  string `json:"avax"`
		DOT   string `json:"dot"`
		ADA   string `json:"ada"`
		SHIB  string `json:"shib"`
		DOGE  string `json:"doge"`
		TRX   string `json:"trx"`
		NEAR  string `json:"near"`
		FTM   string `json:"ftm"`
		ALGO  string `json:"algo"`
		MANA  string `json:"mana"`
		SAND  string `json:"sand"`
		AXS   string `json:"axs"`
		CHZ   string `json:"chz"`
		FLOW  string `json:"flow"`
		ICP   string `json:"icp"`
		VET   string `json:"vet"`
		FIL   string `json:"fil"`
		THETA string `json:"theta"`
		EOS   string `json:"eos"`
		KLAY  string `json:"klay"`
		HBAR  string `json:"hbar"`
	} `json:"data"`
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

// NewCoincheckClient 新しいCoincheckクライアントを作成
func NewCoincheckClient(apiKey, apiSecret string) *CoincheckClient {
	return &CoincheckClient{
		APIKey:    apiKey,
		APISecret: apiSecret,
		BaseURL:   "https://coincheck.com",
	}
}

// createSignature HMAC-SHA256署名を作成
func (c *CoincheckClient) createSignature(nonce, url, body string) string {
	message := nonce + url + body
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// makeRequest 認証付きリクエストを実行
func (c *CoincheckClient) makeRequest(method, endpoint, body string) (*http.Response, error) {
	url := c.BaseURL + endpoint
	nonce := strconv.FormatInt(time.Now().Unix(), 10)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
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

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// GetBalance 口座残高を取得
func (c *CoincheckClient) GetBalance() (*BalanceResponse, error) {
	resp, err := c.makeRequest("GET", "/api/accounts/balance", "")
	if err != nil {
		return nil, fmt.Errorf("残高取得リクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("残高取得失敗: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	var balance BalanceResponse
	if err := json.Unmarshal(body, &balance); err != nil {
		return nil, fmt.Errorf("JSON解析失敗: %v", err)
	}

	if !balance.Success {
		return nil, fmt.Errorf("API呼び出し失敗")
	}

	return &balance, nil
}

// GetTicker 指定通貨の価格情報を取得
func (c *CoincheckClient) GetTicker(pair string) (*TickerResponse, error) {
	url := fmt.Sprintf("%s/api/ticker?pair=%s", c.BaseURL, pair)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("価格取得リクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("価格取得失敗: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み込み失敗: %v", err)
	}

	var ticker TickerResponse
	if err := json.Unmarshal(body, &ticker); err != nil {
		return nil, fmt.Errorf("JSON解析失敗: %v", err)
	}

	return &ticker, nil
}

// FormatBalanceMessage 残高情報をLINEメッセージ用にフォーマット
func (c *CoincheckClient) FormatBalanceMessage(balance *BalanceResponse) string {
	message := "💰 Coincheck 口座残高\n\n"

	// JPY残高
	if balance.Data.JPY != "0" {
		message += fmt.Sprintf("💴 JPY: %s円\n", balance.Data.JPY)
	}

	// 暗号通貨残高（0以外のみ表示）
	cryptoAssets := map[string]string{
		"BTC":   balance.Data.BTC,
		"ETH":   balance.Data.ETH,
		"ETC":   balance.Data.ETC,
		"LSK":   balance.Data.LSK,
		"FCT":   balance.Data.FCT,
		"XRP":   balance.Data.XRP,
		"XEM":   balance.Data.XEM,
		"LTC":   balance.Data.LTC,
		"BCH":   balance.Data.BCH,
		"MONA":  balance.Data.MONA,
		"XLM":   balance.Data.XLM,
		"QTUM":  balance.Data.QTUM,
		"DASH":  balance.Data.DASH,
		"ZEC":   balance.Data.ZEC,
		"BAT":   balance.Data.BAT,
		"IOST":  balance.Data.IOST,
		"ENJ":   balance.Data.ENJ,
		"OMG":   balance.Data.OMG,
		"PLT":   balance.Data.PLT,
		"XTZ":   balance.Data.XTZ,
		"ATOM":  balance.Data.ATOM,
		"MKR":   balance.Data.MKR,
		"LINK":  balance.Data.LINK,
		"COMP":  balance.Data.COMP,
		"YFI":   balance.Data.YFI,
		"UNI":   balance.Data.UNI,
		"AAVE":  balance.Data.AAVE,
		"SNX":   balance.Data.SNX,
		"CRV":   balance.Data.CRV,
		"MATIC": balance.Data.MATIC,
		"SOL":   balance.Data.SOL,
		"AVAX":  balance.Data.AVAX,
		"DOT":   balance.Data.DOT,
		"ADA":   balance.Data.ADA,
		"SHIB":  balance.Data.SHIB,
		"DOGE":  balance.Data.DOGE,
		"TRX":   balance.Data.TRX,
		"NEAR":  balance.Data.NEAR,
		"FTM":   balance.Data.FTM,
		"ALGO":  balance.Data.ALGO,
		"MANA":  balance.Data.MANA,
		"SAND":  balance.Data.SAND,
		"AXS":   balance.Data.AXS,
		"CHZ":   balance.Data.CHZ,
		"FLOW":  balance.Data.FLOW,
		"ICP":   balance.Data.ICP,
		"VET":   balance.Data.VET,
		"FIL":   balance.Data.FIL,
		"THETA": balance.Data.THETA,
		"EOS":   balance.Data.EOS,
		"KLAY":  balance.Data.KLAY,
		"HBAR":  balance.Data.HBAR,
	}

	for symbol, amount := range cryptoAssets {
		if amount != "0" && amount != "" {
			message += fmt.Sprintf("🪙 %s: %s\n", symbol, amount)
		}
	}

	message += "\n📅 " + time.Now().Format("2006年1月2日 15:04")

	return message
}
