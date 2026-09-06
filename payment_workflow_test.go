package main

import "testing"

func TestDecideNotice(t *testing.T) {
	tests := []struct {
		name, risk, action string
		amount             int64
	}{
		{"small low risk", "low", "allow", 2500},
		{"high risk", "high", "review", 2500},
		{"large low risk", "low", "review", 100000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideNotice(PaymentEvent{ID: "evt-1", Amount: tt.amount, Risk: tt.risk})
			if got.Action != tt.action {
				t.Fatalf("action=%q, want %q", got.Action, tt.action)
			}
		})
	}
}
