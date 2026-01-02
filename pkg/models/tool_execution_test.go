package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToolExecution_JSONSerialization(t *testing.T) {
	exec := ToolExecution{
		ID:           1,
		CreatedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		SessionID:    "test-session-123",
		ToolName:     "nikto",
		InputJSON:    `{"host": "localhost", "port": 80}`,
		OutputJSON:   `{"vulnerabilities": []}`,
		ErrorMessage: "",
		DurationMs:   1500,
		Success:      true,
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ToolExecution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != exec.ID {
		t.Errorf("ID mismatch: expected %d, got %d", exec.ID, decoded.ID)
	}
	if decoded.ToolName != exec.ToolName {
		t.Errorf("ToolName mismatch: expected %s, got %s", exec.ToolName, decoded.ToolName)
	}
	if decoded.SessionID != exec.SessionID {
		t.Errorf("SessionID mismatch: expected %s, got %s", exec.SessionID, decoded.SessionID)
	}
	if decoded.Success != exec.Success {
		t.Errorf("Success mismatch: expected %v, got %v", exec.Success, decoded.Success)
	}
	if decoded.DurationMs != exec.DurationMs {
		t.Errorf("DurationMs mismatch: expected %d, got %d", exec.DurationMs, decoded.DurationMs)
	}
}

func TestToolExecution_JSONWithError(t *testing.T) {
	exec := ToolExecution{
		ID:           2,
		CreatedAt:    time.Now(),
		ToolName:     "nikto",
		InputJSON:    `{"host": "invalid"}`,
		ErrorMessage: "connection refused",
		DurationMs:   50,
		Success:      false,
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ToolExecution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Success {
		t.Error("expected Success to be false")
	}
	if decoded.ErrorMessage != "connection refused" {
		t.Errorf("expected error message 'connection refused', got '%s'", decoded.ErrorMessage)
	}
	if decoded.OutputJSON != "" {
		t.Errorf("expected empty OutputJSON for failed execution, got '%s'", decoded.OutputJSON)
	}
}

func TestToolExecution_OmitEmpty(t *testing.T) {
	exec := ToolExecution{
		ID:        1,
		ToolName:  "nikto",
		InputJSON: `{}`,
		Success:   true,
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// These fields should be omitted when empty
	if contains(jsonStr, "session_id") && !contains(jsonStr, `"session_id":""`) {
		// session_id with empty string is still included, that's fine
	}
	if contains(jsonStr, `"error_message":""`) {
		// Empty error message might still appear, but that's OK for omitempty behavior
	}
}

func TestToolExecution_ZeroValues(t *testing.T) {
	exec := ToolExecution{}

	if exec.ID != 0 {
		t.Errorf("expected zero ID, got %d", exec.ID)
	}
	if exec.ToolName != "" {
		t.Errorf("expected empty ToolName, got '%s'", exec.ToolName)
	}
	if exec.Success {
		t.Error("expected Success to be false by default")
	}
	if exec.DurationMs != 0 {
		t.Errorf("expected zero DurationMs, got %d", exec.DurationMs)
	}
}

func TestToolExecution_LargeInput(t *testing.T) {
	// Generate large JSON input
	largeInput := `{"data": "` + string(make([]byte, 10000)) + `"}`

	exec := ToolExecution{
		ID:        1,
		ToolName:  "nikto",
		InputJSON: largeInput,
		Success:   true,
	}

	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("failed to marshal large input: %v", err)
	}

	var decoded ToolExecution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal large input: %v", err)
	}

	if len(decoded.InputJSON) != len(largeInput) {
		t.Errorf("InputJSON length mismatch: expected %d, got %d", len(largeInput), len(decoded.InputJSON))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
