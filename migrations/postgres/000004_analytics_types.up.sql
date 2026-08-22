-- Validate the textual API values before changing their physical types. Optional
-- fields may use an empty string to mean "not set"; required API fields may not.
DO $$
DECLARE
    invalid_column TEXT;
    invalid_value  TEXT;
BEGIN
    SELECT source_column, source_value
    INTO invalid_column, invalid_value
    FROM (
        SELECT 'account.start_date' AS source_column, start_date AS source_value
        FROM account
        WHERE start_date IS NOT NULL
          AND BTRIM(start_date) <> ''
          AND start_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
        UNION ALL
        SELECT 'reminder.end_date', end_date
        FROM reminder
        WHERE end_date IS NOT NULL
          AND BTRIM(end_date) <> ''
          AND end_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
        UNION ALL
        SELECT 'budget.date', date
        FROM budget
        WHERE date IS NOT NULL
          AND date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
        UNION ALL
        SELECT 'reminder.start_date', start_date
        FROM reminder
        WHERE start_date IS NOT NULL
          AND start_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
        UNION ALL
        SELECT 'reminder_marker.date', date
        FROM reminder_marker
        WHERE date IS NOT NULL
          AND date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
        UNION ALL
        SELECT 'transaction.date', date
        FROM transaction
        WHERE date IS NOT NULL
          AND date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    ) invalid_dates
    LIMIT 1;

    IF invalid_column IS NOT NULL THEN
        RAISE EXCEPTION 'cannot migrate %. Invalid ISO date: %', invalid_column, invalid_value;
    END IF;
END
$$;

DO $$
DECLARE
    invalid_column TEXT;
    invalid_value  TEXT;
BEGIN
    SELECT source_column, source_value
    INTO invalid_column, invalid_value
    FROM (
        SELECT 'tag.parent' AS source_column, parent AS source_value
        FROM tag
        WHERE parent IS NOT NULL
          AND BTRIM(parent) <> ''
          AND parent !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'transaction.outcome_account', outcome_account
        FROM transaction
        WHERE outcome_account IS NOT NULL
          AND BTRIM(outcome_account) <> ''
          AND outcome_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'reminder.income_account', income_account
        FROM reminder
        WHERE income_account IS NOT NULL
          AND income_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'reminder.outcome_account', outcome_account
        FROM reminder
        WHERE outcome_account IS NOT NULL
          AND outcome_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'reminder_marker.income_account', income_account
        FROM reminder_marker
        WHERE income_account IS NOT NULL
          AND income_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'reminder_marker.outcome_account', outcome_account
        FROM reminder_marker
        WHERE outcome_account IS NOT NULL
          AND outcome_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        UNION ALL
        SELECT 'transaction.income_account', income_account
        FROM transaction
        WHERE income_account IS NOT NULL
          AND income_account !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    ) invalid_uuids
    LIMIT 1;

    IF invalid_column IS NOT NULL THEN
        RAISE EXCEPTION 'cannot migrate %. Invalid UUID: %', invalid_column, invalid_value;
    END IF;
END
$$;

ALTER TABLE instrument
    ALTER COLUMN rate TYPE NUMERIC USING rate::NUMERIC;

ALTER TABLE account
    ALTER COLUMN balance TYPE NUMERIC USING balance::NUMERIC,
    ALTER COLUMN start_balance TYPE NUMERIC USING start_balance::NUMERIC,
    ALTER COLUMN credit_limit TYPE NUMERIC USING credit_limit::NUMERIC,
    ALTER COLUMN percent TYPE NUMERIC USING percent::NUMERIC,
    ALTER COLUMN start_date TYPE DATE
        USING NULLIF(BTRIM(start_date), '')::DATE;

ALTER TABLE tag
    ALTER COLUMN parent TYPE UUID
        USING NULLIF(BTRIM(parent), '')::UUID;

ALTER TABLE reminder
    ALTER COLUMN income TYPE NUMERIC USING income::NUMERIC,
    ALTER COLUMN outcome TYPE NUMERIC USING outcome::NUMERIC,
    ALTER COLUMN start_date TYPE DATE USING start_date::DATE,
    ALTER COLUMN end_date TYPE DATE
        USING NULLIF(BTRIM(end_date), '')::DATE,
    ALTER COLUMN income_account TYPE UUID USING income_account::UUID,
    ALTER COLUMN outcome_account TYPE UUID USING outcome_account::UUID;

ALTER TABLE reminder_marker
    ALTER COLUMN income TYPE NUMERIC USING income::NUMERIC,
    ALTER COLUMN outcome TYPE NUMERIC USING outcome::NUMERIC,
    ALTER COLUMN date TYPE DATE USING date::DATE,
    ALTER COLUMN income_account TYPE UUID USING income_account::UUID,
    ALTER COLUMN outcome_account TYPE UUID USING outcome_account::UUID;

ALTER TABLE transaction
    ALTER COLUMN date TYPE DATE USING date::DATE,
    ALTER COLUMN income TYPE NUMERIC USING income::NUMERIC,
    ALTER COLUMN outcome TYPE NUMERIC USING outcome::NUMERIC,
    ALTER COLUMN op_income TYPE NUMERIC USING op_income::NUMERIC,
    ALTER COLUMN op_outcome TYPE NUMERIC USING op_outcome::NUMERIC,
    ALTER COLUMN income_account TYPE UUID USING income_account::UUID,
    ALTER COLUMN outcome_account TYPE UUID
        USING NULLIF(BTRIM(outcome_account), '')::UUID;

ALTER TABLE budget
    ALTER COLUMN date TYPE DATE USING date::DATE,
    ALTER COLUMN income TYPE NUMERIC USING income::NUMERIC,
    ALTER COLUMN outcome TYPE NUMERIC USING outcome::NUMERIC;

CREATE INDEX IF NOT EXISTS idx_transaction_user_date_created
    ON transaction ("user", date DESC, created DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_income_account_date
    ON transaction (income_account, date DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_outcome_account_date
    ON transaction (outcome_account, date DESC)
    WHERE outcome_account IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transaction_merchant_date
    ON transaction (merchant, date DESC)
    WHERE merchant IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transaction_income_instrument_date
    ON transaction (income_instrument, date DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_outcome_instrument_date
    ON transaction (outcome_instrument, date DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_tags_gin
    ON transaction USING GIN (tag);
