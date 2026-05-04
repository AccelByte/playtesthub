package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"google.golang.org/grpc"
)

func TestRunApplicantSignup_DryRun(t *testing.T) {
	stub := &stubPlaytestClient{
		signupFunc: func(_ context.Context, _ *pb.SignupRequest, _ ...grpc.CallOption) (*pb.SignupResponse, error) {
			t.Fatal("dry-run must not dial")
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"signup",
		"--slug", testSlugDemo01,
		"--platforms", "STEAM,XBOX",
		"--dry-run",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &got); err != nil {
		t.Fatalf("stdout not JSON: %v: %q", err, stdout.String())
	}
	if got["slug"] != testSlugDemo01 {
		t.Errorf("slug wrong: %v", got["slug"])
	}
	platforms, ok := got["platforms"].([]any)
	if !ok || len(platforms) != 2 {
		t.Fatalf("platforms wrong: %v", got["platforms"])
	}
}

func TestRunApplicantSignup_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		signupFunc: func(_ context.Context, in *pb.SignupRequest, _ ...grpc.CallOption) (*pb.SignupResponse, error) {
			if in.Slug != testSlugDemo01 {
				t.Errorf("slug=%q", in.Slug)
			}
			return &pb.SignupResponse{Applicant: &pb.Applicant{Id: "a1"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"signup",
		"--slug", testSlugDemo01,
		"--platforms", "STEAM",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
	if stub.calls != 1 {
		t.Errorf("expected 1 call, got %d", stub.calls)
	}
}

func TestRunApplicantSignup_RequiresPlatforms(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"signup",
		"--slug", testSlugDemo01,
	}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("missing --platforms exit=%d, want %d", code, exitLocalError)
	}
	if !strings.Contains(stderr.String(), "--platforms") {
		t.Errorf("stderr should name --platforms, got %q", stderr.String())
	}
}

func TestRunApplicantSignup_RequiresSlug(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"signup",
		"--platforms", "STEAM",
	}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("missing --slug exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunApplicantStatus_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		applicantStatusFunc: func(_ context.Context, in *pb.GetApplicantStatusRequest, _ ...grpc.CallOption) (*pb.GetApplicantStatusResponse, error) {
			if in.Slug != testSlugDemo01 {
				t.Errorf("slug=%q", in.Slug)
			}
			return &pb.GetApplicantStatusResponse{Applicant: &pb.Applicant{Id: "a1", Status: pb.ApplicantStatus_APPLICANT_STATUS_PENDING}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"status", "--slug", testSlugDemo01}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantAcceptNDA_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		acceptNDAFunc: func(_ context.Context, in *pb.AcceptNDARequest, _ ...grpc.CallOption) (*pb.AcceptNDAResponse, error) {
			if in.PlaytestId != testPlaytestID01J {
				t.Errorf("playtest_id=%q", in.PlaytestId)
			}
			return &pb.AcceptNDAResponse{Acceptance: &pb.NDAAcceptance{NdaVersionHash: "h"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"accept-nda",
		"--playtest", testPlaytestID01J,
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantAcceptNDA_RequiresPlaytest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	if code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"accept-nda"}, factoryFor(nil)); code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunApplicantList_PassesFilters(t *testing.T) {
	stub := &stubPlaytestClient{
		listApplicantsFunc: func(_ context.Context, in *pb.ListApplicantsRequest, _ ...grpc.CallOption) (*pb.ListApplicantsResponse, error) {
			if in.StatusFilter != pb.ApplicantStatus_APPLICANT_STATUS_PENDING {
				t.Errorf("status_filter=%v", in.StatusFilter)
			}
			if !in.DmFailedFilter {
				t.Errorf("dm_failed_filter not propagated")
			}
			if in.PageSize != 25 {
				t.Errorf("page_size=%d", in.PageSize)
			}
			return &pb.ListApplicantsResponse{}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"list",
		"--playtest", testPlaytestID01J,
		"--status", "PENDING",
		"--dm-failed",
		"--page-size", "25",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantList_RejectsBadStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"list",
		"--playtest", testPlaytestID01J,
		"--status", "BANANA",
	}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
	if !strings.Contains(stderr.String(), "BANANA") {
		t.Errorf("stderr should echo bad value: %q", stderr.String())
	}
}

func TestRunApplicantApprove_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		approveFunc: func(_ context.Context, in *pb.ApproveApplicantRequest, _ ...grpc.CallOption) (*pb.ApproveApplicantResponse, error) {
			if in.ApplicantId != "a-1" {
				t.Errorf("applicant_id=%q", in.ApplicantId)
			}
			return &pb.ApproveApplicantResponse{Applicant: &pb.Applicant{Id: "a-1", Status: pb.ApplicantStatus_APPLICANT_STATUS_APPROVED}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"approve", "--id", "a-1"}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantReject_PassesReason(t *testing.T) {
	stub := &stubPlaytestClient{
		rejectFunc: func(_ context.Context, in *pb.RejectApplicantRequest, _ ...grpc.CallOption) (*pb.RejectApplicantResponse, error) {
			if in.RejectionReason == nil || *in.RejectionReason != "duplicate" {
				t.Errorf("rejection_reason=%v", in.RejectionReason)
			}
			return &pb.RejectApplicantResponse{Applicant: &pb.Applicant{Id: "a-1", Status: pb.ApplicantStatus_APPLICANT_STATUS_REJECTED}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{
		"reject",
		"--id", "a-1",
		"--reason", "duplicate",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantReject_OmitsAbsentReason(t *testing.T) {
	stub := &stubPlaytestClient{
		rejectFunc: func(_ context.Context, in *pb.RejectApplicantRequest, _ ...grpc.CallOption) (*pb.RejectApplicantResponse, error) {
			if in.RejectionReason != nil {
				t.Errorf("rejection_reason should be nil when --reason omitted, got %q", *in.RejectionReason)
			}
			return &pb.RejectApplicantResponse{Applicant: &pb.Applicant{Id: "a-1"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"reject", "--id", "a-1"}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantRetryDM_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		retryDMFunc: func(_ context.Context, in *pb.RetryDMRequest, _ ...grpc.CallOption) (*pb.RetryDMResponse, error) {
			if in.ApplicantId != "a-1" {
				t.Errorf("applicant_id=%q", in.ApplicantId)
			}
			return &pb.RetryDMResponse{Applicant: &pb.Applicant{Id: "a-1"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"retry-dm", "--id", "a-1"}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunApplicantGetCode_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		getGrantedCodeFunc: func(_ context.Context, in *pb.GetGrantedCodeRequest, _ ...grpc.CallOption) (*pb.GetGrantedCodeResponse, error) {
			if in.PlaytestId != testPlaytestID01J {
				t.Errorf("playtest_id=%q", in.PlaytestId)
			}
			return &pb.GetGrantedCodeResponse{
				Value:             "ABCD-1234",
				DistributionModel: pb.DistributionModel_DISTRIBUTION_MODEL_STEAM_KEYS,
			}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"get-code", "--playtest", testPlaytestID01J}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ABCD-1234") {
		t.Errorf("stdout missing code value: %q", stdout.String())
	}
}

func TestRunApplicant_NoAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, nil, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("no action exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunApplicant_UnknownAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565"}
	code := runApplicant(t.Context(), &stdout, &stderr, g, []string{"banana"}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("unknown action exit=%d, want %d", code, exitLocalError)
	}
	if !strings.Contains(stderr.String(), "banana") {
		t.Errorf("stderr should name action, got %q", stderr.String())
	}
}
