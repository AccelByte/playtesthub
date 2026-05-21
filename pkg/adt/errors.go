package adt

import (
	"errors"
	"fmt"
)

// Sentinel error values let the service layer pattern-match by class
// rather than by HTTP status. Callers map them to gRPC codes per
// docs/errors.md.
//
// The 2026-05-21 live-API probe (against develop.blackbox.accelbyte.io,
// public swagger at /profiling/apidocs/api.json v1.35.0) showed ADT
// dispatches on a JSON `errorCode` field whose value is decoupled from
// the HTTP status; the classification path uses errorCode FIRST and
// only falls through to the HTTP status when no JSON body is present
// (plaintext mux-level 404/405 from unknown sub-paths).
//
// Live errorCode → sentinel mapping (Bug 4 fix):
//
//	errorCode 99    → ErrLinkageMissing      ("Namespace is not registered")
//	errorCode 401   → ErrUnauthenticated     (bearer missing/garbage)
//	errorCode 20001 → ErrPermissionDenied    (token valid, route perm absent)
//
// HTTP-status fallback (no JSON body OR errorCode not in the table
// above):
//
//	429    → ErrRateLimited
//	5xx    → ErrUnavailable (after retry exhaustion)
//	other 4xx → *ClientError
var (
	// ErrRateLimited maps to gRPC ResourceExhausted (mirrors
	// pkg/ags.ErrRateLimited). Returned without retry — ADT 429s are
	// admin-actionable.
	ErrRateLimited = errors.New("adt: upstream rate limited (HTTP 429)")

	// ErrUnavailable maps to gRPC Unavailable. Returned only after
	// the retry budget is exhausted on 5xx / timeout.
	ErrUnavailable = errors.New("adt: upstream unavailable after retry exhausted")

	// ErrLinkageMissing maps to gRPC FailedPrecondition "adt linkage
	// no longer exists or service token rejected, re-link required"
	// (docs/errors.md row authored in B1). The 2026-05-21 probe
	// confirmed ADT surfaces this as `HTTP 404 {"errorCode": 99,
	// "errorMessage": "unable to process request: Namespace is not
	// registered"}` — NOT 401 (which is now ErrUnauthenticated). See
	// pkg/adt/http.go classifyJSONError + STATUS_M5.md "Addendum
	// 2026-05-21 — live-API probe".
	ErrLinkageMissing = errors.New("adt: linkage missing or namespace not registered")

	// ErrUnauthenticated maps to gRPC Unauthenticated. ADT returns
	// `HTTP 401 {"errorCode": 401, "errorMessage": "unauthorized"}`
	// when the bearer token is missing, garbage, or expired and AGS
	// JWKS rejected the signature. Operator action is "rotate the AGS
	// IAM client credentials / restart the backend so it re-mints a
	// fresh service token", distinct from the linkage-missing recovery
	// path.
	ErrUnauthenticated = errors.New("adt: bearer token rejected (HTTP 401)")

	// ErrPermissionDenied maps to gRPC PermissionDenied. ADT returns
	// `HTTP 401 {"errorCode": 20001, "errorMessage": "unauthorized
	// access"}` when the bearer token IS valid but lacks the IAM
	// permission scope for the requested route (e.g. M6 telemetry
	// surfaces gated by NAMESPACE:{ns}:SESSION [READ]). Operator action
	// is "ask ADT-eng to grant the missing permission to the
	// playtesthub service client", distinct from linkage-missing and
	// from bearer-broken.
	ErrPermissionDenied = errors.New("adt: bearer token lacks required permission scope")
)

// ClientError is the typed wrapper for HTTP 4xx (other than 429 / the
// errorCode-classified cases) returned by ADT. Mirrors
// pkg/ags.ClientError so service callers can use the same errors.As
// pattern.
type ClientError struct {
	StatusCode int
	Op         string
	Message    string
}

func (e *ClientError) Error() string {
	if e.Op == "" {
		return fmt.Sprintf("adt: client error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("adt: %s: client error %d: %s", e.Op, e.StatusCode, e.Message)
}

// IsRateLimited returns true when err (or any error in its chain) is
// ErrRateLimited.
func IsRateLimited(err error) bool { return errors.Is(err, ErrRateLimited) }

// IsUnavailable returns true when err (or any error in its chain) is
// ErrUnavailable.
func IsUnavailable(err error) bool { return errors.Is(err, ErrUnavailable) }

// IsLinkageMissing returns true when err (or any error in its chain)
// is ErrLinkageMissing.
func IsLinkageMissing(err error) bool { return errors.Is(err, ErrLinkageMissing) }

// IsUnauthenticated returns true when err (or any error in its chain)
// is ErrUnauthenticated.
func IsUnauthenticated(err error) bool { return errors.Is(err, ErrUnauthenticated) }

// IsPermissionDenied returns true when err (or any error in its chain)
// is ErrPermissionDenied.
func IsPermissionDenied(err error) bool { return errors.Is(err, ErrPermissionDenied) }

// IsClientError returns true when err (or any error in its chain) is
// a *ClientError.
func IsClientError(err error) bool {
	var ce *ClientError
	return errors.As(err, &ce)
}
