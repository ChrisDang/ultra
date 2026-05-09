package supabase

import (
	"testing"
)

func TestBuildDatabaseURL_Basic(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-1-us-east-1.pooler.supabase.com", 6543, "mypassword")
	want := "postgresql://postgres.abcdefghijklmnopqrst:mypassword@aws-1-us-east-1.pooler.supabase.com:6543/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURL_SessionMode(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-0-us-east-1.pooler.supabase.com", 5432, "pass123")
	want := "postgresql://postgres.abcdefghijklmnopqrst:pass123@aws-0-us-east-1.pooler.supabase.com:5432/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURL_SpecialCharsInPassword(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-0-us-east-1.pooler.supabase.com", 6543, "p@ss w0rd!#")
	want := "postgresql://postgres.abcdefghijklmnopqrst:p%40ss+w0rd%21%23@aws-0-us-east-1.pooler.supabase.com:6543/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}
