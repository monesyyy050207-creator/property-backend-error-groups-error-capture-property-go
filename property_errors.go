package propertyerrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Failure struct {
	EventID     string `json:"event_id"`
	PropertyID  string `json:"property_id"`
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	Operation   string `json:"operation"`
	Description string `json:"description"`
}

type Capture struct {
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Level       string         `json:"level"`
	Fingerprint []string       `json:"fingerprint"`
	Exception   string         `json:"exception"`
	Context     map[string]any `json:"context"`
}

func Classify(f Failure) (Capture, error) {
	if strings.TrimSpace(f.EventID) == "" || strings.TrimSpace(f.PropertyID) == "" {
		return Capture{}, errors.New("event_id and property_id are required")
	}
	if strings.TrimSpace(f.Category) == "" || strings.TrimSpace(f.Description) == "" {
		return Capture{}, errors.New("category and description are required")
	}

	var label, level string
	switch f.Kind {
	case "maintenance_request":
		label, level = "maintenance request", "error"
	case "tenant_document":
		label, level = "tenant document", "error"
	case "inspection_reminder":
		label, level = "inspection reminder", "warning"
	default:
		return Capture{}, fmt.Errorf("unknown property failure kind %q", f.Kind)
	}

	exception, err := json.Marshal(f)
	if err != nil {
		return Capture{}, fmt.Errorf("encode exception: %w", err)
	}
	return Capture{
		Title:       label + " failed",
		Message:     f.Description,
		Level:       level,
		Fingerprint: []string{"property", f.Kind, f.Category, f.Operation},
		Exception:   string(exception),
		Context: map[string]any{
			"event_id": f.EventID, "property_id": f.PropertyID,
			"kind": f.Kind, "category": f.Category, "operation": f.Operation,
		},
	}, nil
}
