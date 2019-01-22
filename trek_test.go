package trek_test

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
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

func applyMigrations(t *testing.T) {
	t.Helper()

	logger := log.New(os.Stderr, "[trek-test] ", log.LstdFlags)

	if err := trek.Apply(logger, "sqlite3", "db.sqlite", "migrations"); err != nil {
		t.Fatal(err)
	}
}

func rollbackMigrations(t *testing.T) {
	t.Helper()

	logger := log.New(os.Stderr, "[trek-test] ", log.LstdFlags)

	if err := trek.Rollback(logger, "sqlite3", "db.sqlite", "migrations"); err != nil {
		t.Fatal(err)
	}
}

func assertValueOfUsername(t *testing.T, expected string) {
	db, err := sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var username string

	row := db.QueryRow("SELECT username FROM users LIMIT 1")
	if err := row.Scan(&username); err != nil {
		t.Fatal(err)
	}

	if username != expected {
		t.Errorf("expected username to be %v, but got %v", expected, username)
	}
}

func TestMigrationsApplyInOrder(t *testing.T) {
	defer teardown()

	createMigration(t, "create_users", "CREATE TABLE users(username TEXT);")
	createMigration(t, "populate_users", "INSERT INTO users (username) VALUES ('tombell:');")

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf("UPDATE users SET username=username || '%v';", i))
	}

	applyMigrations(t)

	assertValueOfUsername(t, "tombell:01234")
}

func TestMigrationsDontApplyTwice(t *testing.T) {
	defer teardown()

	db, err := sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE table users(username TEXT);
		INSERT INTO users (username) VALUES ('tombell:');
	`)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf("UPDATE users SET username=username || '%v';", i))
	}

	applyMigrations(t)
	applyMigrations(t)

	assertValueOfUsername(t, "tombell:01234")
}

func TestMigrationsCanRollback(t *testing.T) {
	defer teardown()

	db, err := sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE table users(username TEXT);
		INSERT INTO users (username) VALUES ('tombell:');
	`)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		createMigration(t, "concat_values", fmt.Sprintf(
			`
-- up
UPDATE users SET username='%v';
-- down
UPDATE users SET username='%v';
			`,
			i, 4-i,
		))
	}

	applyMigrations(t)
	rollbackMigrations(t)

	assertValueOfUsername(t, "4")
}
