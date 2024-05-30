package payperiods

import (
	"context"
	"database/sql"
	"time"

	"github.com/git-masi/paynext/internal/.gen/model"

	// TODO add to whitelist
	. "github.com/git-masi/paynext/internal/.gen/table"
	jet "github.com/go-jet/jet/v2/sqlite"
)

func GetCurrentPayPeriod(db *sql.DB) (model.PayPeriods, error) {
	stmt := PayPeriods.SELECT(PayPeriods.AllColumns).
		WHERE(PayPeriods.Status.EQ(jet.String(Edit.String())))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var dest model.PayPeriods

	err := stmt.QueryContext(ctx, db, &dest)
	if err != nil {
		return model.PayPeriods{}, err
	}

	return dest, err
}
