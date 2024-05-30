package earnings

import (
	"context"
	"database/sql"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/git-masi/paynext/cmd/internal-server/events"
	"github.com/git-masi/paynext/internal/utils"

	"github.com/git-masi/paynext/internal/.gen/model"

	// TODO add to whitelist
	. "github.com/git-masi/paynext/internal/.gen/table"
	jet "github.com/go-jet/jet/v2/sqlite"
)

type RouterConfig struct {
	DB     *sql.DB
	PubSub *events.PubSub[PubSubEvent]
	Logger *slog.Logger
}

func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /create", createEarning(cfg))

	mux.HandleFunc("GET /sse/created", emitEarningCreated(cfg))

	return mux
}

func createEarning(cfg RouterConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			cfg.Logger.Error("cannot parse form", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Parsing IDs as int64 so they can be used with jet in the future
		workerID, err := strconv.ParseInt(r.PostForm.Get("worker_id"), 10, 64)
		if err != nil {
			cfg.Logger.Error("invalid worker ID", "id", r.PostForm.Get("worker_id"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		exists, err := utils.RowExists(cfg.DB, Workers.TableName(), workerID)
		if err != nil {
			cfg.Logger.Error("error querying worker ID", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !exists {
			cfg.Logger.Error("cannot find worker matching ID", "id", workerID)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		payPeriodID, err := strconv.ParseInt(r.PostForm.Get("pay_period_id"), 10, 64)
		if err != nil {
			cfg.Logger.Error("invalid worker ID", "id", r.PostForm.Get("pay_period_id"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		payPeriodStmt := PayPeriods.SELECT(PayPeriods.StartDate, PayPeriods.EndDate).
			WHERE(PayPeriods.ID.EQ(jet.Int(payPeriodID)))

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var payPeriod model.PayPeriods

		err = payPeriodStmt.QueryContext(ctx, cfg.DB, &payPeriod)
		if err != nil {
			cfg.Logger.Error("error querying pay period ID", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		sd, err := utils.ParseDBDate(payPeriod.StartDate)
		if err != nil {
			cfg.Logger.Error("cannot parse pay period start date", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		ed, err := utils.ParseDBDate(payPeriod.EndDate)
		if err != nil {
			cfg.Logger.Error("cannot parse pay period end date", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		dateOfWork := gofakeit.DateRange(sd, ed)
		payRate := money.New(1+rand.Int64N(10), money.USD)

		earningStmt := Earnings.INSERT(
			Earnings.DateOfWork,
			Earnings.HoursWorked,
			Earnings.PayRateAmount,
			Earnings.PayRateCurrency,
			Earnings.Status,
			Earnings.WorkerID,
		).
			VALUES(
				dateOfWork,
				rand.IntN(8)+1,
				payRate.Amount(),
				payRate.Currency(),
				Pending.String(),
				workerID,
			)

		ctx, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		res, err := earningStmt.ExecContext(ctx, cfg.DB)
		if err != nil {
			cfg.Logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		earningID, err := res.LastInsertId()
		if err != nil {
			cfg.Logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		cfg.Logger.Info("new earning ID", "id", earningID)

		// TODO: make this part of a transaction
		peStmt := PayPeriodEarnings.INSERT(PayPeriodEarnings.PayPeriodID, PayPeriodEarnings.EarningID).
			VALUES(payPeriodID, earningID)

		ctx, cancel3 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel3()

		_, err = peStmt.ExecContext(ctx, cfg.DB)
		if err != nil {
			cfg.Logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		cfg.PubSub.Publish(Created.String(), PubSubEvent{EarningID: earningID})

		w.WriteHeader(http.StatusAccepted)
	}
}

func emitEarningCreated(cfg RouterConfig) func(w http.ResponseWriter, r *http.Request) {
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
				stmt := Earnings.SELECT(Earnings.AllColumns).
					WHERE(Earnings.ID.EQ(jet.Int(e.EarningID)))

				// cfg.Logger.Info(stmt.DebugSql())

				var dest model.Earnings

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

				err := stmt.QueryContext(ctx, cfg.DB, &dest)
				if err != nil {
					cfg.Logger.Error("cannot query earning", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					cancel()
					return
				}

				w.Write(EarningCreated(r.Context(), dest).Bytes())
				flusher.Flush()
				cancel()
			case <-time.After(5 * time.Second):
				w.Write([]byte(":ping\n"))
				flusher.Flush()
			}
		}
	}
}
