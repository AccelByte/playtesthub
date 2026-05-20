package e2e_test

import (
	"testing"
)

// TestGoldenM5Flow is a placeholder for the M5.B ADT golden path. The
// suite harness boots `pkg/adt.MemClient` (per `internal/bootapp` —
// STATUS_M5.md B3) so the in-process surface is reachable, but the
// MemClient is not pre-seeded with a linkage + builds. Seeding requires
// an in-process hook (the bootapp does not expose a "seed ADT fixture"
// admin RPC, by design — the live flow goes through `StartADTLink` →
// browser redirect → `CompleteADTLink`).
//
// The live e2e wiring lands in a follow-up sub-phase (the live ADT
// adapter sub-phase per STATUS_M5.md B3) alongside the SDK-backed
// `pkg/adt` client, where the harness can be configured against ADT's
// staging endpoint or a fixture-pre-seeded MemClient.
//
// Until then the M5.B end-to-end coverage lives in:
//
//   - pkg/service/adt_create_playtest_test.go  (B5 happy path + rejection cases)
//   - pkg/service/adt_approve_test.go          (B6 ADT approve + DM + GetADTDownloadInfo)
//   - admin/src/federated-element.test.tsx     (B7 selector + linkages panel + modal)
//   - player/tests/Pending.test.ts             (B8 player ADT download view)
//   - cmd/pth dry-run `flow golden-m5`         (B9 — pinned via scripts/smoke/pth.sh)
//   - scripts/smoke/cloud.sh M5.B auth probes  (B10 — registered-handler gate)
func TestGoldenM5Flow(t *testing.T) {
	t.Skip("M5.B live e2e blocked on the SDK-backed ADT adapter sub-phase (STATUS_M5.md B3 follow-up)")
}
