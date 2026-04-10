CREATE TABLE payments (
                          id TEXT PRIMARY KEY,
                          order_id TEXT,
                          transaction_id TEXT,
                          amount BIGINT,
                          status TEXT
);