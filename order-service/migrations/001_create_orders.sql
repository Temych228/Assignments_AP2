CREATE TABLE IF NOT EXISTS orders (
                        id TEXT PRIMARY KEY,
                        customer_id TEXT NOT NULL,
                        customer_email TEXT NOT NULL DEFAULT '',
                        item_name TEXT NOT NULL,
                        amount BIGINT NOT NULL,
                        status TEXT NOT NULL,
                        created_at TIMESTAMP NOT NULL,
                        idempotency_key TEXT UNIQUE
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_email TEXT NOT NULL DEFAULT '';

CREATE OR REPLACE FUNCTION notify_order_status_update()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' OR NEW.status IS DISTINCT FROM OLD.status THEN
        PERFORM pg_notify(
            'order_status_updates',
            json_build_object(
                'OrderID', NEW.id,
                'Status', NEW.status,
                'UpdatedAt', clock_timestamp()
            )::text
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_status_notify ON orders;

CREATE TRIGGER orders_status_notify
AFTER INSERT OR UPDATE OF status ON orders
FOR EACH ROW
EXECUTE FUNCTION notify_order_status_update();
