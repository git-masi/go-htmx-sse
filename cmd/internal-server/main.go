package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/git-masi/paynext/cmd/internal-server/domains/workers"
	"github.com/git-masi/paynext/cmd/internal-server/features"
	"github.com/git-masi/paynext/internal/.gen/model"
	"github.com/git-masi/paynext/internal/.gen/table"
	"github.com/git-masi/paynext/internal/sqlitedb"
	jetsqlite "github.com/go-jet/jet/v2/sqlite"
)

type config struct {
	dsn string
}

func main() {
	var cfg config

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	wps := workers.NewWorkerPubSub()

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

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		features.Home().Render(r.Context(), w)
	})

	mux.HandleFunc("GET /workers/sse/created", func(w http.ResponseWriter, r *http.Request) {
		defer r.Context().Done()

		// Set the headers for Server-Sent Events
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Disable chunked transfer encoding to prevent ERR_INCOMPLETE_CHUNKED_ENCODING on the client
		w.Header().Set("Transfer-Encoding", "identity")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		topic := workers.Created.String()
		ch := wps.Subscribe(topic)
		defer wps.Unsubscribe(topic, ch)

		for {
			select {
			case e := <-ch:
				stmt := table.Workers.SELECT(table.Workers.AllColumns).
					WHERE(table.Workers.ID.EQ(jetsqlite.Int(e.WorkerID)))

				logger.Info(stmt.DebugSql())

				var dest []model.Workers

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				err := stmt.QueryContext(ctx, db, &dest)
				if err != nil {
					logger.Error("cannot query worker", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					cancel()
					return
				}

				w.Write(workers.WorkerCreated(r.Context(), dest[0]).Bytes())
				flusher.Flush()
				cancel()
			case <-time.After(5 * time.Second):
				w.Write([]byte(":ping"))
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("POST /workers", func(w http.ResponseWriter, r *http.Request) {
		stmt := table.Workers.INSERT(
			table.Workers.FirstName,
			table.Workers.LastName,
			table.Workers.Email,
		).VALUES(
			gofakeit.FirstName(),
			gofakeit.LastName(),
			gofakeit.Email(),
		)

		logger.Info(stmt.DebugSql())

		res, err := stmt.ExecContext(r.Context(), db)
		if err != nil {
			logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := res.LastInsertId()
		if err != nil {
			logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		logger.Info("new worker id", "id", id)

		wps.Publish(workers.Created.String(), workers.PubSubEvent{WorkerID: id})
	})

	server := http.Server{
		// TODO: make this an arg
		Addr:    ":8080",
		Handler: mux,
		// Use route level timeouts
		// ReadTimeout:  1 * time.Second,
		// WriteTimeout: 10 * time.Second,
	}

	logger.Info("server starting on port 8080")

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server err", "error", err)
		os.Exit(1)
	}
}

type Worker struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type Earning struct {
	WorkerID    int         `json:"worker_id"`
	DateOfWork  time.Time   `json:"date_of_work"`
	HoursWorked float32     `json:"hours_worked"`
	PayRate     money.Money `json:"pay_rate"`
	Status      string      `json:"status"`
}

type PayPeriod struct {
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	WorkerEarnings []Earning `json:"worker_earnings"`
	Status         string    `json:"status"`
}
