package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/git-masi/paynext/cmd/internal-server/domains/earnings"
	payperiods "github.com/git-masi/paynext/cmd/internal-server/domains/pay-periods"
	"github.com/git-masi/paynext/cmd/internal-server/domains/workers"
	"github.com/git-masi/paynext/cmd/internal-server/features"
	"github.com/git-masi/paynext/internal/.gen/table"
	"github.com/git-masi/paynext/internal/sqlitedb"
	"github.com/git-masi/paynext/internal/utils"
)

type config struct {
	dsn string
}

func main() {
	var cfg config

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	wps := workers.NewWorkerPubSub()
	eps := earnings.NewEarningPubSub()

	flag.StringVar(&cfg.dsn, "dsn", "", "A data source name (DSN) for the database")
	flag.Parse()

	if cfg.dsn == "" {
		logger.Error("missing dsn")
		os.Exit(1)
	}

	db, err := sqlitedb.OpenDB(cfg.dsn)
	if err != nil {
		logger.Error("cannot open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	initPayPeriod(db, logger)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./cmd/internal-server/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		features.Home().Render(r.Context(), w)
	})

	workersRouter := workers.NewRouter(workers.Config{DB: db, PubSub: wps, Logger: logger})
	workersRouter.Handle("/workers/", http.StripPrefix("/workers", mux))

	earningsRouter := earnings.NewRouter(earnings.Config{DB: db, PubSub: eps, Logger: logger})
	earningsRouter.Handle("/earnings/", http.StripPrefix("/earnings", mux))

	server := http.Server{
		// TODO: make this an arg
		Addr:    ":8080",
		Handler: mux,
		// Use route level timeouts
	}

	logger.Info("server starting on port 8080")

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server err", "error", err)
		os.Exit(1)
	}
}

func initPayPeriod(db *sql.DB, logger *slog.Logger) {
	exists, err := utils.RowExists(db, table.PayPeriods.TableName(), 1)
	if err != nil {
		logger.Error("cannot query pay period table", "error", err)
		os.Exit(1)
	}
	if !exists {
		startDate, endDate := utils.GetWeekStartEnd(time.Now().UTC())
		stmt := table.PayPeriods.INSERT(table.PayPeriods.StartDate, table.PayPeriods.EndDate, table.PayPeriods.Status).
			VALUES(startDate, endDate, payperiods.Edit.String())

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		_, err = stmt.ExecContext(ctx, db)
		if err != nil {
			logger.Error("cannot insert pay period", "error", err)
			cancel()
			os.Exit(1)
		}

		logger.Info("successfully added a new pay period!")
		cancel()
	}
}
