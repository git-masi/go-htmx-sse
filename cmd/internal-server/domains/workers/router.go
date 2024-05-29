package workers

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	payperiods "github.com/git-masi/paynext/cmd/internal-server/domains/pay-periods"
	"github.com/git-masi/paynext/cmd/internal-server/events"
	"github.com/git-masi/paynext/internal/.gen/model"

	// TODO add to whitelist
	. "github.com/git-masi/paynext/internal/.gen/table"
	jetsqlite "github.com/go-jet/jet/v2/sqlite"
)

type Config struct {
	DB     *sql.DB
	PubSub *events.PubSub[PubSubEvent]
	Logger *slog.Logger
}

func NewRouter(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /sse/created", emitWorkerCreated(cfg))

	mux.HandleFunc("POST /create", addWorker(cfg))

	return mux
}

func emitWorkerCreated(cfg Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

		topic := Created.String()
		ch := cfg.PubSub.Subscribe(topic)
		defer cfg.PubSub.Unsubscribe(topic, ch)

		for {
			select {
			case e := <-ch:
				stmt := Workers.SELECT(Workers.AllColumns).
					WHERE(Workers.ID.EQ(jetsqlite.Int(e.WorkerID)))

				cfg.Logger.Info(stmt.DebugSql())

				var dest model.Workers

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

				err := stmt.QueryContext(ctx, cfg.DB, &dest)
				if err != nil {
					cfg.Logger.Error("cannot query worker", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					cancel()
					return
				}

				payPeriodID, err := payperiods.GetCurrentPayPeriod(cfg.DB)
				if err != nil {
					cfg.Logger.Error("cannot get pay period ID", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					cancel()
					return
				}

				w.Write(WorkerCreated(r.Context(), dest, payPeriodID).Bytes())
				flusher.Flush()
				cancel()
			case <-time.After(5 * time.Second):
				w.Write([]byte(":ping\n"))
				flusher.Flush()
			}
		}
	}
}

func addWorker(cfg Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		stmt := Workers.INSERT(
			Workers.FirstName,
			Workers.LastName,
			Workers.Email,
			Workers.Status,
		).VALUES(
			gofakeit.FirstName(),
			gofakeit.LastName(),
			gofakeit.Email(),
			Pending.String(),
		)

		cfg.Logger.Info(stmt.DebugSql())

		res, err := stmt.ExecContext(r.Context(), cfg.DB)
		if err != nil {
			cfg.Logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		id, err := res.LastInsertId()
		if err != nil {
			cfg.Logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		cfg.Logger.Info("new worker id", "id", id)

		cfg.PubSub.Publish(Created.String(), PubSubEvent{WorkerID: id})
	}
}
