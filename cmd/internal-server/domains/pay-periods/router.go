package payperiods

import (
	"context"
	"database/sql"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	// TODO add to whitelist
	"github.com/git-masi/paynext/cmd/internal-server/domains/earnings"
	"github.com/git-masi/paynext/internal/.gen/model"
	. "github.com/git-masi/paynext/internal/.gen/table"
	"github.com/git-masi/paynext/internal/utils"
	jet "github.com/go-jet/jet/v2/sqlite"
)

type RouterConfig struct {
	DB *sql.DB
	// PubSub *events.PubSub[PubSubEvent]
	Logger *slog.Logger
}

func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /submit/{id}", submitPayPeriod(cfg))

	return mux
}

func submitPayPeriod(cfg RouterConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			cfg.Logger.Error("invalid pay period ID", "id", r.PathValue("id"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		pp, err := getPayPeriod(cfg.DB, id)
		if err != nil {
			cfg.Logger.Error("cannot get pay period", "id", id)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		stmt := PayPeriods.UPDATE(PayPeriods.Status).
			SET(Pending.String()).
			WHERE(PayPeriods.ID.EQ(jet.Int(id)))

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_, err = stmt.ExecContext(ctx, cfg.DB)
		if err != nil {
			cfg.Logger.Error("cannot update pay period", "id", id)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		prevEnd, err := utils.ParseDBDate(pp.EndDate)
		if err != nil {
			cfg.Logger.Error("cannot parse date string", "date", pp.EndDate)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = newPayPeriod(cfg.DB, prevEnd.Add(24*time.Hour))
		if err != nil {
			cfg.Logger.Error("cannot init new pay period")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		go func() {
			ppe, err := GetPayPeriodEarnings(cfg.DB, id)
			if err != nil {
				cfg.Logger.Error("cannot get pay period earnings", "id", id)
				return
			}

			for _, e := range ppe {
				time.Sleep(time.Duration(rand.IntN(2000)) * time.Millisecond)

				stmt := Earnings.UPDATE(Earnings.Status).SET(earnings.Active).WHERE(Earnings.ID.EQ(jet.Int(int64(e.EarningID))))

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

				_, err = stmt.ExecContext(ctx, cfg.DB)
				if err != nil {
					cfg.Logger.Error("cannot update earning status", "id", e.EarningID)
					cancel()
					return
				}

				cancel()
				cfg.Logger.Info("updated earning status", "id", e.EarningID)
			}
		}()

		http.Redirect(w, r, "/home", http.StatusSeeOther)
	}
}

func getPayPeriod(db *sql.DB, id int64) (model.PayPeriods, error) {
	stmt := PayPeriods.SELECT(PayPeriods.AllColumns).
		WHERE(PayPeriods.ID.EQ(jet.Int(id)))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var dest model.PayPeriods

	err := stmt.QueryContext(ctx, db, &dest)
	if err != nil {
		return model.PayPeriods{}, err
	}

	return dest, nil
}

func newPayPeriod(db *sql.DB, day time.Time) error {
	startDate, endDate := utils.GetWeekStartEnd(day)
	stmt := PayPeriods.INSERT(PayPeriods.StartDate, PayPeriods.EndDate, PayPeriods.Status).
		VALUES(startDate, endDate, Edit.String())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := stmt.ExecContext(ctx, db)
	if err != nil {
		return err
	}

	return nil
}
