package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var workers = []string{
	"worker-browsesnap",
	"worker-browsesnap2",
	"worker-browsesnap3",
	"worker-browsesnap4",
	"worker-browsesnap5",
	"worker-browsesnap6",
}

var logMutex = &sync.Mutex{}

func main() {
	// Buat direktori logs di root project
	logsDir := filepath.Join("..", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("❌ Gagal membuat direktori logs: %v", err)
	}

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/logs/", handleSingleLog)
	http.HandleFunc("/start", handleStartWorkers)
	http.HandleFunc("/update-env", handleUpdateEnv)
	http.HandleFunc("/get-env", handleGetEnvShared)

	fmt.Println("🌐 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startWorkers(loopCount int) {
	for _, worker := range workers {
		go runWorker(worker, loopCount)
	}
}

func runWorker(worker string, loopCount int) {
	// Dapatkan absolute path ke direktori bs-work-space
	baseDir := "/home/bens/projects/myrepo/bs-work-space"

	// Path untuk log file
	logFile := filepath.Join(baseDir, "logs", worker+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("❌ Gagal buat log untuk %s: %v", worker, err)
		return
	}
	defer f.Close()

	// Path ke direktori worker
	workerDir := filepath.Join(baseDir, worker)

	log.Printf("🚀 Menjalankan %s dari %s", worker, workerDir)

	for i := 1; i <= loopCount; i++ {
		writeLog(f, worker, fmt.Sprintf("🔁 Loop ke-%d", i))

		cmd := exec.Command("go", "run", "main.go")
		cmd.Dir = workerDir // Gunakan absolute path
		cmd.Stdout = f
		cmd.Stderr = f

		if err := cmd.Run(); err != nil {
			writeLog(f, worker, fmt.Sprintf("❌ Error di loop ke-%d: %v", i, err))
			continue
		}

		writeLog(f, worker, fmt.Sprintf("✅ Loop ke-%d selesai", i))
	}
	writeLog(f, worker, fmt.Sprintf("🎉 Worker %s selesai", worker))
}

func writeLog(f *os.File, worker, message string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	line := fmt.Sprintf("%s\n", message)
	if _, err := f.WriteString(line); err != nil {
		log.Printf("❌ Gagal menulis log untuk %s: %v", worker, err)
	}
	fmt.Print(line) // Tampilkan juga di console
}

// Handler untuk mulai workers
func handleStartWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Loop int `json:"loop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Loop <= 0 {
		http.Error(w, "Invalid loop count", http.StatusBadRequest)
		return
	}

	go startWorkers(body.Loop)
	fmt.Fprintf(w, "Started workers with %d loops each", body.Loop)
}

// Handler untuk melihat log
func handleSingleLog(w http.ResponseWriter, r *http.Request) {
	worker := strings.TrimPrefix(r.URL.Path, "/logs/")
	data, err := ioutil.ReadFile("logs/" + worker + ".log")
	if err != nil {
		http.Error(w, "Log not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

// Handler untuk update environment
func handleUpdateEnv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ResponseBody string `json:"response_body"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || strings.TrimSpace(body.ResponseBody) == "" {
		http.Error(w, "invalid JSON or empty response_body", http.StatusBadRequest)
		return
	}

	content := fmt.Sprintf("RESPONSE_BODY=%s\n", body.ResponseBody)
	err = os.WriteFile(".env.shared", []byte(content), 0644)
	if err != nil {
		http.Error(w, "failed to write .env.shared", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("bash", "update-env.sh")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to run update script: %s", output), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("✅ RESPONSE_BODY updated and env files refreshed.\n"))
}

// Handler untuk get environment
func handleGetEnvShared(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(".env.shared")
	if err != nil {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

// HTML UI sederhana
// func serveHTML(w http.ResponseWriter, r *http.Request) {
func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Worker Monitor</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			margin: 0;
			padding: 20px;
			background-color: #f1f5f9;
		}
		h1 {
			font-size: 24px;
			margin-bottom: 20px;
			text-align: center;
		}
		.container {
			max-width: 1000px;
			margin: auto;
		}
		.controls {
			display: flex;
			flex-wrap: wrap;
			gap: 10px;
			margin-bottom: 20px;
			align-items: center;
			justify-content: space-between;
		}
		input, textarea, button {
			padding: 8px;
			border: 1px solid #ccc;
			border-radius: 6px;
			font-size: 14px;
		}
		button {
			background-color: #2563eb;
			color: white;
			cursor: pointer;
		}
		button:hover {
			background-color: #1d4ed8;
		}
		.worker {
			background-color: white;
			border-radius: 8px;
			box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
			margin-bottom: 16px;
			padding: 16px;
		}
		.worker h3 {
			margin: 0 0 8px 0;
			font-size: 18px;
		}
		.log {
			background-color: #f3f4f6;
			height: 200px;
			overflow-y: auto;
			padding: 10px;
			border-radius: 6px;
			font-size: 13px;
			white-space: pre-wrap;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Worker Monitor</h1>

		<div class="controls">
			<div>
				<input type="number" id="loopCount" min="1" value="1" placeholder="Loop count">
				<button onclick="startWorkers()">Start Workers</button>
			</div>
			<div>
				<textarea id="responseBody" placeholder="RESPONSE_BODY" rows="2" cols="40"></textarea><br>
				<button onclick="updateEnv()">Update Environment</button>
			</div>
			
		</div>

		<div id="workers"></div>
	</div>

	<script>
		const workers = %s;

		function renderWorkers() {
			let html = "";
			workers.forEach(worker => {
				html += "<div class='worker'>" +
							"<h3>" + worker + "</h3>" +
							"<pre class='log' id='log-" + worker + "'>Loading...</pre>" +
						"</div>";
			});
			document.getElementById("workers").innerHTML = html;
		}
		
		function clearLogs() {
			if (!confirm("Yakin ingin menghapus semua log?")) return;

			fetch("/clear-logs", {
				method: "POST"
			})
			.then(r => r.text())
			.then(msg => {
				alert(msg);
				workers.forEach(viewLog); // Refresh log tampilan
			});
		}


		function startWorkers() {
			const loopCount = document.getElementById("loopCount").value;
			fetch("/start", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ loop: parseInt(loopCount) })
			}).then(() => alert("Workers started"));
		}

		function updateEnv() {
			const value = document.getElementById("responseBody").value;
			fetch("/update-env", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ response_body: value })
			}).then(() => alert("Environment updated"));
		}

		function viewLog(worker) {
			fetch("/logs/" + worker)
				.then(r => r.text())
				.then(log => {
					const logEl = document.getElementById("log-" + worker);
					if (logEl) {
						logEl.textContent = log;
						logEl.scrollTop = logEl.scrollHeight;
					}
				});
		}

		// Auto-load log setiap 3 detik
		setInterval(() => {
			workers.forEach(viewLog);
		}, 3000);

		// Load environment saat awal
		fetch("/get-env")
			.then(r => r.text())
			.then(text => {
				const match = text.match(/RESPONSE_BODY=(.*)/);
				if (match) document.getElementById("responseBody").value = match[1].trim();
			});

		renderWorkers();
	</script>
</body>
</html>`

	workersJson, _ := json.Marshal(workers)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html, string(workersJson))
}
