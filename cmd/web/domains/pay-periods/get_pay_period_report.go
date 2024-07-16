package payperiods

import (
	"context"
	"database/sql"
	"time"
)

type WorkerGrossEarnings struct {
	WorkerID         *int32
	FirstName        string
	LastName         string
	PayPeriodID      *int32
	TotalHoursWorked float32
	GrossPay         float32
}

func GetPayPeriodReport(db *sql.DB, id int64) ([]*WorkerGrossEarnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Could do this using jet but I'm lazy
	stmt := `SELECT
				pp.id AS pay_period_id,
				w.id AS worker_id,
				w.first_name,
				w.last_name,
				SUM(e.hours_worked) AS total_hours_worked,
				SUM(e.hours_worked * e.pay_rate_amount) AS gross_pay
			FROM 
				workers w
				JOIN earnings e ON w.id = e.worker_id
				JOIN pay_period_earnings ppe ON e.id = ppe.earning_id
				JOIN pay_periods pp ON ppe.pay_period_id = pp.id
			WHERE 
				pp.id = ?
			GROUP BY 
				pp.id, w.id, w.first_name, w.last_name;`

	rows, err := db.QueryContext(ctx, stmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	earnings := []*WorkerGrossEarnings{}

	for rows.Next() {
		we := new(WorkerGrossEarnings)

		err = rows.Scan(&we.PayPeriodID, &we.WorkerID, &we.FirstName, &we.LastName, &we.TotalHoursWorked, &we.GrossPay)
		if err != nil {
			return nil, err
		}

		earnings = append(earnings, we)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return earnings, nil
}
