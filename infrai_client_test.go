package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfraiClientRequestFields(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests = append(requests, body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/email/send" {
			w.Write([]byte(`{"ok":true,"data":{"message_id":"msg_42"}}`))
			return
		}
		w.Write([]byte(`{"ok":true,"data":{}}`))
	}))
	defer server.Close()
	client := &InfraiClient{baseURL: server.URL, key: "test", widgetRecordID: "widget_1", http: server.Client()}
	if _, err := client.VerifyCaptcha(context.Background(), "token_1", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SendInboxEmail(context.Background(), "chenhua@changba.com", "subject", "plain text", "key"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0]["widget_record_id"] != "widget_1" || requests[1]["body"] != "plain text" {
		t.Fatalf("unexpected request fields: %+v", requests)
	}
	if _, invalid := requests[1]["text"]; invalid {
		t.Fatalf("email request contains unsupported text field: %+v", requests[1])
	}
}
