package trek

import "database/sql"

// Driver ...
type Driver interface {
	CreateVersionsTable(db *sql.DB) error
	HasVersionBeenExecuted(db *sql.DB, version string) (bool, error)
	MarkVersionAsExecuted(tx *sql.Tx, version string) error
	UnmarkVersionAsExecuted(tx *sql.Tx, version string) error
}

// PostgresDriver ...
type PostgresDriver struct{}

// CreateVersionsTable ...
func (d *PostgresDriver) CreateVersionsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS database_versions(version TEXT);`)
	return err
}

// HasVersionBeenExecuted ...
func (d *PostgresDriver) HasVersionBeenExecuted(db *sql.DB, version string) (bool, error) {
	var count int

	if err := db.Get(&count, "SELECT COUNT(*) FROM database_versions WHERE version=$1", version); err != nil {
		return false, err
	}

	return count > 0, nil
}

// MarkVersionAsExecuted ...
func (d *PostgresDriver) MarkVersionAsExecuted(tx *sql.Tx, version string) error {
	_, err := tx.Exec("INSERT INTO database_versions (version) VALUES ($1)", version)
	return err
}

// UnmarkVersionAsExecuted ...
func (d *PostgresDriver) UnmarkVersionAsExecuted(tx *sql.Tx, version string) error {
	_, err := transaction.Exec("DELETE FROM database_versions WHERE version=$1", version)
	return err
}
