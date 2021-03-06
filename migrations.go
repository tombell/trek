package trek

import (
	"database/sql"
	"io/ioutil"
	"path"
	"sort"
	"strings"
)

// Migrations represents a set of migrations to apply to or rollback from a
// database.
type Migrations []*Migration

func (m Migrations) Len() int           { return len(m) }
func (m Migrations) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m Migrations) Less(i, j int) bool { return m[i].Version.Before(m[j].Version) }

// LoadMigrations returns all the loaded migrations from the given directory
// path.
func LoadMigrations(migrationsPath string) (Migrations, error) {
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return nil, err
	}

	var migrations Migrations

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		migration, err := NewMigration(path.Join(migrationsPath, file.Name()))
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// Migrate applies all the migrations that have not already been applied to the
// given database.
func (m Migrations) Migrate(driver Driver, db *sql.DB) error {
	sort.Sort(m)

	for _, migration := range m {
		hasBeenMigrated, err := migration.HasBeenMigrated(driver, db)
		if err != nil {
			return err
		}

		if !hasBeenMigrated {
			if err := migration.Migrate(driver, db); err != nil {
				return err
			}
		}
	}

	return nil
}

// Rollback rolls back all the migrations that have been applied to the given
// database.
func (m Migrations) Rollback(driver Driver, db *sql.DB, steps int) error {
	sort.Sort(sort.Reverse(m))

	if steps <= 0 {
		steps = len(m)
	}

	for _, migration := range m[:steps] {
		hasBeenMigrated, err := migration.HasBeenMigrated(driver, db)
		if err != nil {
			return err
		}

		if hasBeenMigrated {
			if err := migration.Rollback(driver, db); err != nil {
				return err
			}
		}
	}

	return nil
}
