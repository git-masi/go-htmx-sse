package payperiods

import (
	"context"
	"database/sql"
	"time"

	"github.com/git-masi/paynext/internal/.gen/model"

	// TODO add to whitelist
	. "github.com/git-masi/paynext/internal/.gen/table"
	jetsqlite "github.com/go-jet/jet/v2/sqlite"
)

func GetCurrentPayPeriod(db *sql.DB) (int64, error) {
	stmt := PayPeriods.SELECT(PayPeriods.ID).
		WHERE(PayPeriods.Status.EQ(jetsqlite.String(Edit.String())))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var dest model.PayPeriods

	err := stmt.QueryContext(ctx, db, &dest)
	if err != nil {
		return 0, err
	}

	return int64(*dest.ID), err
}
