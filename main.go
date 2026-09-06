package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	client, err := NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var event PaymentEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "invalid payment", http.StatusBadRequest)
			return
		}
		notice := decideNotice(event)
		ctx := r.Context()
		if err := client.CreateChannel(ctx, "payments-"+event.AccountID); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := client.Publish(ctx, event, notice); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notice)
	})
	log.Println("fintech chat service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
