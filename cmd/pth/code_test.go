package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"google.golang.org/grpc"
)

const testPlaytestID01J = "01J0CODESTPLAYTEST00000000"

func TestRunCode_NoAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	if code := runCode(t.Context(), &stdout, &stderr, g, nil, factoryFor(nil)); code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunCode_UnknownAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	if code := runCode(t.Context(), &stdout, &stderr, g, []string{"banana"}, factoryFor(nil)); code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
	if !strings.Contains(stderr.String(), "banana") {
		t.Errorf("stderr should name action, got %q", stderr.String())
	}
}

func TestRunCodeUpload_Success(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "keys.csv")
	if err := os.WriteFile(csvPath, []byte("ABCD-1234\nEFGH-5678\n"), 0o600); err != nil {
		t.Fatalf("seed csv: %v", err)
	}
	stub := &stubPlaytestClient{
		uploadCodesFunc: func(_ context.Context, in *pb.UploadCodesRequest, _ ...grpc.CallOption) (*pb.UploadCodesResponse, error) {
			if in.PlaytestId != testPlaytestID01J {
				t.Errorf("playtest_id=%q", in.PlaytestId)
			}
			if in.Filename != "keys.csv" {
				t.Errorf("filename=%q, want keys.csv", in.Filename)
			}
			if !strings.Contains(in.CsvContent, "ABCD-1234") {
				t.Errorf("csv body did not propagate: %q", in.CsvContent)
			}
			return &pb.UploadCodesResponse{Inserted: 2}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"upload",
		"--playtest", testPlaytestID01J,
		"--file", csvPath,
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
	if stub.calls != 1 {
		t.Errorf("calls=%d", stub.calls)
	}
}

func TestRunCodeUpload_RequiresFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"upload",
		"--playtest", testPlaytestID01J,
	}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
	if !strings.Contains(stderr.String(), "--file") {
		t.Errorf("stderr should name --file: %q", stderr.String())
	}
}

func TestRunCodeUpload_RequiresPlaytest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{"upload", "--file", "k.csv"}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunCodeTopUp_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		topUpCodesFunc: func(_ context.Context, in *pb.TopUpCodesRequest, _ ...grpc.CallOption) (*pb.TopUpCodesResponse, error) {
			if in.Quantity != 250 {
				t.Errorf("quantity=%d, want 250", in.Quantity)
			}
			return &pb.TopUpCodesResponse{Added: 250}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"top-up",
		"--playtest", testPlaytestID01J,
		"--quantity", "250",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunCodeTopUp_RequiresQuantity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"top-up",
		"--playtest", testPlaytestID01J,
	}, factoryFor(nil))
	if code != exitLocalError {
		t.Fatalf("exit=%d, want %d", code, exitLocalError)
	}
}

func TestRunCodeSyncFromAGS_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		syncFromAGSFunc: func(_ context.Context, in *pb.SyncFromAGSRequest, _ ...grpc.CallOption) (*pb.SyncFromAGSResponse, error) {
			if in.PlaytestId != testPlaytestID01J {
				t.Errorf("playtest_id=%q", in.PlaytestId)
			}
			return &pb.SyncFromAGSResponse{Added: 5}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"sync-from-ags",
		"--playtest", testPlaytestID01J,
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunCodePool_Success(t *testing.T) {
	stub := &stubPlaytestClient{
		getCodePoolFunc: func(_ context.Context, in *pb.GetCodePoolRequest, _ ...grpc.CallOption) (*pb.GetCodePoolResponse, error) {
			if in.PlaytestId != testPlaytestID01J {
				t.Errorf("playtest_id=%q", in.PlaytestId)
			}
			return &pb.GetCodePoolResponse{Stats: &pb.CodePoolStats{Total: 1}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{"pool", "--playtest", testPlaytestID01J}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
}

func TestRunCodePool_DryRun(t *testing.T) {
	stub := &stubPlaytestClient{
		getCodePoolFunc: func(_ context.Context, _ *pb.GetCodePoolRequest, _ ...grpc.CallOption) (*pb.GetCodePoolResponse, error) {
			t.Fatal("dry-run must not dial")
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	g := &Globals{Addr: "localhost:6565", Namespace: testNamespaceDev}
	code := runCode(t.Context(), &stdout, &stderr, g, []string{
		"pool",
		"--playtest", testPlaytestID01J,
		"--dry-run",
	}, factoryFor(stub))
	if code != exitOK {
		t.Fatalf("exit=%d, want %d (stderr=%q)", code, exitOK, stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &got); err != nil {
		t.Fatalf("stdout not JSON: %v: %q", err, stdout.String())
	}
	if got["playtest_id"] != testPlaytestID01J {
		t.Errorf("playtest_id wrong: %v", got["playtest_id"])
	}
}
