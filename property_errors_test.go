package propertyerrors

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestClassifyGroupsPropertyFailures(t *testing.T) {
	tests := []struct {
		name, kind, category, operation, level string
		wantFingerprint                        string
	}{
		{"maintenance", "maintenance_request", "plumbing", "dispatch", "error", "property/maintenance_request/plumbing/dispatch"},
		{"document", "tenant_document", "lease", "index", "error", "property/tenant_document/lease/index"},
		{"inspection", "inspection_reminder", "fire_safety", "schedule", "warning", "property/inspection_reminder/fire_safety/schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture, err := Classify(Failure{EventID: "evt-17", PropertyID: "prop-8", Kind: tt.kind, Category: tt.category, Operation: tt.operation, Description: "backend operation failed"})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(capture.Fingerprint, "/"); got != tt.wantFingerprint || capture.Level != tt.level {
				t.Fatalf("fingerprint = %q, level = %q", got, capture.Level)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestCaptureDecodesEnvelopeThenRetries(t *testing.T) {
	var calls int
	client, err := NewClient("test-key", &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPost || req.Header.Get("Idempotency-Key") != "property-error:evt-17" {
			t.Fatalf("request method or identity changed")
		}
		if calls == 1 {
			return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader(`{"ok":false,"data":null,"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`))}, nil
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":{"event_id":"evt_remote"},"error":null,"metadata":{}}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	client.sleep = func(context.Context, time.Duration) error { return nil }
	_, err = client.Capture(context.Background(), Capture{Exception: "maintenance request dispatch"}, "property-error:evt-17")
	if err != nil || calls != 2 {
		t.Fatalf("calls = %d, error = %v", calls, err)
	}
}
