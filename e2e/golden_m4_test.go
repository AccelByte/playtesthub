package e2e_test

import (
	"testing"
)

// TestGoldenM4Flow drives the M4 window-enforcement path end-to-end:
//
//   - admin login → user create → login-as keep the boot pattern
//     identical to M1/M2/M3, even though golden-m4 itself is admin-only,
//     so a future expansion that adds player steps can reuse the same
//     scaffolding without test churn.
//
//   - `pth flow golden-m4` then drives four NDJSON steps:
//
//     1. create-playtest (DRAFT + startsAt=now+2s + endsAt=now+4s)
//     2. await-auto-open (poll AdminGetPlaytest until status=OPEN)
//     3. await-auto-close (poll until status=CLOSED)
//     4. assert-system-transitions (ListAuditLog actor=system,
//     action=playtest.status_transition; expect ≥2 rows)
//
// The window worker is started by suiteHarness with TickInterval=1s so
// the bounded ~10s runtime budget holds. The test asserts every step
// emits status=OK and that the terminal audit-log row count is at
// least 2 (one DRAFT→OPEN row, one OPEN→CLOSED row, both system-
// attributed per schema.md).
func TestGoldenM4Flow(t *testing.T) {
	h := getHarness(t)
	suffix := uniqueSuffix(t)
	slug := "e2e-m4-" + suffix
	adminProfile := "e2e-m4-admin-" + suffix

	t.Logf("e2e harness: addr=%s suffix=%s namespace=%s", h.addr, suffix, h.env.AGSNamespace)

	// 1. Admin login.
	loginOut := runPTH(t, h, runOpts{
		stdin: h.env.AdminPassword,
		args: []string{
			"--addr", h.addr, "--insecure",
			"--namespace", h.env.AGSNamespace,
			"--profile", adminProfile,
			"auth", "login", "--password",
			"--username", h.env.AdminUsername,
			"--password-stdin",
		},
	})
	if got := jsonString(t, loginOut, "userId"); got == "" {
		t.Fatalf("auth login: missing userId in response: %s", loginOut)
	}

	// 2. Run the M4 composite flow.
	flowOut := runPTH(t, h, runOpts{
		args: []string{
			"--addr", h.addr, "--insecure",
			"--namespace", h.env.AGSNamespace,
			"flow", "golden-m4", "--slug", slug,
			"--admin-profile", adminProfile,
			// The harness worker ticks every 1s; pad the poll budget so
			// CI scheduling jitter doesn't flake the wait.
			"--start-offset", "2s",
			"--end-offset", "4s",
			"--poll-interval", "250ms",
			"--poll-timeout-open", "15s",
			"--poll-timeout-close", "15s",
		},
	})

	steps := parseNDJSON(t, flowOut)
	wantOrder := []string{
		"create-playtest",
		"await-auto-open",
		"await-auto-close",
		"assert-system-transitions",
	}
	if len(steps) != len(wantOrder) {
		t.Fatalf("flow emitted %d lines, want %d: %s", len(steps), len(wantOrder), flowOut)
	}
	for i, want := range wantOrder {
		if got := jsonString(t, steps[i], "step"); got != want {
			t.Fatalf("flow line %d step=%q want %q (line: %s)", i+1, got, want, steps[i])
		}
		if got := jsonString(t, steps[i], "status"); got != "OK" {
			t.Fatalf("flow line %d status=%q want OK (line: %s)", i+1, got, steps[i])
		}
	}

	// Soft-delete the playtest on teardown.
	playtestID := jsonNested(t, steps[0], "response", "playtest", "id")
	if playtestID == "" {
		t.Fatalf("create-playtest response missing playtest.id: %s", steps[0])
	}
	t.Cleanup(func() {
		_, _ = tryPTH(h, runOpts{
			args: []string{
				"--addr", h.addr, "--insecure",
				"--namespace", h.env.AGSNamespace,
				"--profile", adminProfile,
				"playtest", "delete", "--id", playtestID,
			},
		})
	})

	// Belt-and-braces: the final audit-log step really carries ≥2
	// system-attributed playtest.status_transition rows. The CLI
	// already short-circuits if not, but pinning here keeps the
	// e2e contract explicit.
	if n := jsonArrayLen(t, steps[3], "response", "entries"); n < 2 {
		t.Fatalf("assert-system-transitions returned %d audit rows, want >=2: %s", n, steps[3])
	}
}
