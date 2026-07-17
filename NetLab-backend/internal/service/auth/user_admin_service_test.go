package auth

import (
	"encoding/json"
	"testing"
	"time"

	"netlab-backend/internal/model"
)

func TestToAdminUserViewDoesNotExposeAdminFlag(t *testing.T) {
	view := toAdminUserView(&model.User{
		ID: 1, Username: "viewer", Email: "viewer@example.com",
		Role: model.RoleViewer, Status: model.StatusActive,
		CreatedAt: time.Unix(0, 0).UTC(),
	})
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal user view: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("decode user view: %v", err)
	}
	if _, exists := fields["isAdmin"]; exists {
		t.Fatalf("user view exposed isAdmin: %s", payload)
	}
}
