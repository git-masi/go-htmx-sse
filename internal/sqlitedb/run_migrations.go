package sqlitedb

import (
	"database/sql"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	msqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func RunMigrations(db *sql.DB, sourceURL url.URL) error {
	driver, err := msqlite3.WithInstance(db, &msqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL.String(), "", driver)
	if err != nil {
		return err
	}

	return m.Up()
}
