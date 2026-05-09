package supabase

import (
	"testing"
)

func TestParseProjectList_MatchesRef(t *testing.T) {
	jsonData := `[
		{"ref":"aaabbbcccdddeeefffgg","name":"OtherProject","region":"eu-west-1","status":"ACTIVE_HEALTHY","database":{"host":"db.aaabbbcccdddeeefffgg.supabase.co"}},
		{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"ACTIVE_HEALTHY","database":{"host":"db.bwxvxzfzujkxnphedtom.supabase.co"}}
	]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Ref != "bwxvxzfzujkxnphedtom" {
		t.Errorf("Ref = %q, want %q", info.Ref, "bwxvxzfzujkxnphedtom")
	}
	if info.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", info.Region, "us-east-1")
	}
	if info.Status != "ACTIVE_HEALTHY" {
		t.Errorf("Status = %q, want %q", info.Status, "ACTIVE_HEALTHY")
	}
	if info.Name != "VibeCloudAI" {
		t.Errorf("Name = %q, want %q", info.Name, "VibeCloudAI")
	}
}

func TestParseProjectList_NotFound(t *testing.T) {
	jsonData := `[{"ref":"aaabbbcccdddeeefffgg","name":"Other","region":"eu-west-1","status":"ACTIVE_HEALTHY"}]`
	_, err := parseProjectList(jsonData, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}

func TestParseProjectList_PausedProject(t *testing.T) {
	jsonData := `[{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"INACTIVE"}]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != "INACTIVE" {
		t.Errorf("Status = %q, want %q", info.Status, "INACTIVE")
	}
	if !info.IsPaused() {
		t.Error("IsPaused() = false, want true")
	}
}

func TestParseProjectList_ComingUp(t *testing.T) {
	jsonData := `[{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"COMING_UP"}]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsStartingUp() {
		t.Error("IsStartingUp() = false, want true")
	}
}

func TestParseProjectList_InvalidJSON(t *testing.T) {
	_, err := parseProjectList("not json", "anything")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
