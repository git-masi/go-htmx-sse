package workers

import (
	"context"
	"database/sql"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/git-masi/paynext/cmd/internal-server/events"
	// TODO add to whitelist
	. "github.com/git-masi/paynext/internal/.gen/table"
	jetsqlite "github.com/go-jet/jet/v2/sqlite"
)

type SyncConfig struct {
	DB             *sql.DB
	PubSub         *events.PubSub[PubSubEvent]
	Logger         *slog.Logger
	MaxConcurrency int
}

// Imagine this is syncing with some external API
func syncWorkers(cfg SyncConfig) {
	ch := cfg.PubSub.Subscribe(Created.String())
	defer close(ch)

	for range cfg.MaxConcurrency {
		go func() {
			for w := range ch {
				time.Sleep(time.Duration(rand.IntN(4000)) * time.Millisecond)
				cfg.Logger.Info("setting worker to active", "id", w.WorkerID)

				stmt := Workers.UPDATE(Workers.Status).
					SET(Active.String()).
					WHERE(Workers.ID.EQ(jetsqlite.Int(w.WorkerID)))

				ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)

				_, err := stmt.ExecContext(ctx, cfg.DB)
				if err != nil {
					// TODO: need better error handling
					cfg.Logger.Error("cannot set worker to active", "id", w.WorkerID)
				}

				cfg.PubSub.Publish(Active.String(), PubSubEvent{WorkerID: w.WorkerID})

				cancel()
			}
		}()
	}
}
