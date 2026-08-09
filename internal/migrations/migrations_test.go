package migrations

import "testing"

// TestGetMigrations_VersionsAreSequentialAndWellFormed guards against the
// class of bug that's easy to introduce by hand-editing a migration list:
// a skipped version number, a duplicate, or a migration with no Up
// function would silently break Migrate()'s "run everything newer than
// the current version" logic.
func TestGetMigrations_VersionsAreSequentialAndWellFormed(t *testing.T) {
	migs := getMigrations()
	if len(migs) == 0 {
		t.Fatal("expected at least one migration")
	}

	seen := make(map[int]bool)
	for i, m := range migs {
		if m.Version != i+1 {
			t.Errorf("migration at index %d has version %d, expected sequential version %d", i, m.Version, i+1)
		}
		if seen[m.Version] {
			t.Errorf("duplicate migration version %d", m.Version)
		}
		seen[m.Version] = true

		if m.Description == "" {
			t.Errorf("migration %d has an empty description", m.Version)
		}
		if m.Up == nil {
			t.Errorf("migration %d has a nil Up function", m.Version)
		}
	}
}

func TestNewMigrator_LoadsMigrations(t *testing.T) {
	m := NewMigrator(nil)
	if len(m.migrations) == 0 {
		t.Error("expected NewMigrator to load a non-empty migration list")
	}
}
