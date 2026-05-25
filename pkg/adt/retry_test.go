package adt_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/anggorodewanto/playtesthub/pkg/adt"
)

type httpErr struct {
	status    int
	errorCode int
}

func (e *httpErr) Error() string     { return fmt.Sprintf("http %d errorCode=%d", e.status, e.errorCode) }
func (e *httpErr) HTTPStatus() int   { return e.status }
func (e *httpErr) ADTErrorCode() int { return e.errorCode }

func nopSleep(d time.Duration) {}

func policyForTest() adt.RetryPolicy {
	return adt.RetryPolicy{
		MaxAttempts:       4,
		PerAttemptTimeout: 100 * time.Millisecond,
		InitialBackoff:    0,
		MaxBackoff:        0,
		Sleep:             nopSleep,
	}
}

func TestRetry_SuccessOnFirstTry(t *testing.T) {
	calls := 0
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestRetry_5xxRetriesThenExhausts(t *testing.T) {
	calls := 0
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		calls++
		return &httpErr{status: 503}
	})
	if !errors.Is(err, adt.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if calls != 4 {
		t.Fatalf("calls = %d, want 4", calls)
	}
}

func TestRetry_429FailsImmediately(t *testing.T) {
	calls := 0
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		calls++
		return &httpErr{status: 429}
	})
	if !errors.Is(err, adt.ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on 429)", calls)
	}
}

func TestRetry_404ErrorCode99MapsToLinkageMissing(t *testing.T) {
	calls := 0
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		calls++
		return &httpErr{status: 404, errorCode: 99}
	})
	if !errors.Is(err, adt.ErrLinkageMissing) {
		t.Fatalf("err = %v, want ErrLinkageMissing", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on linkage-missing)", calls)
	}
}

func TestRetry_401ErrorCode401MapsToUnauthenticated(t *testing.T) {
	calls := 0
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		calls++
		return &httpErr{status: 401, errorCode: 401}
	})
	if !errors.Is(err, adt.ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on unauthenticated)", calls)
	}
}

func TestRetry_401ErrorCode20001MapsToPermissionDenied(t *testing.T) {
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		return &httpErr{status: 401, errorCode: 20001}
	})
	if !errors.Is(err, adt.ErrPermissionDenied) {
		t.Fatalf("err = %v, want ErrPermissionDenied", err)
	}
}

// ADT's downloadUrls endpoint returns HTTP 404 + errorCode=1003303402
// ("build not found") when the build was deleted from ADT. classify
// must map it to ErrBuildNotFound — distinct from the plaintext 404
// (route missing → *ClientError) and from errorCode=99 (linkage gone).
func TestRetry_404ErrorCode1003303402MapsToBuildNotFound(t *testing.T) {
	err := policyForTest().Run(context.Background(), "IssueDownloadURL", func(_ context.Context) error {
		return &httpErr{status: 404, errorCode: 1003303402}
	})
	if !errors.Is(err, adt.ErrBuildNotFound) {
		t.Fatalf("err = %v, want ErrBuildNotFound", err)
	}
}

// Bare 401 with no JSON errorCode (e.g. plaintext mux-level reply)
// falls through to *ClientError per the 2026-05-21 classification
// order — no longer collapses to ErrLinkageMissing.
func TestRetry_BarePlaintext401IsClientError(t *testing.T) {
	err := policyForTest().Run(context.Background(), "Op", func(_ context.Context) error {
		return &httpErr{status: 401}
	})
	if !adt.IsClientError(err) {
		t.Fatalf("err = %v, want *ClientError (no errorCode → status fallback)", err)
	}
}

func TestRetry_400WrapsClientError(t *testing.T) {
	err := policyForTest().Run(context.Background(), "MyOp", func(_ context.Context) error {
		return &httpErr{status: 400}
	})
	if !adt.IsClientError(err) {
		t.Fatalf("err = %v, want *ClientError", err)
	}
	var ce *adt.ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("errors.As failed: %v", err)
	}
	if ce.StatusCode != http.StatusBadRequest || ce.Op != "MyOp" {
		t.Fatalf("ClientError = %+v", ce)
	}
}

func TestRetry_ContextCanceledShortCircuits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := policyForTest().Run(ctx, "Op", func(_ context.Context) error {
		t.Fatal("op should not run when ctx already canceled")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDefaultRetryPolicy_Shape(t *testing.T) {
	p := adt.DefaultRetryPolicy()
	if p.MaxAttempts != 4 {
		t.Errorf("MaxAttempts = %d, want 4", p.MaxAttempts)
	}
	if p.PerAttemptTimeout != 30*time.Second {
		t.Errorf("PerAttemptTimeout = %v, want 30s", p.PerAttemptTimeout)
	}
}
