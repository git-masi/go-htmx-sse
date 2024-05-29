package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/git-masi/paynext/cmd/internal-server/domains/earnings"
	payperiods "github.com/git-masi/paynext/cmd/internal-server/domains/pay-periods"
	"github.com/git-masi/paynext/cmd/internal-server/domains/workers"
	"github.com/git-masi/paynext/cmd/internal-server/features"
	"github.com/git-masi/paynext/internal/.gen/model"
	"github.com/git-masi/paynext/internal/.gen/table"
	"github.com/git-masi/paynext/internal/sqlitedb"
	"github.com/git-masi/paynext/internal/utils"
	jetsqlite "github.com/go-jet/jet/v2/sqlite"
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

	exists, err := RowExists(db, table.PayPeriods.TableName(), 1)
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

	mux := http.NewServeMux()

	workersRouter := workers.NewRouter(workers.Config{DB: db, PubSub: wps, Logger: logger})

	workersRouter.Handle("/workers/", http.StripPrefix("/v1", mux))

	fs := http.FileServer(http.Dir("./cmd/internal-server/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		features.Home().Render(r.Context(), w)
	})

	mux.HandleFunc("POST /earnings", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			logger.Error("cannot parse form", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Parsing IDs as int64 so they can be used with jet in the future
		workerID, err := strconv.ParseInt(r.PostForm.Get("worker_id"), 10, 64)
		if err != nil {
			logger.Error("invalid worker ID", "id", r.PostForm.Get("worker_id"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		exists, err := RowExists(db, table.Workers.TableName(), workerID)
		if err != nil {
			logger.Error("error querying worker ID", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !exists {
			logger.Error("cannot find worker matching ID", "id", workerID)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		payPeriodID, err := strconv.ParseInt(r.PostForm.Get("pay_period_id"), 10, 64)
		if err != nil {
			logger.Error("invalid worker ID", "id", r.PostForm.Get("pay_period_id"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		payPeriodStmt := table.PayPeriods.SELECT(table.PayPeriods.StartDate, table.PayPeriods.EndDate).
			WHERE(table.PayPeriods.ID.EQ(jetsqlite.Int(payPeriodID)))

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var payPeriod model.PayPeriods

		err = payPeriodStmt.QueryContext(ctx, db, &payPeriod)
		if err != nil {
			logger.Error("error querying pay period ID", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		sd, err := utils.ParseDBDate(payPeriod.StartDate)
		if err != nil {
			logger.Error("cannot parse pay period start date", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		ed, err := utils.ParseDBDate(payPeriod.EndDate)
		if err != nil {
			logger.Error("cannot parse pay period end date", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		dateOfWork := gofakeit.DateRange(sd, ed)
		payRate := money.New(1+rand.Int64N(10), money.USD)

		earningStmt := table.Earnings.INSERT(
			table.Earnings.DateOfWork,
			table.Earnings.HoursWorked,
			table.Earnings.PayRateAmount,
			table.Earnings.PayRateCurrency,
			table.Earnings.Status,
			table.Earnings.WorkerID,
		).
			VALUES(
				dateOfWork,
				rand.IntN(8)+1,
				payRate.Amount(),
				payRate.Currency(),
				earnings.Pending,
				workerID,
			)

		ctx, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		res, err := earningStmt.ExecContext(ctx, db)
		if err != nil {
			logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		earningID, err := res.LastInsertId()
		if err != nil {
			logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		logger.Info("new earning ID", "id", earningID)

		// TODO: make this part of a transaction
		peStmt := table.PayPeriodEarnings.INSERT(table.PayPeriodEarnings.PayPeriodID, table.PayPeriodEarnings.EarningID).
			VALUES(payPeriodID, earningID)

		ctx, cancel3 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel3()

		_, err = peStmt.ExecContext(ctx, db)
		if err != nil {
			logger.Error("sql exec err", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		eps.Publish(earnings.Created.String(), earnings.PubSubEvent{EarningID: earningID})

		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("GET /earnings/sse/created", func(w http.ResponseWriter, r *http.Request) {
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

		topic := earnings.Created.String()
		ch := eps.Subscribe(topic)
		defer eps.Unsubscribe(topic, ch)

		for {
			select {
			case e := <-ch:
				stmt := table.Earnings.SELECT(table.Earnings.AllColumns).
					WHERE(table.Earnings.ID.EQ(jetsqlite.Int(e.EarningID)))

				logger.Info(stmt.DebugSql())

				var dest model.Earnings

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

				err := stmt.QueryContext(ctx, db, &dest)
				if err != nil {
					logger.Error("cannot query earning", "error", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					cancel()
					return
				}

				w.Write(earnings.EarningCreated(r.Context(), dest).Bytes())
				flusher.Flush()
				cancel()
			case <-time.After(5 * time.Second):
				w.Write([]byte(":ping\n"))
				flusher.Flush()
			}
		}
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

func RowExists(db *sql.DB, tableName string, id int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// It is possible to do this using jet's `RawStatement` or `EXISTS` but it is not clear how
	// Generally it is a bad idea to use `fmt.Sprintf` due to the risk of SQL injection but
	// sqlite doesn't support table names as parameters
	res := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id = ?);`, tableName), id)

	var exists int64

	err := res.Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}
