package utils

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func ParseDBDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05.999999-07:00", s)
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
