package main

import (
	"log"
	"net/http"
	"os"

	"crypt-info-v2/api"
	"crypt-info-v2/api/cron"
	"crypt-info-v2/api/health"
)

func main() {
	// 環境変数の読み込み確認
	requiredEnvVars := []string{
		"LINE_CHANNEL_SECRET",
		"LINE_CHANNEL_ACCESS_TOKEN",
		"COINCHECK_API_KEY",
		"COINCHECK_API_SECRET",
		"LINE_USER_ID",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("環境変数 %s が設定されていません", envVar)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("サーバーを起動中... ポート: %s", port)
	log.Printf("LINE Webhook URL: http://localhost:%s/api", port)
	log.Printf("定期実行テスト URL: http://localhost:%s/api/cron", port)
	log.Printf("ヘルスチェック URL: http://localhost:%s/api/health", port)

	// ローカル開発用の統合ハンドラー
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			api.POST(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/cron", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			cron.GET(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			health.GET(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
