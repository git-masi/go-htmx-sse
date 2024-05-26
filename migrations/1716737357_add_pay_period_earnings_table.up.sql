-- Migration for: add_pay_period_earnings_table (UP)
CREATE TABLE pay_period_earnings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pay_period_id INTEGER NOT NULL,
    earning_id INTEGER NOT NULL,
    FOREIGN KEY (pay_period_id) REFERENCES pay_period(id),
    FOREIGN KEY (earning_id) REFERENCES earning(id)
);