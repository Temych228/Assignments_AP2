package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type MockEmailSender struct {
	Latency     time.Duration
	FailureRate float64
}

func NewMockEmailSender(latency time.Duration, failureRate float64) *MockEmailSender {
	return &MockEmailSender{
		Latency:     latency,
		FailureRate: failureRate,
	}
}

func (m *MockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	select {
	case <-time.After(m.Latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	if rand.Float64() < m.FailureRate {
		return fmt.Errorf("simulated provider failure")
	}

	log.Printf("[Notification][SIMULATED] email to=%s subject=%q body=%q", to, subject, body)
	return nil
}
