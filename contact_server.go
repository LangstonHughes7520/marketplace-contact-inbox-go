package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	client, err := NewInfraiClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	inbox := os.Getenv("TEAM_INBOX")
	if inbox == "" {
		log.Fatal("TEAM_INBOX is required")
	}
	router := ContactRouter{verifier: client, sender: client, inbox: inbox}
	http.HandleFunc("/contact", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer req.Body.Close()
		var form ContactRequest
		if err := json.NewDecoder(req.Body).Decode(&form); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := router.Route(req.Context(), form, clientIP(req))
		if err != nil {
			http.Error(w, "contact could not be accepted", http.StatusBadGateway)
			return
		}
		if result.Rejected {
			http.Error(w, "captcha rejected", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message_id": result.MessageID})
	})
	log.Println("contact server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func clientIP(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(req.RemoteAddr)
}
