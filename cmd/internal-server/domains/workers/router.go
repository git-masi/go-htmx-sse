package workers

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"math/rand/v2"
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

type RouterConfig struct {
	DB     *sql.DB
	PubSub *events.PubSub[PubSubEvent]
	Logger *slog.Logger
}

func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /create", createWorker(cfg))

	mux.HandleFunc("GET /sse/created", sseHandler(cfg))

	return mux
}

func createWorker(cfg RouterConfig) func(http.ResponseWriter, *http.Request) {
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

		// cfg.Logger.Info(stmt.DebugSql())

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

		cfg.PubSub.Publish(Topic, PubSubEvent{WorkerID: id, Event: Created})

		go func() {
			time.Sleep(time.Duration(math.Max(1000, float64(rand.IntN(4000)))) * time.Millisecond)
			cfg.Logger.Info("setting worker to active", "id", id)

			stmt := Workers.UPDATE(Workers.Status).
				SET(Active.String()).
				WHERE(Workers.ID.EQ(jetsqlite.Int(id)))

			ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)

			_, err := stmt.ExecContext(ctx, cfg.DB)
			if err != nil {
				// TODO: need better error handling
				cfg.Logger.Error("cannot set worker to active", "id", id)
				cancel()
				return
			}

			cfg.PubSub.Publish(Topic, PubSubEvent{WorkerID: id, Event: Updated})

			cancel()
		}()
	}
}

func sseHandler(cfg RouterConfig) func(http.ResponseWriter, *http.Request) {
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

		ch := cfg.PubSub.Subscribe(Topic)
		defer cfg.PubSub.Unsubscribe(Topic, ch)

		for {
			select {
			case e := <-ch:
				stmt := Workers.SELECT(Workers.AllColumns).
					WHERE(Workers.ID.EQ(jetsqlite.Int(e.WorkerID)))

				// cfg.Logger.Info(stmt.DebugSql())

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
