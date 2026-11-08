//go:build integration

package integration

import (
	"context"
	"database/sql"
	"strconv"
	"sync"
	"testing"

	"investment_monitoring/internal/audit"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/internal/testutil"

	"github.com/google/uuid"
)

func verifyChain(t *testing.T, db *sql.DB) (brokenSeq int64, reason string, intact bool) {
	t.Helper()
	err := db.QueryRow(`SELECT broken_seq, reason FROM verify_audit_log_chain()`).Scan(&brokenSeq, &reason)
	if err == sql.ErrNoRows {
		return 0, "", true
	}
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	return brokenSeq, reason, false
}

func TestAuditLogIntegrity(t *testing.T) {
	db, store := testutil.OpenTestDB(t)
	f := testutil.NewFixture(t, db)
	ctx := context.Background()
	recorder := audit.NewService(store)

	org := f.Org(entity.OrganizationRoleEnumEXECUTINGAGENCY)
	actor := f.Member(org, entity.MemberRoleEnumADMIN)
	record := func() uuid.UUID {
		entityID := uuid.New()
		err := store.ExecTx(ctx, func(q *entity.Queries) error {
			return recorder.RecordTx(ctx, q, audit.RecordParams{
				ActorUserID: actor.UserID,
				ActorOrgID:  actor.OrgID,
				Action:      audit.ActionUpdateTicketStatus,
				EntityType:  audit.EntityTicket,
				EntityID:    entityID,
				BeforeData:  map[string]any{"status": "IN_PROGRESS"},
				AfterData:   map[string]any{"status": "RESOLVED"},
			})
		})
		if err != nil {
			t.Fatalf("record audit log: %v", err)
		}
		return entityID
	}

	t.Run("AUDIT-01 every inserted row is hashed and linked to its predecessor", func(t *testing.T) {
		first, second := record(), record()
		var firstHash, secondPrev, secondHash string
		if err := db.QueryRow(`SELECT row_hash FROM audit_logs WHERE entity_id = $1`, first).Scan(&firstHash); err != nil {
			t.Fatalf("first row: %v", err)
		}
		if err := db.QueryRow(`SELECT prev_hash, row_hash FROM audit_logs WHERE entity_id = $1`, second).Scan(&secondPrev, &secondHash); err != nil {
			t.Fatalf("second row: %v", err)
		}
		if len(firstHash) != 64 || len(secondHash) != 64 {
			t.Fatalf("hash lengths = %d, %d, want 64 hex characters", len(firstHash), len(secondHash))
		}
		if secondPrev != firstHash {
			t.Fatalf("second.prev_hash = %s, want first.row_hash %s", secondPrev, firstHash)
		}
		if _, reason, intact := verifyChain(t, db); !intact {
			t.Fatalf("chain broken: %s", reason)
		}
	})

	t.Run("AUDIT-02 UPDATE, DELETE and TRUNCATE are rejected", func(t *testing.T) {
		entityID := record()
		for name, stmt := range map[string]string{
			"update":   `UPDATE audit_logs SET action = 'TAMPERED' WHERE entity_id = '` + entityID.String() + `'`,
			"delete":   `DELETE FROM audit_logs WHERE entity_id = '` + entityID.String() + `'`,
			"truncate": `TRUNCATE audit_logs`,
		} {
			if _, err := db.Exec(stmt); err == nil {
				t.Errorf("%s on audit_logs succeeded, want it rejected", name)
			}
		}
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM audit_logs WHERE entity_id = $1 AND action = $2`, entityID, audit.ActionUpdateTicketStatus).Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 1 {
			t.Fatalf("row count = %d, want the original row untouched", count)
		}
	})

	t.Run("AUDIT-03 deleting an actor referenced by the trail is rejected", func(t *testing.T) {
		record()
		if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, actor.UserID); err == nil {
			t.Fatal("deleting a user referenced by audit_logs succeeded")
		}
	})

	t.Run("AUDIT-04 tampering that bypasses the triggers is detected", func(t *testing.T) {
		entityID := record()
		var seq int64
		var original string
		if err := db.QueryRow(`SELECT seq, after_data::text FROM audit_logs WHERE entity_id = $1`, entityID).Scan(&seq, &original); err != nil {
			t.Fatalf("row: %v", err)
		}

		// A privileged user disables the guard and rewrites history.
		tamper := func(afterData string) {
			t.Helper()
			for _, stmt := range []string{
				`ALTER TABLE audit_logs DISABLE TRIGGER audit_logs_no_update`,
				`UPDATE audit_logs SET after_data = '` + afterData + `'::jsonb WHERE seq = ` + strconv.FormatInt(seq, 10),
				`ALTER TABLE audit_logs ENABLE TRIGGER audit_logs_no_update`,
			} {
				if _, err := db.Exec(stmt); err != nil {
					t.Fatalf("tamper: %v", err)
				}
			}
		}
		tamper(`{"status": "CLOSED"}`)
		brokenSeq, reason, intact := verifyChain(t, db)
		if intact || brokenSeq != seq {
			t.Fatalf("verify = (seq %d, %q, intact=%v), want the tampered row %d reported", brokenSeq, reason, intact, seq)
		}
		t.Logf("AUDIT-04 detected tampering at seq=%d: %s", brokenSeq, reason)

		tamper(original)
		if _, reason, intact := verifyChain(t, db); !intact {
			t.Fatalf("chain still broken after restoring the row: %s", reason)
		}
	})

	t.Run("AUDIT-05 concurrent inserts keep a single unforked chain", func(t *testing.T) {
		const writers = 25
		var wg sync.WaitGroup
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				record()
			}()
		}
		wg.Wait()

		var forks int
		if err := db.QueryRow(`SELECT count(*) FROM (SELECT prev_hash FROM audit_logs WHERE prev_hash IS NOT NULL GROUP BY prev_hash HAVING count(*) > 1) d`).Scan(&forks); err != nil {
			t.Fatalf("fork check: %v", err)
		}
		if forks != 0 {
			t.Fatalf("chain forked at %d points", forks)
		}
		if _, reason, intact := verifyChain(t, db); !intact {
			t.Fatalf("chain broken after concurrent inserts: %s", reason)
		}
	})
}
