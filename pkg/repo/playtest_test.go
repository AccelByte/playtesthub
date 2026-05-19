package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

const testNamespace = "testns"

func newSteamKeysPlaytest(slug string) *repo.Playtest {
	return &repo.Playtest{
		Namespace:         testNamespace,
		Slug:              slug,
		Title:             "Test playtest",
		Description:       "",
		BannerImageURL:    "",
		Platforms:         []string{"STEAM"},
		Status:            "DRAFT",
		NDARequired:       false,
		DistributionModel: "STEAM_KEYS",
	}
}

func TestPlaytestCreate_AssignsDBDefaults(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)

	got, err := store.Create(context.Background(), newSteamKeysPlaytest("test-create"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID == uuid.Nil {
		t.Error("Create did not populate ID from DB default")
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("Create did not populate created_at/updated_at")
	}
	if got.Status != "DRAFT" {
		t.Errorf("status default: got %q, want DRAFT", got.Status)
	}
	if got.DeletedAt != nil {
		t.Errorf("new row should have deleted_at=nil, got %v", *got.DeletedAt)
	}
}

func TestPlaytestCreate_SlugUniqueInNamespace(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	if _, err := store.Create(ctx, newSteamKeysPlaytest("dup-slug")); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := store.Create(ctx, newSteamKeysPlaytest("dup-slug"))
	if err == nil {
		t.Fatal("second Create with duplicate slug: got nil error, want ErrUniqueViolation")
	}
	if !errors.Is(err, repo.ErrUniqueViolation) {
		t.Errorf("second Create: got %v, want ErrUniqueViolation", err)
	}
}

// Key M1-phase-4 invariant (STATUS.md): slug uniqueness spans
// soft-deleted rows — a soft-deleted slug cannot be reused.
func TestPlaytestCreate_SlugUniqueAcrossSoftDelete(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("recycled-slug"))
	if err != nil {
		t.Fatalf("initial Create: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, created.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = store.Create(ctx, newSteamKeysPlaytest("recycled-slug"))
	if err == nil {
		t.Fatal("recreate with soft-deleted slug: got nil error, want ErrUniqueViolation")
	}
	if !errors.Is(err, repo.ErrUniqueViolation) {
		t.Errorf("recreate: got %v, want ErrUniqueViolation", err)
	}
}

func TestPlaytestGetByID_NotFound(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)

	_, err := store.GetByID(context.Background(), testNamespace, uuid.New())
	if !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("unknown id: got %v, want ErrNotFound", err)
	}
}

func TestPlaytestList_ExcludesSoftDeletedByDefault(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	keep, err := store.Create(ctx, newSteamKeysPlaytest("list-keep"))
	if err != nil {
		t.Fatalf("Create keep: %v", err)
	}
	drop, err := store.Create(ctx, newSteamKeysPlaytest("list-drop"))
	if err != nil {
		t.Fatalf("Create drop: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, drop.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	live, err := store.List(ctx, testNamespace, false)
	if err != nil {
		t.Fatalf("List(includeDeleted=false): %v", err)
	}
	if len(live) != 1 || live[0].ID != keep.ID {
		t.Errorf("List(includeDeleted=false) = %+v; want single row %v", live, keep.ID)
	}

	all, err := store.List(ctx, testNamespace, true)
	if err != nil {
		t.Fatalf("List(includeDeleted=true): %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List(includeDeleted=true) len = %d, want 2", len(all))
	}
}

func TestPlaytestUpdate_RoundTrip(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("update-me"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Title = "Renamed"
	created.Description = "new description"
	created.Platforms = []string{"STEAM", "EPIC"}
	created.NDARequired = true
	created.NDAText = "legal stuff"
	created.CurrentNDAVersionHash = "sha256:deadbeef"

	updated, err := store.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Renamed" {
		t.Errorf("title: got %q, want Renamed", updated.Title)
	}
	if updated.Description != "new description" {
		t.Errorf("description: got %q", updated.Description)
	}
	if len(updated.Platforms) != 2 {
		t.Errorf("platforms len = %d, want 2", len(updated.Platforms))
	}
	if !updated.NDARequired {
		t.Error("nda_required did not persist")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at not advanced: %v !> %v", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestPlaytestUpdate_RefusesSoftDeleted(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("deleted-then-edited"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, created.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	created.Title = "too late"
	_, err = store.Update(ctx, created)
	if !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Update on soft-deleted row: got %v, want ErrNotFound", err)
	}
}

func TestPlaytestSoftDelete_Idempotent(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("double-delete"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, created.ID); err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	err = store.SoftDelete(ctx, testNamespace, created.ID)
	if !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("second SoftDelete: got %v, want ErrNotFound", err)
	}
}

func TestPlaytestTransitionStatus_HappyPath(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("transition-ok"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	opened, err := store.TransitionStatus(ctx, testNamespace, created.ID, "DRAFT", "OPEN")
	if err != nil {
		t.Fatalf("TransitionStatus DRAFT→OPEN: %v", err)
	}
	if opened.Status != "OPEN" {
		t.Errorf("status after transition = %q, want OPEN", opened.Status)
	}
}

// Key M1-phase-4 invariant (STATUS.md): status-transition CAS —
// a preempted second writer sees ErrStatusCASMismatch, not silent
// overwrite.
func TestPlaytestTransitionStatus_CASMismatch(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("transition-race"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First writer wins.
	if _, err := store.TransitionStatus(ctx, testNamespace, created.ID, "DRAFT", "OPEN"); err != nil {
		t.Fatalf("first transition: %v", err)
	}
	// Second writer still thinks the row is DRAFT.
	_, err = store.TransitionStatus(ctx, testNamespace, created.ID, "DRAFT", "OPEN")
	if !errors.Is(err, repo.ErrStatusCASMismatch) {
		t.Errorf("race loser: got %v, want ErrStatusCASMismatch", err)
	}
}

// M4 phase 3: ListDueForAutoTransition returns DRAFTs past starts_at and
// OPENs past ends_at, but NOT rows without the relevant date set, soft-
// deleted rows, or rows whose boundary is still in the future.
func TestPlaytestListDueForAutoTransition(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Due: DRAFT with starts_at in the past.
	dueDraft := newSteamKeysPlaytest("due-draft")
	dueDraft.StartsAt = &past
	if _, err := store.Create(ctx, dueDraft); err != nil {
		t.Fatalf("Create due-draft: %v", err)
	}

	// Due: OPEN with ends_at in the past.
	dueOpen := newSteamKeysPlaytest("due-open")
	dueOpen.EndsAt = &past
	created, err := store.Create(ctx, dueOpen)
	if err != nil {
		t.Fatalf("Create due-open: %v", err)
	}
	if _, err := store.TransitionStatus(ctx, testNamespace, created.ID, "DRAFT", "OPEN"); err != nil {
		t.Fatalf("transition due-open to OPEN: %v", err)
	}

	// Not due: DRAFT with starts_at in the future.
	notYet := newSteamKeysPlaytest("not-yet")
	notYet.StartsAt = &future
	if _, err := store.Create(ctx, notYet); err != nil {
		t.Fatalf("Create not-yet: %v", err)
	}

	// Not due: DRAFT with no starts_at at all (manual mode).
	manual := newSteamKeysPlaytest("manual")
	if _, err := store.Create(ctx, manual); err != nil {
		t.Fatalf("Create manual: %v", err)
	}

	// Not due: soft-deleted DRAFT even with starts_at past.
	deleted := newSteamKeysPlaytest("deleted-due")
	deleted.StartsAt = &past
	createdDel, err := store.Create(ctx, deleted)
	if err != nil {
		t.Fatalf("Create deleted-due: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, createdDel.ID); err != nil {
		t.Fatalf("SoftDelete deleted-due: %v", err)
	}

	got, err := store.ListDueForAutoTransition(ctx, testNamespace, now)
	if err != nil {
		t.Fatalf("ListDueForAutoTransition: %v", err)
	}
	gotSlugs := make(map[string]bool, len(got))
	for _, p := range got {
		gotSlugs[p.Slug] = true
	}
	if !gotSlugs["due-draft"] || !gotSlugs["due-open"] {
		t.Errorf("missing due rows; got %v", gotSlugs)
	}
	for _, s := range []string{"not-yet", "manual", "deleted-due"} {
		if gotSlugs[s] {
			t.Errorf("row %q should not be due; got %v", s, gotSlugs)
		}
	}
}

func TestPlaytestTransitionStatus_RefusesSoftDeleted(t *testing.T) {
	truncateAll(t)
	store := repo.NewPgPlaytestStore(testPool)
	ctx := context.Background()

	created, err := store.Create(ctx, newSteamKeysPlaytest("transition-deleted"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SoftDelete(ctx, testNamespace, created.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = store.TransitionStatus(ctx, testNamespace, created.ID, "DRAFT", "OPEN")
	if !errors.Is(err, repo.ErrStatusCASMismatch) {
		t.Errorf("transition on soft-deleted: got %v, want ErrStatusCASMismatch", err)
	}
}
