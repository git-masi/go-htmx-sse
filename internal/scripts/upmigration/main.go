package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/git-masi/paynext/internal/sqlitedb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	dsn  string
	path string
}

func main() {
	var cfg config

	flag.StringVar(&cfg.dsn, "dsn", "", "A data source name (DSN) for the database")
	flag.StringVar(&cfg.path, "path", "", "The path to the migrations files")
	flag.Parse()

	if cfg.dsn == "" {
		log.Fatal("missing DSN")
	}

	if cfg.path == "" {
		log.Fatal("missing path to migration files")
	}

	db, err := sqlitedb.OpenDB(cfg.dsn)
	if err != nil {
		log.Fatalf("cannot open DB, %v\n", err)
	}
	defer db.Close()

	err = sqlitedb.RunMigrations(db, url.URL{
		Scheme: "file",
		Path:   cfg.path,
	})
	if err != nil {
		log.Fatalf("cannot run migrations, %v\n", err)
	}

	log.Println("migration success!")
}
