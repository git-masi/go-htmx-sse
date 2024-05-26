-- Migration for: add_worker_earnings_table (UP)
CREATE TABLE earnings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date_of_work TEXT NOT NULL,
    hours_worked REAL NOT NULL,
    pay_rate_amount INTEGER NOT NULL,
    pay_rate_currency TEXT NOT NULL,
    status TEXT NOT NULL,
    worker_id INTEGER NOT NULL,
    FOREIGN KEY (worker_id) REFERENCES workers(id)
);
CREATE INDEX earnings_worker_id_idx ON earnings (worker_id);