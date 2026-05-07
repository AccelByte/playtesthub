package iam_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/anggorodewanto/playtesthub/pkg/iam"
)

// platformsResponse mirrors the AGS shape exactly enough for the lookup
// to walk it. Keeping it inline (rather than exported) prevents the
// production decode shape from drifting silently to match a test helper.
type platformsResponse struct {
	Platforms []platformEntry `json:"platforms"`
}

type platformEntry struct {
	PlatformID     string `json:"platformId"`
	PlatformName   string `json:"platformName"`
	PlatformUserID string `json:"platformUserId"`
}

// fakeAGS exposes the two endpoints AGSAdminPlatformLookup hits:
// `/iam/v3/oauth/token` (client-credentials) and the
// `/iam/v3/admin/namespaces/{ns}/users/{userId}/distinctPlatforms` GET.
// Counters let tests assert the token cache short-circuits the second
// hit. AGS rejects hyphenated UUIDs on user-path params, so the handler
// echoes whatever userId arrives; tests assert the stripped form.
func fakeAGS(t *testing.T) (*httptest.Server, *atomic.Int64, *atomic.Int64, *string) {
	t.Helper()
	tokenHits := &atomic.Int64{}
	platformHits := &atomic.Int64{}
	lastUserID := new(string)

	mux := http.NewServeMux()
	mux.HandleFunc("/iam/v3/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		tokenHits.Add(1)
		if user, _, ok := r.BasicAuth(); !ok || user != "client-id" {
			http.Error(w, "bad basic auth", http.StatusUnauthorized)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			http.Error(w, "wrong grant", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "admin-token",
			"expires_in":   3600,
		})
	})
	mux.HandleFunc("/iam/v3/admin/namespaces/test-ns/users/", func(w http.ResponseWriter, r *http.Request) {
		platformHits.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		// Path: /iam/v3/admin/namespaces/test-ns/users/{userId}/distinctPlatforms
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/iam/v3/admin/namespaces/test-ns/users/"), "/")
		if len(parts) < 2 || parts[1] != "distinctPlatforms" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		*lastUserID = parts[0]
		_ = json.NewEncoder(w).Encode(platformsResponse{
			Platforms: []platformEntry{
				{PlatformID: "justice", PlatformName: "justice", PlatformUserID: "f31ee36368224d86a43848159177f2c0"},
				{PlatformID: "discord", PlatformName: "discord", PlatformUserID: "1089351036650668143"},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, tokenHits, platformHits, lastUserID
}

func newLookup(srv *httptest.Server) *iam.AGSAdminPlatformLookup {
	return &iam.AGSAdminPlatformLookup{
		HTTPClient:   srv.Client(),
		BaseURL:      srv.URL,
		Namespace:    "test-ns",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}
}

func TestAGSAdminPlatformLookup_ReturnsDiscordSnowflake(t *testing.T) {
	srv, _, _, lastUserID := fakeAGS(t)
	got, err := newLookup(srv).GetDiscordID(context.Background(), "f31ee363-6822-4d86-a438-48159177f2c0")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "1089351036650668143" {
		t.Errorf("snowflake = %q, want 1089351036650668143", got)
	}
	if *lastUserID != "f31ee36368224d86a43848159177f2c0" {
		t.Errorf("AGS userId path = %q, want hyphens stripped", *lastUserID)
	}
}

func TestAGSAdminPlatformLookup_TokenCachedAcrossCalls(t *testing.T) {
	srv, tokenHits, platformHits, _ := fakeAGS(t)
	l := newLookup(srv)
	for i := range 3 {
		if _, err := l.GetDiscordID(context.Background(), "abc"); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := tokenHits.Load(); got != 1 {
		t.Errorf("token endpoint hits = %d, want 1 (cached)", got)
	}
	if got := platformHits.Load(); got != 3 {
		t.Errorf("platform endpoint hits = %d, want 3", got)
	}
}

func TestAGSAdminPlatformLookup_NoDiscordPlatform_ReturnsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/iam/v3/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 60})
	})
	mux.HandleFunc("/iam/v3/admin/namespaces/test-ns/users/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(platformsResponse{
			Platforms: []platformEntry{{PlatformID: "steam", PlatformName: "steam", PlatformUserID: "76561"}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	got, err := newLookup(srv).GetDiscordID(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "" {
		t.Errorf("snowflake = %q, want empty (user has no discord link)", got)
	}
}

func TestAGSAdminPlatformLookup_AGSError_Surfaces(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/iam/v3/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 60})
	})
	mux.HandleFunc("/iam/v3/admin/namespaces/test-ns/users/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"errorCode":12345,"errorMessage":"upstream"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if _, err := newLookup(srv).GetDiscordID(context.Background(), "abc"); err == nil {
		t.Fatal("expected error on 502")
	}
}

func TestAGSAdminPlatformLookup_TokenFetchFailure_Surfaces(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/iam/v3/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if _, err := newLookup(srv).GetDiscordID(context.Background(), "abc"); err == nil {
		t.Fatal("expected error when token endpoint rejects")
	}
}

func TestAGSAdminPlatformLookup_NilReceiver_SafeError(t *testing.T) {
	var l *iam.AGSAdminPlatformLookup
	if _, err := l.GetDiscordID(context.Background(), "abc"); err == nil {
		t.Fatal("expected error on nil receiver")
	}
}
