package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const target = "https://192.168.49.2:8443"

// --- metrics ---

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_requests_total",
			Help: "Total number of requests forwarded by the proxy",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_request_duration_seconds",
			Help:    "Latency of requests forwarded to the API server",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	watchEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_watch_events_total",
			Help: "Total number of watch events received from the API server",
		},
		[]string{"event_type", "resource"}, // ADDED/MODIFIED/DELETED/BOOKMARK
	)

	streamBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_stream_bytes_total",
			Help: "Total bytes forwarded through watch streams",
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, watchEventsTotal, streamBytesTotal)
}

// --- watch event parsing ---

type watchEvent struct {
	Type string `json:"type"`
}

func extractEventType(line []byte) string {
	var e watchEvent
	if err := json.Unmarshal(line, &e); err != nil {
		return "UNKNOWN"
	}
	return e.Type
}

// resourceFromPath extracts the resource name from a path like /apis/demo.test.com/v1/toys
func resourceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

func main() {
	// to enable tls
	certFile := os.ExpandEnv("$HOME/.minikube/profiles/minikube/client.crt")
	keyFile := os.ExpandEnv("$HOME/.minikube/profiles/minikube/client.key")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("[PROXY] failed to load client cert: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
			},
		},
	}

	// expose /metrics for Prometheus
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := target + r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}

		// recreate the request and forward it to the api-server
		req, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Header = r.Header.Clone()
		req.Header.Set("Accept-Encoding", "identity")

		log.Printf("[PROXY] %s %s", r.Method, r.URL.Path)

		start := time.Now()
		// forward the request
		resp, err := client.Do(req)
		elapsed := time.Since(start).Seconds()

		if err != nil {
			log.Printf("[PROXY] ERROR forwarding: %v", err)
			requestsTotal.WithLabelValues(r.Method, r.URL.Path, "502").Inc()
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		statusStr := http.StatusText(resp.StatusCode)
		log.Printf("[PROXY] %s %s -> %d", r.Method, url, resp.StatusCode)

		requestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()   // increment counter
		requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(elapsed) // register the elapsed time

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)

		// watch interaction are a never ending stream of bytes, we need to flush them as they come
		flusher, canFlush := w.(http.Flusher)
		resource := resourceFromPath(r.URL.Path)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()

			// only parse watch streams (they carry JSON event objects)
			if r.URL.Query().Get("watch") == "true" {
				eventType := extractEventType(line)
				log.Printf("[PROXY] event %s %s [%s]: %s", r.Method, r.URL.Path, eventType, line)
				watchEventsTotal.WithLabelValues(eventType, resource).Inc() // increment number of watchevents
			} else {
				log.Printf("[PROXY] event %s %s: %s", r.Method, r.URL.Path, line)
			}

			streamBytesTotal.WithLabelValues(r.URL.Path).Add(float64(len(line) + 1)) // increment bytes sent

			w.Write(line)
			w.Write([]byte("\n"))
			if canFlush {
				flusher.Flush()
			}
		}
	})

	log.Println("[PROXY] started on :8080, metrics on :8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
