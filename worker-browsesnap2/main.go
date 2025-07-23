package main

import (
	"bs/generate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	ua        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
	chUA      = `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`
	baseURL   = "https://browsesnap.com"
	email     = "fitriapipit080100@gmail.com"
	password  = "QWEASD123"
	sUsername = "fitria080100"
	sID       = "1513211342"
	LP        = 708000
)

type TaskResponse struct {
	Task struct {
		APIURL     string `json:"api_url"`
		WebsiteURL string `json:"website_url"`
		TaskID     int    `json:"task_id"`
	} `json:"task"`
	Detail string `json:"detail"`
}

type StatsResponse struct {
	TodayTasksCompleted int     `json:"today_tasks_completed"`
	TotalEarnings       float64 `json:"total_earnings"`
}

func makeRequest(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

func main() {
	headersGen, err := generate.GenerateAllHeaders()
	if err != nil {
		fmt.Println(err)
	}
	csrf := headersGen["X-CSRFToken"]
	af_ac_enc_dat := headersGen["af-ac-enc-dat"]
	sz := headersGen["sz-token"]
	x_sap_ri := headersGen["x-sap-ri"]
	x_sap_sec := headersGen["x-sap-sec"]
	af_ac_enc_sz_token := headersGen["af-ac-enc-sz-token"]
	d_non_ptcha := headersGen["d-nonptcha-sync"]
	date := headersGen["date"]
	x_request_id := headersGen["x-request-id"]
	rand_time := headersGen["response_time"]

	if err := godotenv.Load(); err != nil {
		fmt.Println("Gagal load .env:", err)
	}

	email := email
	pass := password
	sUsername := sUsername
	sID := sID
	isID, _ := strconv.Atoi(sID)
	resBody := os.Getenv("RESPONSE_BODY")
	authB64 := base64.StdEncoding.EncodeToString([]byte(email + ":" + pass))

	// ===== GET /tasks loop until found =====
	taskHeaders := map[string]string{
		"Authorization":      "Basic " + authB64,
		"User-Agent":         ua,
		"sec-ch-ua":          chUA,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "Windows",
		"Content-Type":       "application/json",
		"Referer":            "https://shopee.co.id/",
	}

	var task TaskResponse
	foundTask := false
	for !foundTask {
		taskResp, err := makeRequest("GET", baseURL+"/tasks", taskHeaders, nil)
		if err != nil {
			fmt.Println("❌ Gagal GET /tasks:", err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer taskResp.Body.Close()

		json.NewDecoder(taskResp.Body).Decode(&task)

		if task.Task.APIURL != "" && task.Task.WebsiteURL != "" {
			foundTask = true
			fmt.Println("✅ sukses tsk:", task.Task.TaskID)
		} else if task.Detail == "User has reached the task limit" {
			fmt.Println("🤷‍♂️  Sudah limit, task kosong")
			return
		} else {
			fmt.Println("⚠️  Tidak ada task valid, retry...")
			time.Sleep(2 * time.Second)
		}
	}

	if foundTask {
		time.Sleep(1 * time.Second)

		// ===== POST /collect-user-behaviour =====
		behaviourHeaders := map[string]string{
			"Authorization":      "Basic " + authB64,
			"Accept":             "*/*",
			"Accept-Language":    "en-US,en;q=0.9",
			"User-Agent":         ua,
			"sec-ch-ua":          chUA,
			"sec-ch-ua-mobile":   "?0",
			"sec-ch-ua-platform": "Windows",
			"Sec-Fetch-Dest":     "empty",
			"Sec-Fetch-Mode":     "cors",
			"Sec-Fetch-Site":     "cross-site",
			"Priority":           "u=1, i",
			"Origin":             "https://shopee.co.id",
			"Referer":            "https://shopee.co.id/",
			"Content-Type":       "application/json",
		}
		makeRequest("POST", baseURL+"/collect-user-behaviour", behaviourHeaders, nil)

		time.Sleep(1 * time.Second)

		// ===== POST /collect =====
		apiURL := task.Task.APIURL
		sourceURL := task.Task.WebsiteURL

		payload := fmt.Sprintf(`{"request":{"url":"%s","method":"GET","request_headers":{"Accept":"application/json","Content-Type":"application/json","X-Shopee-Language":"id","X-Requested-With":"XMLHttpRequest","X-CSRFToken":"%s","X-API-SOURCE":"pc","af-ac-enc-dat":"%s","sz-token":"%s","x-sz-sdk-version":"1.12.20","x-sap-ri":"%s","x-sap-sec":"%s","af-ac-enc-sz-token":"%s","d-nonptcha-sync":"%s"},"request_body":null,"response_headers":{"alt-svc":"","content-encoding":"gzip","content-type":"application/json","date":"%s","server":"SGW","vary":"Accept-Encoding","x-request-id":"%s"},"response_body":"%s","response_status":200,"response_time":"%s","response_type":"basic"},"user_data":{"shopee_username":"%s","shopee_user_id":%d},"source_url":"%s"}`,
			apiURL, csrf, af_ac_enc_dat, sz, x_sap_ri, x_sap_sec, af_ac_enc_sz_token, d_non_ptcha, date, x_request_id, resBody, rand_time, sUsername, isID, sourceURL)

		collectHeaders := behaviourHeaders // same headers
		respClt, err := makeRequest("POST", baseURL+"/collect", collectHeaders, strings.NewReader(payload))
		if err != nil {
			fmt.Println("❌ Gagal POST /collect:", err)
		} else {
			if respClt.StatusCode != 201 {
				fmt.Println("❌ data ga masuk clt:", respClt.Status)
			} else {
				fmt.Println("✅ sukses clt:", respClt.Status)
			}
		}
	}

	// ===== GET /stats =====
	statsURL := baseURL + "/stats?timezone=Asia%2FJakarta"
	statsHeaders := map[string]string{
		"Authorization":      "Basic " + authB64,
		"Accept":             "*/*",
		"Accept-Language":    "en,en-US;q=0.9,id;q=0.8",
		"User-Agent":         ua,
		"sec-ch-ua":          chUA,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "Windows",
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "cross-site",
		"Priority":           "u=1, i",
		"Origin":             "https://shopee.co.id",
		"Referer":            "https://shopee.co.id/",
		"Content-Type":       "application/json",
	}
	respStats, err := makeRequest("GET", statsURL, statsHeaders, nil)
	if err != nil {
		fmt.Println("❌ Gagal GET /stats:", err)
		return
	}
	defer respStats.Body.Close()

	var stats StatsResponse
	if err := json.NewDecoder(respStats.Body).Decode(&stats); err != nil {
		fmt.Println("❌ Gagal decode /stats:", err)
		return
	}

	fmt.Printf("✅ today task completed = %d\n", stats.TodayTasksCompleted)
	latestPayment := LP
	fmt.Printf("✅ next payment %.0f\n", stats.TotalEarnings-float64(latestPayment))
}
