package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

var workers = []string{
	"worker-browsesnap",
	"worker-browsesnap2",
	"worker-browsesnap3",
	"worker-browsesnap4",
	"worker-browsesnap5",
	"worker-browsesnap6",
}

var (
	stopChan    = make(chan struct{})
	mu          sync.Mutex
	activeProcs []*os.Process
)

func main() {
	logsDir := filepath.Join("..", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("❌ Gagal membuat direktori logs: %v", err)
	}

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/logs/", handleSingleLog)
	http.HandleFunc("/start", handleStartWorkers)
	http.HandleFunc("/stop", handleStopWorkers)
	http.HandleFunc("/update-env", handleUpdateEnv)
	http.HandleFunc("/get-env", handleGetEnvShared)
	http.HandleFunc("/clear-logs", handleClearLogs)

	fmt.Println("🌐 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startWorkers(loopCount int) {
	mu.Lock()
	stopChan = make(chan struct{})
	activeProcs = []*os.Process{}
	mu.Unlock()
	for _, worker := range workers {
		go runWorker(worker, loopCount)
	}
}

func writeLog(f *os.File, worker, message string) {
	logLine := fmt.Sprintf("[%s][%s] %s\n", worker, time.Now().Format("2006-01-02 15:04:05"), message)
	_, err := f.WriteString(logLine)
	if err != nil {
		fmt.Printf("❌ Gagal menulis log ke file: %v\n", err)
	}
}

func runWorker(worker string, loopCount int) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	for {
		if filepath.Base(cwd) == "bs-ws" || cwd == "/" {
			break
		}
		cwd = filepath.Dir(cwd)
	}

	if filepath.Base(cwd) != "bs-ws" {
		log.Fatal("🛑 Tidak menemukan folder 'bs-ws' di path manapun.")
	}

	baseDir := cwd
	logFile := filepath.Join(baseDir, "logs", worker+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("❌ Gagal buat log untuk %s: %v\n", worker, err)
		return
	}
	defer f.Close()

	log.SetOutput(f)
	workerDir := filepath.Join(baseDir, worker)

	for i := 1; i <= loopCount; i++ {
		select {
		case <-stopChan:
			writeLog(f, worker, "⛔ Worker dihentikan secara paksa")
			return
		default:
		}

		writeLog(f, worker, fmt.Sprintf("🔁 Loop ke-%d", i))
		cmd := exec.Command("go", "run", "main.go")
		cmd.Dir = workerDir
		cmd.Stdout = f
		cmd.Stderr = f

		// Pastikan kill group, bukan hanya parent
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			writeLog(f, worker, fmt.Sprintf("❌ Tidak bisa start: %v", err))
			continue
		}

		mu.Lock()
		activeProcs = append(activeProcs, cmd.Process)
		mu.Unlock()

		err = cmd.Wait()
		if err != nil {
			writeLog(f, worker, fmt.Sprintf("❌ Error di loop ke-%d: %v", i, err))
			continue
		}
	}

	writeLog(f, worker, fmt.Sprintf("🎉 Worker sukses %d selesai ", loopCount))
}

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

func handleStopWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	close(stopChan)
	for _, p := range activeProcs {
		if p != nil {
			syscall.Kill(-p.Pid, syscall.SIGKILL) // negatif = kill seluruh group
		}
	}
	activeProcs = []*os.Process{}
	mu.Unlock()
	fmt.Fprint(w, "⛔ Semua workers & proses dihentikan paksa")
}

func getLogPath(worker string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get runtime caller")
	}
	baseDir := filepath.Dir(filename)
	projectRoot := filepath.Join(baseDir, "..")
	logsDir := filepath.Join(projectRoot, "logs")
	return filepath.Join(logsDir, worker+".log"), nil
}

func handleSingleLog(w http.ResponseWriter, r *http.Request) {
	worker := strings.TrimPrefix(r.URL.Path, "/logs/")
	logPath, err := getLogPath(worker)
	if err != nil {
		http.Error(w, "Failed to resolve log path", http.StatusInternalServerError)
		return
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		// http.Error(w, "Log not found: "+err.Error(), http.StatusNotFound)
		http.Error(w, "Log not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

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
func handleClearLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmd := exec.Command("bash", "clear-logs.sh")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to clear logs: %s", output), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(string(output)))
}

func handleGetEnvShared(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(".env.shared")
	if err != nil {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Worker Monitor</title>
	<style>
		body {font-family: Arial, sans-serif;margin: 0;padding: 20px;background-color: #f1f5f9;}
		h1 {font-size: 24px;margin-bottom: 20px;text-align: center;}
		.container {max-width: 1200px;margin: auto;}
		.controls {display: flex;flex-wrap: wrap;gap: 10px;margin-bottom: 20px;align-items: center;justify-content: space-between;}
		input, textarea, button {padding: 8px;border: 1px solid #ccc;border-radius: 6px;font-size: 14px;}
		button {background-color: #2563eb;color: white;cursor: pointer;}
		button:hover {background-color: #1d4ed8;}

		/* Grid untuk workers */
		#workers {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
			gap: 16px;
		}

		.worker {background-color: white;border-radius: 8px;box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);padding: 16px;}
		.worker h3 {margin: 0 0 8px 0;font-size: 18px;}
		.log {background-color: #f3f4f6;height: 200px;overflow-y: auto;padding: 10px;border-radius: 6px;font-size: 13px;white-space: pre-wrap;}
	</style>
</head>
<body>
	<div class="container">
		<h1>Worker Monitor</h1>
		<div class="controls">
			<div>
				<input type="number" id="loopCount" min="1" value="1" placeholder="Loop count">
				<button onclick="startWorkers()">Start Workers</button>
				<button onclick="stopWorkers()">Stop All Workers</button>
				<button onclick="clearLogs()">Clear Logs</button>
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

		function startWorkers() {
			const loopCount = document.getElementById("loopCount").value;
			fetch("/start", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ loop: parseInt(loopCount) })
			}).then(() => alert("Workers started"));
		}

		function stopWorkers() {
			fetch("/stop", { method: "POST" }).then(() => alert("Workers stopped"));
		}

		function clearLogs() {
			if (!confirm("Yakin hapus semua log?")) return;
			fetch("/clear-logs", { method: "POST" })
				.then(r => r.text())
				.then(msg => alert(msg));
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

		setInterval(() => {
			workers.forEach(viewLog);
		}, 3000);

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
