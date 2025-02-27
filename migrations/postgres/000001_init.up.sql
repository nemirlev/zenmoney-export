CREATE TABLE IF NOT EXISTS instrument
(
    id          INT,
    changed     INT,
    title       TEXT,
    short_title TEXT,
    symbol      TEXT,
    rate        FLOAT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS company
(
    id           INT,
    changed      INT,
    title        TEXT,
    full_title   TEXT,
    www          TEXT,
    country      INT,
    deleted      BOOLEAN DEFAULT FALSE,
    country_code TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS "user"
(
    id                        BIGINT,
    country                   INT           NULL,
    country_code              TEXT          NULL,
    email                     TEXT          NULL,
    changed                   BIGINT,
    login                     TEXT,
    currency                  INT,
    parent                    INT           NULL,
    paid_till                 BIGINT,
    month_start_day           INT,
    is_forecast_enabled       BOOLEAN,
    plan_balance_mode         TEXT,
    plan_settings             TEXT,
    subscription              TEXT,
    subscription_renewal_date BIGINT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS country
(
    id       INT,
    title    TEXT,
    currency INT,
    domain   TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS account
(
    id                       UUID,
    changed                  INT,
    "user"                   INT,
    role                     INT,
    instrument               INT,
    company                  INT,
    type                     TEXT,
    title                    TEXT,
    sync_id                  TEXT[],
    balance                  FLOAT,
    start_balance            FLOAT,
    credit_limit             FLOAT,
    in_balance               BOOLEAN,
    savings                  BOOLEAN,
    enable_correction        BOOLEAN,
    enable_sms               BOOLEAN,
    archive                  BOOLEAN,
    capitalization           BOOLEAN,
    percent                  FLOAT,
    start_date               TEXT,
    end_date_offset          INT,
    end_date_offset_interval TEXT,
    payoff_step              INT,
    payoff_interval          TEXT,
    private                  BOOLEAN,
    balance_correction_type  TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS tag
(
    id             UUID,
    changed        INT,
    "user"         INT,
    title          TEXT,
    parent         TEXT,
    icon           TEXT,
    picture        TEXT,
    color          BIGINT,
    show_income    BOOLEAN,
    show_outcome   BOOLEAN,
    budget_income  BOOLEAN,
    budget_outcome BOOLEAN,
    required       BOOLEAN,
    static_id      TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS merchant
(
    id      UUID,
    changed INT,
    "user"  INT,
    title   TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS reminder
(
    id                 UUID,
    changed            INT,
    "user"             INT,
    income_instrument  INT,
    income_account     TEXT,
    income             FLOAT,
    outcome_instrument INT,
    outcome_account    TEXT,
    outcome            FLOAT,
    tag                UUID[],
    merchant           UUID,
    payee              TEXT,
    comment            TEXT,
    interval           TEXT,
    step               INT,
    points             INT[],
    start_date         TEXT,
    end_date           TEXT,
    notify             BOOLEAN,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS reminder_marker
(
    id                 UUID,
    changed            INT,
    "user"             INT,
    income_instrument  INT,
    income_account     TEXT,
    income             FLOAT,
    outcome_instrument INT,
    outcome_account    TEXT,
    outcome            FLOAT,
    tag                UUID[],
    merchant           UUID,
    payee              TEXT,
    comment            TEXT,
    date               TEXT,
    reminder           UUID,
    state              TEXT,
    notify             BOOLEAN,
    is_forecast        BOOLEAN,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS transaction
(
    id                    UUID,
    changed               INT,
    created               INT,
    "user"                INT,
    deleted               BOOLEAN,
    hold                  BOOLEAN,
    income_instrument     INT,
    income_account        TEXT,
    income                FLOAT,
    outcome_instrument    INT,
    outcome_account       TEXT,
    outcome               FLOAT,
    tag                   UUID[],
    merchant              UUID,
    payee                 TEXT,
    original_payee        TEXT,
    comment               TEXT,
    date                  TEXT,
    mcc                   INT,
    reminder_marker       UUID,
    op_income             FLOAT,
    op_income_instrument  INT,
    op_outcome            FLOAT,
    op_outcome_instrument INT,
    latitude              FLOAT,
    longitude             FLOAT,
    viewed               BOOLEAN,
    qr_code               TEXT,
    source                TEXT,
    income_bank_id        TEXT,
    outcome_bank_id       TEXT,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS budget
(
    changed             INT,
    "user"              INT,
    tag                 UUID,
    date                TEXT,
    income              FLOAT,
    income_lock         BOOLEAN,
    outcome             FLOAT,
    outcome_lock        BOOLEAN,
    is_income_forecast  BOOLEAN,
    is_outcome_forecast BOOLEAN
);

-- Create sync type enum
CREATE TYPE sync_type AS ENUM ('full', 'partial', 'force');

-- Create sync status enum
CREATE TYPE status AS ENUM ('completed', 'failed');

-- Create sync status table
CREATE TABLE IF NOT EXISTS sync_status
(
    id                BIGSERIAL PRIMARY KEY,
    started_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at       TIMESTAMP WITH TIME ZONE,
    sync_type         sync_type                NOT NULL,
    server_timestamp  BIGINT                   NOT NULL,
    records_processed INTEGER                           DEFAULT 0,
    status            VARCHAR(20)              NOT NULL DEFAULT 'completed',
    error_message     TEXT,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS deletion_history
(
    id          BIGSERIAL PRIMARY KEY,
    object_id   TEXT                     NOT NULL,
    object_type TEXT                     NOT NULL,
    user_id     INTEGER                  NOT NULL,
    deleted_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Создаем индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_deletion_history_object
    ON deletion_history (object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_deletion_history_user
    ON deletion_history (user_id);
CREATE INDEX IF NOT EXISTS idx_deletion_history_deleted_at
    ON deletion_history (deleted_at);