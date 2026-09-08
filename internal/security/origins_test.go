package security

import (
	"strings"
	"testing"
)

func TestDefaultDevOrigins(t *testing.T) {
	got := DefaultDevOrigins()
	if len(got) != 2 {
		t.Fatalf("expected 2 defaults, got %d", len(got))
	}
	if got[0] != "http://localhost:5173" || got[1] != "http://127.0.0.1:5173" {
		t.Fatalf("unexpected defaults: %#v", got)
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{
			name: "valid pair",
			raw:  "https://learn.example.com,http://localhost:5173",
			want: []string{"https://learn.example.com", "http://localhost:5173"},
		},
		{
			name: "normalize host case and trim",
			raw:  " https://Learn.Example.COM ",
			want: []string{"https://learn.example.com"},
		},
		{
			name:    "wildcard rejected",
			raw:     "https://*.example.com",
			wantErr: "wildcard",
		},
		{
			name:    "star alone rejected",
			raw:     "*",
			wantErr: "wildcard",
		},
		{
			name:    "blank entry",
			raw:     "https://a.com,,https://b.com",
			wantErr: "blank",
		},
		{
			name:    "path rejected",
			raw:     "https://example.com/app",
			wantErr: "path",
		},
		{
			name:    "query rejected",
			raw:     "https://example.com?x=1",
			wantErr: "query",
		},
		{
			name:    "ftp rejected",
			raw:     "ftp://example.com",
			wantErr: "scheme",
		},
		{
			name:    "empty",
			raw:     "   ",
			wantErr: "empty",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseAllowedOrigins(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantErr)) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got %#v want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestResolveAllowedOrigins(t *testing.T) {
	t.Parallel()
	t.Run("dev defaults", func(t *testing.T) {
		got, err := ResolveAllowedOrigins("", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("expected defaults, got %#v", got)
		}
	})
	t.Run("production missing fatal", func(t *testing.T) {
		_, err := ResolveAllowedOrigins("production", "")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("production wildcard fatal", func(t *testing.T) {
		_, err := ResolveAllowedOrigins("production", "*")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("dev explicit replaces defaults", func(t *testing.T) {
		got, err := ResolveAllowedOrigins("development", "http://localhost:3000")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != "http://localhost:3000" {
			t.Fatalf("got %#v", got)
		}
	})
}

func TestOriginAllowed(t *testing.T) {
	t.Parallel()
	origins := []string{"http://localhost:5173", "https://learn.example.com"}
	if !OriginAllowed(origins, "http://localhost:5173") {
		t.Fatal("expected allow")
	}
	if !OriginAllowed(origins, "HTTPS://Learn.Example.COM") {
		t.Fatal("expected case-normalized allow")
	}
	if OriginAllowed(origins, "https://evil.example.com") {
		t.Fatal("expected deny")
	}
	if OriginAllowed(origins, "") {
		t.Fatal("expected deny empty")
	}
}
