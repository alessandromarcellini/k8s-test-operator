package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net/http"
	"os"
)

const target = "https://192.168.49.2:8443"

func main() {
	// Minikube client cert, the proxy authenticates on behalf of the operator
	certFile := os.ExpandEnv("$HOME/.minikube/profiles/minikube/client.crt")
	keyFile := os.ExpandEnv("$HOME/.minikube/profiles/minikube/client.key")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("[PROXY] failed to load client cert: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // skip server cert verification (dev only)
				Certificates:       []tls.Certificate{cert},
			},
		},
	}
	//NOT WORKING BECAUSE: Kubernetes watch connections are infinite streams, they do not "reach completion", io.Copy was blocking all of it
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	url := target + r.URL.Path
	// 	if r.URL.RawQuery != "" {
	// 		url += "?" + r.URL.RawQuery
	// 	}

	// 	req, err := http.NewRequest(r.Method, url, r.Body)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusBadRequest)
	// 		return
	// 	}
	// 	req.Header = r.Header.Clone()

	// 	log.Printf("[PROXY] %s %s", r.Method, r.URL.Path)

	// 	resp, err := client.Do(req)
	// 	if err != nil {
	// 		log.Printf("[PROXY] ERROR forwarding: %v", err)
	// 		http.Error(w, err.Error(), http.StatusBadGateway)
	// 		return
	// 	}
	// 	defer resp.Body.Close()

	// 	log.Printf("[PROXY] %s %s → %d", r.Method, r.URL.Path, resp.StatusCode)

	// 	for k, v := range resp.Header {
	// 		w.Header()[k] = v
	// 	}
	// 	w.WriteHeader(resp.StatusCode)
	// 	io.Copy(w, resp.Body)
	// })

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := target + r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}

		req, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Header = r.Header.Clone()

		// Disable compression, it interferes with streaming
		req.Header.Set("Accept-Encoding", "identity")

		log.Printf("[PROXY] %s %s", r.Method, r.URL.Path)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[PROXY] ERROR forwarding: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		log.Printf("[PROXY] %s %s -> %d", r.Method, url, resp.StatusCode)

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)

		// flush incrementally so watch streams are forwarded in real time
		flusher, canFlush := w.(http.Flusher)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			log.Printf("[PROXY] event %s %s: %s", r.Method, r.URL.Path, line)
			w.Write(line)
			w.Write([]byte("\n"))
			if canFlush {
				flusher.Flush()
			}
		}
	})

	log.Println("[PROXY] started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
