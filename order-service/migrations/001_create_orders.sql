CREATE TABLE orders (
                        id TEXT PRIMARY KEY,
                        customer_id TEXT NOT NULL,
                        item_name TEXT NOT NULL,
                        amount BIGINT NOT NULL,
                        status TEXT NOT NULL,
                        created_at TIMESTAMP NOT NULL,
                        idempotency_key TEXT UNIQUE
);