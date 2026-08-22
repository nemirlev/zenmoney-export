ALTER TABLE budget
    ALTER COLUMN changed TYPE INT;

ALTER TABLE transaction
    ALTER COLUMN changed TYPE INT,
    ALTER COLUMN created TYPE INT;

ALTER TABLE reminder_marker
    ALTER COLUMN changed TYPE INT;

ALTER TABLE reminder
    ALTER COLUMN changed TYPE INT;

ALTER TABLE merchant
    DROP COLUMN mcc,
    ALTER COLUMN changed TYPE INT;

ALTER TABLE tag
    DROP COLUMN archive,
    ALTER COLUMN changed TYPE INT;

ALTER TABLE account
    ALTER COLUMN changed TYPE INT;

ALTER TABLE company
    ALTER COLUMN changed TYPE INT;

ALTER TABLE instrument
    ALTER COLUMN changed TYPE INT;
