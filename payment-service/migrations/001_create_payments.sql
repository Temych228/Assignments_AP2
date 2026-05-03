CREATE TABLE IF NOT EXISTS payments (
                          id TEXT PRIMARY KEY,
                          order_id TEXT,
                          customer_email TEXT NOT NULL DEFAULT '',
                          transaction_id TEXT,
                          amount BIGINT,
                          status TEXT
);

ALTER TABLE payments ADD COLUMN IF NOT EXISTS customer_email TEXT NOT NULL DEFAULT '';
