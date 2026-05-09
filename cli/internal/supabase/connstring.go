package supabase

import (
	"fmt"
	"net/url"
)

// BuildDatabaseURL constructs a Supavisor pooler connection string.
// The password is URL-encoded to handle special characters.
func BuildDatabaseURL(ref, poolerHost string, port int, password string) string {
	return fmt.Sprintf("postgresql://postgres.%s:%s@%s:%d/postgres",
		ref,
		url.QueryEscape(password),
		poolerHost,
		port,
	)
}
