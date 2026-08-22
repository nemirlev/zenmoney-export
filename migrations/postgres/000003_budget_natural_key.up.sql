-- ZenMoney budgets have no standalone ID. Their identity is the combination
-- of user, month, and tag. Keep the newest copy before enforcing that key on
-- databases that may already contain duplicates from previous full syncs.
WITH ranked_budgets AS (
    SELECT ctid,
           ROW_NUMBER() OVER (
               PARTITION BY "user", date, tag
               ORDER BY changed DESC NULLS LAST, ctid DESC
           ) AS duplicate_number
    FROM budget
)
DELETE
FROM budget
USING ranked_budgets
WHERE budget.ctid = ranked_budgets.ctid
  AND ranked_budgets.duplicate_number > 1;

-- NULL is a real tag state in the API (uncategorized operations), so it must
-- participate in uniqueness instead of behaving as a distinct value per row.
ALTER TABLE budget
    ADD CONSTRAINT budget_user_date_tag_key
        UNIQUE NULLS NOT DISTINCT ("user", date, tag);
