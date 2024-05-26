-- Migration for: add_pay_period_table (UP)
CREATE TABLE pay_period (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    status TEXT NOT NULL
);