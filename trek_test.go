package trek_test

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tombell/trek"
)

var currentMigrationTime = time.Now()

func teardown() {
	os.Remove("./db.sqlite")
	os.RemoveAll("./migrations")
}

func openDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func createMigration(t *testing.T, migrationName string, sql string) {
	t.Helper()

	if err := os.MkdirAll("migrations", 0700); err != nil {
		t.Fatal(err)
	}

	sql = strings.TrimSpace(sql)

	currentMigrationTime = currentMigrationTime.Add(1 * time.Second)

	migrationTime := currentMigrationTime.Format("20060102150405")
	migrationName = migrationTime + "_" + migrationName + ".sql"

	filename := path.Join("migrations", migrationName)

	if err := ioutil.WriteFile(filename, []byte(sql), 0700); err != nil {
		t.Fatal(err)
	}
}

func migrateMigrations(t *testing.T) {
	t.Helper()

	if err := trek.Migrate(nil, "sqlite3", "db.sqlite", "migrations"); err != nil {
		t.Fatal(err)
	}
}

func rollbackMigrations(t *testing.T, steps int) {
	t.Helper()

	if err := trek.Rollback(nil, "sqlite3", "db.sqlite", "migrations", steps); err != nil {
		t.Fatal(err)
	}
}

func assertValueOfUsername(t *testing.T, expected string) {
	db := openDatabase(t)
	defer db.Close()

	var username string

	row := db.QueryRow("SELECT username FROM users LIMIT 1")
	if err := row.Scan(&username); err != nil {
		t.Fatal(err)
	}

	if username != expected {
		t.Fatalf("expected username to be %v, but got %v", expected, username)
	}
}

func TestMigrationsMigrateInOrder(t *testing.T) {
	defer teardown()

	createMigration(t, "create_users", "CREATE TABLE users(username TEXT);")
	createMigration(t, "populate_users", "INSERT INTO users (username) VALUES ('tombell:');")

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf("UPDATE users SET username=username || '%v';", i))
	}

	migrateMigrations(t)
	assertValueOfUsername(t, "tombell:01234")
}

func TestMigrationsDontMigrateTwice(t *testing.T) {
	defer teardown()

	db := openDatabase(t)
	defer db.Close()

	if _, err := db.Exec(`
		CREATE table users(username TEXT);
		INSERT INTO users (username) VALUES ('tombell:');
	`); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf("UPDATE users SET username=username || '%v';", i))
	}

	migrateMigrations(t)
	assertValueOfUsername(t, "tombell:01234")

	migrateMigrations(t)
	assertValueOfUsername(t, "tombell:01234")
}

func TestMigrationsRollbackInOrder(t *testing.T) {
	defer teardown()

	db := openDatabase(t)
	defer db.Close()

	if _, err := db.Exec(`
		CREATE table users(username TEXT);
		INSERT INTO users (username) VALUES ('tombell:');
	`); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf(
			`
-- UP
UPDATE users SET username='up:%v';

-- DOWN
UPDATE users SET username='down:%v';`,
			i, i,
		))
	}

	assertValueOfUsername(t, "tombell:")

	migrateMigrations(t)
	assertValueOfUsername(t, "up:4")

	rollbackMigrations(t, -1)
	assertValueOfUsername(t, "down:0")
}

func TestMigrationsRollbackInSteps(t *testing.T) {
	defer teardown()

	db := openDatabase(t)
	defer db.Close()

	if _, err := db.Exec(`
		CREATE table users(username TEXT);
		INSERT INTO users (username) VALUES ('tombell:');
	`); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf(
			`
-- UP
UPDATE users SET username='up:%v';

-- DOWN
UPDATE users SET username='down:%v';`,
			i, i,
		))
	}

	assertValueOfUsername(t, "tombell:")

	migrateMigrations(t)
	assertValueOfUsername(t, "up:4")

	rollbackMigrations(t, 2)
	assertValueOfUsername(t, "down:3")
}
