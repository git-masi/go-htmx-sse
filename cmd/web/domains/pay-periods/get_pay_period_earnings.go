package payperiods

import (
	"context"
	"database/sql"
	"time"

	"github.com/git-masi/go-htmx-sse/internal/.gen/model"
	. "github.com/git-masi/go-htmx-sse/internal/.gen/table"
	jet "github.com/go-jet/jet/v2/sqlite"
)

func GetPayPeriodEarnings(db *sql.DB, id int64) ([]model.PayPeriodEarnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stmt := PayPeriodEarnings.SELECT(PayPeriodEarnings.AllColumns).WHERE(PayPeriodEarnings.PayPeriodID.EQ(jet.Int(id)))

	var dest []model.PayPeriodEarnings

	err := stmt.QueryContext(ctx, db, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}
