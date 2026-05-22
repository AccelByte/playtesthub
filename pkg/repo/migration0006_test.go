package repo_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestMigration0006_PlaytestADTColumns pins the four ADT columns added
// to playtest by migration 0006: adt_namespace, adt_game_id, adt_build_id,
// adt_fallback_download_url — all TEXT NULL. See docs/PRD.md §5.1 and
// docs/STATUS_M5.md B2.
func TestMigration0006_PlaytestADTColumns(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	const sql = `
		SELECT column_name, data_type, is_nullable
		  FROM information_schema.columns
		 WHERE table_schema = 'public'
		   AND table_name   = 'playtest'
		   AND column_name IN ('adt_namespace', 'adt_game_id', 'adt_build_id', 'adt_fallback_download_url')
		 ORDER BY column_name`

	rows, err := testPool.Query(ctx, sql)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type col struct{ name, dataType, isNullable string }
	var got []col
	for rows.Next() {
		var c col
		if scanErr := rows.Scan(&c.name, &c.dataType, &c.isNullable); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		got = append(got, c)
	}
	if rows.Err() != nil {
		t.Fatalf("rows.Err: %v", rows.Err())
	}

	want := []col{
		{"adt_build_id", "text", "YES"},
		{"adt_fallback_download_url", "text", "YES"},
		{"adt_game_id", "text", "YES"},
		{"adt_namespace", "text", "YES"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMigration0006_DistributionModelEnumIncludesADT confirms the
// extended enum CHECK now accepts 'ADT'.
func TestMigration0006_DistributionModelEnumIncludesADT(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	_, err := testPool.Exec(ctx,
		`INSERT INTO playtest (namespace, slug, title, distribution_model, adt_namespace, adt_game_id, adt_build_id)
		 VALUES ($1, $2, $3, 'ADT', $4, $5, $6)`,
		"ns", "adt-enum-ok", "title", "adt-ns", "game-1", "build-1")
	if err != nil {
		t.Fatalf("insert ADT playtest: %v", err)
	}

	// A nonsense model still fails the enum check.
	_, err = testPool.Exec(ctx,
		`INSERT INTO playtest (namespace, slug, title, distribution_model)
		 VALUES ($1, $2, $3, 'BOGUS')`,
		"ns", "bogus-enum", "title")
	if err == nil {
		t.Fatalf("BOGUS model accepted; want CHECK rejection")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "playtest_distribution_model_enum" {
		t.Errorf("err = %v; want playtest_distribution_model_enum violation", err)
	}
}

// TestMigration0006_ADTNamespaceModelCheckRejects pins the
// model↔adt_namespace invariant: distribution_model='ADT' iff
// adt_namespace IS NOT NULL.
func TestMigration0006_ADTNamespaceModelCheckRejects(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		sql   string
		args  []any
		label string
	}{
		{
			name: "adt_model_null_namespace",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model, adt_namespace, adt_game_id, adt_build_id)
			      VALUES ($1, $2, $3, 'ADT', NULL, 'g', 'b')`,
			args:  []any{"ns", "adt-null-ns", "title"},
			label: "ADT + adt_namespace=NULL",
		},
		{
			name: "steam_keys_with_adt_namespace",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model, adt_namespace)
			      VALUES ($1, $2, $3, 'STEAM_KEYS', 'adt-ns')`,
			args:  []any{"ns", "steam-with-adt", "title"},
			label: "STEAM_KEYS + adt_namespace set",
		},
		{
			name: "ags_campaign_with_adt_namespace",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model, initial_code_quantity, adt_namespace)
			      VALUES ($1, $2, $3, 'AGS_CAMPAIGN', 100, 'adt-ns')`,
			args:  []any{"ns", "ags-with-adt", "title"},
			label: "AGS_CAMPAIGN + adt_namespace set",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testPool.Exec(ctx, tc.sql, tc.args...)
			if err == nil {
				t.Fatalf("%s: insert succeeded, want CHECK rejection", tc.label)
			}
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("%s: err is %T, want *pgconn.PgError: %v", tc.label, err, err)
			}
			if pgErr.ConstraintName != "playtest_adt_namespace_model" {
				t.Errorf("%s: constraint = %q, want playtest_adt_namespace_model", tc.label, pgErr.ConstraintName)
			}
		})
	}
}

// TestMigration0006_ADTNamespaceModelCheckAccepts confirms the valid
// combinations the CHECK constraint allows.
func TestMigration0006_ADTNamespaceModelCheckAccepts(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	cases := []struct {
		slug string
		sql  string
		args []any
	}{
		{
			slug: "adt-happy",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model, adt_namespace, adt_game_id, adt_build_id, adt_fallback_download_url)
			      VALUES ($1, $2, $3, 'ADT', $4, $5, $6, $7)`,
			args: []any{"ns", "adt-happy", "title", "adt-ns", "g", "b", "https://example.com/build"},
		},
		{
			slug: "steam-happy",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model)
			      VALUES ($1, $2, $3, 'STEAM_KEYS')`,
			args: []any{"ns", "steam-happy", "title"},
		},
		{
			slug: "ags-happy",
			sql: `INSERT INTO playtest (namespace, slug, title, distribution_model, initial_code_quantity)
			      VALUES ($1, $2, $3, 'AGS_CAMPAIGN', 50)`,
			args: []any{"ns", "ags-happy", "title"},
		},
	}
	for _, tc := range cases {
		_, err := testPool.Exec(ctx, tc.sql, tc.args...)
		if err != nil {
			t.Errorf("insert slug=%s: %v", tc.slug, err)
		}
	}
}

// TestMigration0006_InitialCodeQuantityForbiddenOnADT keeps the
// initial_code_quantity_model rule intact for ADT: NULL only.
func TestMigration0006_InitialCodeQuantityForbiddenOnADT(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	_, err := testPool.Exec(ctx,
		`INSERT INTO playtest (namespace, slug, title, distribution_model, initial_code_quantity, adt_namespace, adt_game_id, adt_build_id)
		 VALUES ($1, $2, $3, 'ADT', 100, $4, $5, $6)`,
		"ns", "adt-with-code-qty", "title", "adt-ns", "g", "b")
	if err == nil {
		t.Fatalf("ADT + initial_code_quantity accepted; want CHECK rejection")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "playtest_initial_code_quantity_model" {
		t.Errorf("err = %v; want playtest_initial_code_quantity_model violation", err)
	}
}

// TestMigration0006_ADTLinkageTableShape pins the adt_linkage column
// set + types. Regression canary: NO `adt_credential_*` columns exist
// (PRD §4.8 / schema.md §"adt_linkage table" — auth is the AGS service
// IAM JWT, no credential is stored).
func TestMigration0006_ADTLinkageTableShape(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	const sql = `
		SELECT column_name, data_type, is_nullable
		  FROM information_schema.columns
		 WHERE table_schema = 'public'
		   AND table_name   = 'adt_linkage'
		 ORDER BY column_name`

	rows, err := testPool.Query(ctx, sql)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type col struct{ name, dataType, isNullable string }
	var got []col
	for rows.Next() {
		var c col
		if scanErr := rows.Scan(&c.name, &c.dataType, &c.isNullable); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		got = append(got, c)
	}
	if rows.Err() != nil {
		t.Fatalf("rows.Err: %v", rows.Err())
	}

	want := []col{
		{"adt_namespace", "text", "NO"},
		{"deleted_at", "timestamp with time zone", "YES"},
		{"id", "uuid", "NO"},
		{"linked_at", "timestamp with time zone", "NO"},
		{"linked_by_user_id", "uuid", "NO"},
		{"studio_namespace", "text", "NO"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d columns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("col[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Regression canary: no adt_credential_* column may exist on
	// adt_linkage. The no-credential-storage property is load-bearing
	// (PRD §4.8 / STATUS_M5.md "no ADT credential exists to leak").
	for _, c := range got {
		if strings.HasPrefix(c.name, "adt_credential") {
			t.Errorf("adt_linkage has %q column; the no-credential-storage property is violated", c.name)
		}
	}
}

// TestMigration0006_ADTLinkagePartialUnique pins the partial unique
// index that allows re-linking the same adt_namespace after a soft
// delete (schema.md §"adt_linkage table").
func TestMigration0006_ADTLinkagePartialUnique(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	const sql = `
		SELECT indexdef
		  FROM pg_indexes
		 WHERE schemaname = 'public'
		   AND tablename  = 'adt_linkage'
		   AND indexname  = 'adt_linkage_studio_adt_uniq'`

	var indexDef string
	if err := testPool.QueryRow(ctx, sql).Scan(&indexDef); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if !strings.Contains(indexDef, "UNIQUE INDEX") {
		t.Errorf("indexdef %q missing UNIQUE INDEX", indexDef)
	}
	if !strings.Contains(indexDef, "(studio_namespace, adt_namespace)") {
		t.Errorf("indexdef %q missing (studio_namespace, adt_namespace)", indexDef)
	}
	if !strings.Contains(indexDef, "deleted_at IS NULL") {
		t.Errorf("indexdef %q missing WHERE deleted_at IS NULL", indexDef)
	}

	// Behavioural check: two live rows for the same (studio_ns,
	// adt_ns) are forbidden; but re-link after soft delete succeeds.
	uid := "00000000-0000-0000-0000-000000000001"
	_, err := testPool.Exec(ctx,
		`INSERT INTO adt_linkage (studio_namespace, adt_namespace, linked_by_user_id) VALUES ($1, $2, $3)`,
		"studio-1", "adt-ns", uid)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err = testPool.Exec(ctx,
		`INSERT INTO adt_linkage (studio_namespace, adt_namespace, linked_by_user_id) VALUES ($1, $2, $3)`,
		"studio-1", "adt-ns", uid)
	if err == nil {
		t.Fatalf("duplicate live insert succeeded; want unique violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Errorf("err = %v; want unique_violation (23505)", err)
	}

	// Soft-delete the first row → next insert succeeds (re-link path).
	if _, err = testPool.Exec(ctx,
		`UPDATE adt_linkage SET deleted_at = NOW() WHERE studio_namespace = $1 AND adt_namespace = $2`,
		"studio-1", "adt-ns"); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err = testPool.Exec(ctx,
		`INSERT INTO adt_linkage (studio_namespace, adt_namespace, linked_by_user_id) VALUES ($1, $2, $3)`,
		"studio-1", "adt-ns", uid); err != nil {
		t.Errorf("re-link after soft delete: %v", err)
	}
}

// TestMigration0006_ADTLinkPendingShape pins adt_link_pending column
// set + types.
func TestMigration0006_ADTLinkPendingShape(t *testing.T) {
	truncateAll(t)
	ctx := context.Background()

	const sql = `
		SELECT column_name, data_type, is_nullable
		  FROM information_schema.columns
		 WHERE table_schema = 'public'
		   AND table_name   = 'adt_link_pending'
		 ORDER BY column_name`

	rows, err := testPool.Query(ctx, sql)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type col struct{ name, dataType, isNullable string }
	var got []col
	for rows.Next() {
		var c col
		if scanErr := rows.Scan(&c.name, &c.dataType, &c.isNullable); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		got = append(got, c)
	}
	if rows.Err() != nil {
		t.Fatalf("rows.Err: %v", rows.Err())
	}

	want := []col{
		{"expires_at", "timestamp with time zone", "NO"},
		{"started_by_user_id", "uuid", "NO"},
		{"state", "text", "NO"},
		{"studio_namespace", "text", "NO"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d columns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("col[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
