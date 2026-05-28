package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/anggorodewanto/playtesthub/pkg/adt"
	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
)

// TestListADTBuilds_MapsBuildType verifies the service handler surfaces
// ADT's `build_type` discriminator end-to-end so the admin picker can
// grey out non-downloadable (smartbuild) entries. Wire-added 2026-05-28
// alongside the ADT builds-list addendum.
func TestListADTBuilds_MapsBuildType(t *testing.T) {
	h := newADTTestServer(t)
	linkage := h.linkage.live[testStudioNamespace+"|"+testADTNamespace]
	h.mem.SeedBuilds(testADTNamespace, testADTGameID, []adt.Build{
		{ID: "b-info", Name: "0.0.3", Version: "v3", Platform: "windows", BuildType: adt.BuildTypeBuildInfo, UploadedAt: time.Now()},
		{ID: "b-smart", Name: "0.0.2 SB", Version: "v2", Platform: "windows", BuildType: adt.BuildTypeSmartBuild, UploadedAt: time.Now().Add(-time.Hour)},
	})

	resp, err := h.svr.ListADTBuilds(authCtx(uuid.New()), &pb.ListADTBuildsRequest{
		Namespace:    testNamespace,
		AdtLinkageId: linkage.ID.String(),
		AdtGameId:    testADTGameID,
	})
	if err != nil {
		t.Fatalf("ListADTBuilds: %v", err)
	}
	byID := map[string]*pb.ADTBuild{}
	for _, b := range resp.GetBuilds() {
		byID[b.GetId()] = b
	}
	if got := byID["b-info"].GetBuildType(); got != "buildinfo" {
		t.Errorf("buildinfo build_type = %q, want buildinfo", got)
	}
	if got := byID["b-smart"].GetBuildType(); got != "smartbuild" {
		t.Errorf("smartbuild build_type = %q, want smartbuild", got)
	}
}
