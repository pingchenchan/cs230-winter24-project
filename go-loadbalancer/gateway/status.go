package gateway

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Context key
type ContextKey int

const (
	Attempts ContextKey = iota
	Retries
)

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retries).(int); ok {
		return retry
	}
	return 0
}

func isBackendAlive(rawUrl string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(rawUrl)
	if err != nil {
		// log.Println("Site unreachable, error: ", err)
		return false
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode > http.StatusOK {
		// log.Println("Error Status Code: ", resp.StatusCode)
		return false
	}

	return true
}

type MonitorReport struct {
	Dead []string `json:"dead"`
}

func reportDeadBackends(monitorUrl string, dead []string) error {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	report := MonitorReport{
		Dead: dead,
	}
	b, err := json.Marshal(report)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, monitorUrl, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	return nil
}

func HealthCheck(gtw *Gateway, duration time.Duration) {
	healthCheckWithCount(gtw, duration, -1)
}

func healthCheckWithCount(gtw *Gateway, duration time.Duration, maxCount int) {
	t := time.NewTicker(duration)
	count := 0
	for range t.C {
		log.Println("Starting health check...")
		gtw.healthCheck()
		log.Println("Health check completed")
		count++
		if maxCount >= 0 && count >= maxCount {
			return
		}
	}
}

func ResurrectServers(gtw *Gateway, duration time.Duration) {
	t := time.NewTicker(duration)
	for range t.C {
		log.Println("Starting server resurrection...")
		gtw.resurrectDisconnected()
		log.Println("Resurrection completed")
	}
}

func ReportDeadServers(gtw *Gateway, duration time.Duration) {
	t := time.NewTicker(duration)
	for range t.C {
		log.Println("Starting server resurrection...")
		gtw.report()
		log.Println("Resurrection completed")
	}
}
