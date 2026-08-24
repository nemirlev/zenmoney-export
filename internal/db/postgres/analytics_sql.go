package postgres

// Analytics query parameters shared by the read-only reports:
//
//	$1 authenticated user IDs (bigint[])
//	$2 inclusive start date
//	$3 exclusive end date
//	$4 include held transactions
//	$5 account IDs (uuid[])
//	$6 category IDs (uuid[])
//	$7 merchant IDs (uuid[])
//
// Keeping this scope in one CTE makes the tenant and transaction-state policy
// identical across every aggregate.
const analyticsTransactionScopeSQL = `
scoped_transactions AS (
    SELECT t.id,
           t."user",
           t.date,
           t.created,
           t.income,
           t.outcome,
           t.income_instrument,
           t.outcome_instrument,
           t.income_account,
           t.outcome_account,
           t.tag,
           t.merchant,
           t.payee,
           t.original_payee,
           t.comment,
           t.hold,
           income_account.type AS account_type
    FROM transaction t
    JOIN account income_account
      ON income_account.id = t.income_account
     AND income_account."user" = t."user"
    WHERE t."user"::bigint = ANY($1::bigint[])
      AND t.deleted IS FALSE
      AND (t.hold IS FALSE OR ($4::boolean AND t.hold IS TRUE))
      AND income_account.in_balance IS TRUE
      AND t.date >= $2::date
      AND t.date < $3::date
      AND (
          cardinality($5::uuid[]) = 0
          OR t.income_account = ANY($5::uuid[])
          OR EXISTS (
              SELECT 1
              FROM account selected_outcome_account
              WHERE selected_outcome_account.id = t.outcome_account
                AND selected_outcome_account."user" = t."user"
                AND selected_outcome_account.id = ANY($5::uuid[])
          )
      )
      AND (
          cardinality($6::uuid[]) = 0
          OR EXISTS (
              SELECT 1
              FROM unnest(COALESCE(t.tag, '{}'::uuid[])) AS selected_tag(id)
              JOIN tag selected_category
                ON selected_category.id = selected_tag.id
               AND selected_category."user" = t."user"
              WHERE selected_category.id = ANY($6::uuid[])
                 OR selected_category.parent = ANY($6::uuid[])
          )
      )
      AND (
          cardinality($7::uuid[]) = 0
          OR EXISTS (
              SELECT 1
              FROM merchant selected_merchant
              WHERE selected_merchant.id = t.merchant
                AND selected_merchant."user" = t."user"
                AND selected_merchant.id = ANY($7::uuid[])
          )
      )
),
regular_legs AS (
    SELECT id AS transaction_id,
           "user",
           date,
           created,
           'income'::text AS direction,
           income AS amount,
           income_instrument AS instrument_id,
           income_account AS account_id,
           tag,
           merchant
    FROM scoped_transactions
    WHERE income_account IS NOT DISTINCT FROM outcome_account
      AND account_type IS DISTINCT FROM 'debt'
      AND income > 0

    UNION ALL

    SELECT id AS transaction_id,
           "user",
           date,
           created,
           'outcome'::text AS direction,
           outcome AS amount,
           outcome_instrument AS instrument_id,
           income_account AS account_id,
           tag,
           merchant
    FROM scoped_transactions
    WHERE income_account IS NOT DISTINCT FROM outcome_account
      AND account_type IS DISTINCT FROM 'debt'
      AND outcome > 0
),
target_instrument AS (
    SELECT report_user.id::bigint AS user_id,
           report_instrument.id,
           report_instrument.rate,
           report_instrument.short_title
    FROM "user" report_user
    JOIN instrument report_instrument ON report_instrument.id = report_user.currency
    WHERE report_user.id::bigint = ANY($1::bigint[])
),
converted_legs AS (
    SELECT leg.*,
           CASE
               WHEN leg.instrument_id = target.id THEN leg.amount
               WHEN source.rate > 0 AND target.rate > 0
                   THEN leg.amount * source.rate / target.rate
           END AS report_amount,
           CASE
               WHEN leg.instrument_id = target.id THEN FALSE
               WHEN source.id IS NULL OR source.rate IS NULL OR source.rate <= 0
                   OR target.rate IS NULL OR target.rate <= 0 THEN TRUE
               ELSE FALSE
           END AS invalid_rate
    FROM regular_legs leg
    JOIN target_instrument target ON target.user_id = leg."user"::bigint
    LEFT JOIN instrument source ON source.id = leg.instrument_id
)`

const spendingTotalsSQL = `WITH ` + analyticsTransactionScopeSQL + `
SELECT COALESCE(SUM(report_amount) FILTER (
           WHERE direction = 'outcome' AND NOT invalid_rate
       ), 0)::text AS total,
       COUNT(DISTINCT transaction_id) FILTER (
           WHERE direction = 'outcome' AND NOT invalid_rate
       ) AS transaction_count,
       COUNT(*) FILTER (
           WHERE direction = 'outcome' AND invalid_rate
       ) AS invalid_rate_count
FROM converted_legs`

const spendingCategoriesSQL = `WITH ` + analyticsTransactionScopeSQL + `,
outcome_legs AS (
    SELECT *
    FROM converted_legs
    WHERE direction = 'outcome'
      AND NOT invalid_rate
),
canonical_transaction_tags AS (
    SELECT DISTINCT outcome.transaction_id,
           COALESCE(parent.id, category.id) AS category_id,
           COALESCE(parent.title, category.title) AS category_title
    FROM outcome_legs outcome
    CROSS JOIN LATERAL unnest(COALESCE(outcome.tag, '{}'::uuid[])) AS tx_tag(id)
    JOIN tag category
      ON category.id = tx_tag.id
     AND category."user" = outcome."user"
    LEFT JOIN tag parent
      ON parent.id = category.parent
     AND parent."user" = outcome."user"
     AND parent.id <> category.id
),
tag_counts AS (
    SELECT transaction_id, COUNT(*) AS tag_count
    FROM canonical_transaction_tags
    GROUP BY transaction_id
),
allocated_categories AS (
    SELECT tags.category_id,
           tags.category_title,
           outcome.transaction_id,
           outcome.report_amount / counts.tag_count AS allocated_amount
    FROM outcome_legs outcome
    JOIN canonical_transaction_tags tags
      ON tags.transaction_id = outcome.transaction_id
    JOIN tag_counts counts
      ON counts.transaction_id = outcome.transaction_id
),
uncategorized AS (
    SELECT NULL::uuid AS category_id,
           NULL::text AS category_title,
           outcome.transaction_id,
           outcome.report_amount AS allocated_amount
    FROM outcome_legs outcome
    WHERE NOT EXISTS (
        SELECT 1
        FROM canonical_transaction_tags tags
        WHERE tags.transaction_id = outcome.transaction_id
    )
),
all_allocations AS (
    SELECT * FROM allocated_categories
    UNION ALL
    SELECT * FROM uncategorized
)
SELECT category_id::text,
       category_title,
       SUM(allocated_amount)::text AS amount,
       CASE
           WHEN SUM(SUM(allocated_amount)) OVER () = 0 THEN '0'
           ELSE (SUM(allocated_amount) * 100 /
                 SUM(SUM(allocated_amount)) OVER ())::text
       END AS share_percent,
       COUNT(DISTINCT transaction_id) AS transaction_count
FROM all_allocations
GROUP BY category_id, category_title
ORDER BY SUM(allocated_amount) DESC, category_id NULLS LAST
LIMIT $8::integer`

const cashflowTotalsSQL = `WITH ` + analyticsTransactionScopeSQL + `
SELECT COALESCE(SUM(report_amount) FILTER (
           WHERE direction = 'income' AND NOT invalid_rate
       ), 0)::text AS income,
       COALESCE(SUM(report_amount) FILTER (
           WHERE direction = 'outcome' AND NOT invalid_rate
       ), 0)::text AS outcome,
       COALESCE(SUM(CASE
           WHEN direction = 'income' AND NOT invalid_rate THEN report_amount
           WHEN direction = 'outcome' AND NOT invalid_rate THEN -report_amount
           ELSE 0
       END), 0)::text AS net,
       COUNT(*) FILTER (WHERE invalid_rate) AS invalid_rate_count
FROM converted_legs`

const budgetProgressCTESQL = analyticsTransactionScopeSQL + `,
outcome_legs AS (
    SELECT *
    FROM converted_legs
    WHERE direction = 'outcome'
      AND NOT invalid_rate
),
canonical_transaction_tags AS (
    SELECT DISTINCT outcome.transaction_id,
           COALESCE(parent.id, category.id) AS category_id
    FROM outcome_legs outcome
    CROSS JOIN LATERAL unnest(COALESCE(outcome.tag, '{}'::uuid[])) AS tx_tag(id)
    JOIN tag category
      ON category.id = tx_tag.id
     AND category."user" = outcome."user"
    LEFT JOIN tag parent
      ON parent.id = category.parent
     AND parent."user" = outcome."user"
     AND parent.id <> category.id
),
tag_counts AS (
    SELECT transaction_id, COUNT(*) AS tag_count
    FROM canonical_transaction_tags
    GROUP BY transaction_id
),
allocated_spending AS (
    SELECT tags.category_id,
           outcome.transaction_id,
           outcome.report_amount / counts.tag_count AS allocated_amount
    FROM outcome_legs outcome
    JOIN canonical_transaction_tags tags
      ON tags.transaction_id = outcome.transaction_id
    JOIN tag_counts counts
      ON counts.transaction_id = outcome.transaction_id

    UNION ALL

    SELECT NULL::uuid,
           outcome.transaction_id,
           outcome.report_amount
    FROM outcome_legs outcome
    WHERE NOT EXISTS (
        SELECT 1
        FROM canonical_transaction_tags tags
        WHERE tags.transaction_id = outcome.transaction_id
    )
),
spending_by_category AS (
    SELECT category_id,
           SUM(allocated_amount) AS spent,
           COUNT(DISTINCT transaction_id) AS transaction_count
    FROM allocated_spending
    GROUP BY category_id
),
converted_budgets AS (
    SELECT CASE
               WHEN budget.tag = '00000000-0000-0000-0000-000000000000'::uuid
                   THEN budget.tag
               ELSE COALESCE(parent.id, category.id)
           END AS category_id,
           CASE
               WHEN budget.tag = '00000000-0000-0000-0000-000000000000'::uuid
                   THEN 'All categories'
               ELSE COALESCE(parent.title, category.title)
           END AS category_title,
           CASE
               WHEN budget_user.currency = target.id THEN budget.outcome
               WHEN source.rate > 0 AND target.rate > 0
                   THEN budget.outcome * source.rate / target.rate
           END AS report_budget,
           CASE
               WHEN budget_user.currency = target.id THEN FALSE
               WHEN source.rate IS NULL OR source.rate <= 0
                 OR target.rate IS NULL OR target.rate <= 0 THEN TRUE
               ELSE FALSE
           END AS invalid_rate
    FROM budget
    JOIN "user" budget_user
      ON budget_user.id = budget."user"
     AND budget_user.id::bigint = ANY($1::bigint[])
    JOIN target_instrument target
      ON target.user_id = budget_user.id::bigint
    LEFT JOIN instrument source ON source.id = budget_user.currency
    LEFT JOIN tag category
      ON category.id = budget.tag
     AND category."user" = budget."user"
    LEFT JOIN tag parent
      ON parent.id = category.parent
     AND parent."user" = budget."user"
     AND parent.id <> category.id
    WHERE budget.date >= date_trunc('month', $2::date)::date
      AND budget.date < $3::date
      AND (
          cardinality($6::uuid[]) = 0
          OR category.id = ANY($6::uuid[])
          OR category.parent = ANY($6::uuid[])
      )
),
budgets_by_category AS (
    SELECT category_id,
           category_title,
           SUM(report_budget) AS budget
    FROM converted_budgets
    WHERE NOT invalid_rate
    GROUP BY category_id, category_title
),
budget_progress AS (
    SELECT budget.category_id,
           budget.category_title,
           budget.budget,
           CASE
               WHEN budget.category_id = '00000000-0000-0000-0000-000000000000'::uuid
                   THEN COALESCE((SELECT SUM(allocated_amount) FROM allocated_spending), 0)
               ELSE COALESCE(spending.spent, 0)
           END AS spent,
           CASE
               WHEN budget.category_id = '00000000-0000-0000-0000-000000000000'::uuid
                   THEN COALESCE((SELECT COUNT(DISTINCT transaction_id) FROM allocated_spending), 0)
               ELSE COALESCE(spending.transaction_count, 0)
           END AS transaction_count
    FROM budgets_by_category budget
    LEFT JOIN spending_by_category spending
      ON spending.category_id IS NOT DISTINCT FROM budget.category_id
)`

const budgetProgressRowsSQL = `WITH ` + budgetProgressCTESQL + `
SELECT category_id::text,
       category_title,
       budget::text,
       spent::text,
       (budget - spent)::text AS remaining,
       CASE WHEN budget = 0 THEN NULL ELSE (spent * 100 / budget)::text END AS percent,
       transaction_count
FROM budget_progress
ORDER BY spent DESC, category_id NULLS LAST
LIMIT $8::integer`

const budgetProgressTotalsSQL = `WITH ` + budgetProgressCTESQL + `
,
total_rows AS (
    SELECT *
    FROM budget_progress
    WHERE category_id = '00000000-0000-0000-0000-000000000000'::uuid

    UNION ALL

    SELECT *
    FROM budget_progress
    WHERE NOT EXISTS (
        SELECT 1
        FROM budget_progress
        WHERE category_id = '00000000-0000-0000-0000-000000000000'::uuid
    )
)
SELECT COALESCE(SUM(budget), 0)::text,
       COALESCE(SUM(spent), 0)::text,
       COALESCE(SUM(budget - spent), 0)::text,
       CASE WHEN COALESCE(SUM(budget), 0) = 0 THEN NULL
            ELSE (SUM(spent) * 100 / SUM(budget))::text
       END,
       (SELECT COUNT(*) FROM converted_legs
        WHERE direction = 'outcome' AND invalid_rate) +
       (SELECT COUNT(*) FROM converted_budgets WHERE invalid_rate)
FROM total_rows`

const latestCompletedSyncSQL = `
SELECT started_at, finished_at, status, sync_type::text,
       server_timestamp, records_processed
FROM sync_status
WHERE status = 'completed'
ORDER BY id DESC
LIMIT 1`

const latestSyncAttemptSQL = `
SELECT started_at, finished_at, status, sync_type::text,
       server_timestamp, records_processed
FROM sync_status
ORDER BY id DESC
LIMIT 1`

const searchTransactionsSQL = `
WITH target_instrument AS (
    SELECT report_user.id::bigint AS user_id,
           report_instrument.id,
           report_instrument.rate,
           report_instrument.short_title
    FROM "user" report_user
    JOIN instrument report_instrument ON report_instrument.id = report_user.currency
    WHERE report_user.id::bigint = ANY($1::bigint[])
),
search_rows AS (
    SELECT t.id,
           t."user",
           t.date,
           t.created,
           t.hold,
           t.income_account,
           outcome_account.id AS outcome_account,
           income_account.title AS account_title,
           income_account.type AS income_account_type,
           outcome_account.type AS outcome_account_type,
           t.income,
           t.outcome,
           t.income_instrument,
           t.outcome_instrument,
           t.tag,
           merchant.id AS merchant,
           t.payee,
           t.original_payee,
           t.comment,
           merchant.title AS merchant_title
    FROM transaction t
    JOIN account income_account
      ON income_account.id = t.income_account
     AND income_account."user" = t."user"
    LEFT JOIN account outcome_account
      ON outcome_account.id = t.outcome_account
     AND outcome_account."user" = t."user"
    LEFT JOIN merchant
      ON merchant.id = t.merchant
     AND merchant."user" = t."user"
    WHERE t."user"::bigint = ANY($1::bigint[])
      AND t.deleted IS FALSE
      AND (t.hold IS FALSE OR ($4::boolean AND t.hold IS TRUE))
      AND income_account.in_balance IS TRUE
      AND t.date >= $2::date
      AND t.date < $3::date
      AND (
          cardinality($5::uuid[]) = 0
          OR t.income_account = ANY($5::uuid[])
          OR outcome_account.id = ANY($5::uuid[])
      )
      AND (
          cardinality($6::uuid[]) = 0
          OR EXISTS (
              SELECT 1
              FROM unnest(COALESCE(t.tag, '{}'::uuid[])) AS selected_tag(id)
              JOIN tag selected_category
                ON selected_category.id = selected_tag.id
               AND selected_category."user" = t."user"
              WHERE selected_category.id = ANY($6::uuid[])
                 OR selected_category.parent = ANY($6::uuid[])
          )
      )
      AND (
          cardinality($7::uuid[]) = 0
          OR merchant.id = ANY($7::uuid[])
      )
      AND (
          BTRIM($8::text) = ''
          OR COALESCE(t.payee, '') ILIKE '%' || $8 || '%'
          OR COALESCE(t.original_payee, '') ILIKE '%' || $8 || '%'
          OR COALESCE(t.comment, '') ILIKE '%' || $8 || '%'
          OR COALESCE(merchant.title, '') ILIKE '%' || $8 || '%'
          OR EXISTS (
              SELECT 1
              FROM unnest(COALESCE(t.tag, '{}'::uuid[])) AS searched_tag(id)
              JOIN tag searched_category
                ON searched_category.id = searched_tag.id
               AND searched_category."user" = t."user"
              WHERE searched_category.title ILIKE '%' || $8 || '%'
          )
      )
      AND (
          $9::date IS NULL
          OR (t.date, t.created, t.id) < ($9::date, $10::bigint, $11::uuid)
      )
),
converted_search_rows AS (
    SELECT searched.*,
           CASE
               WHEN searched.income_instrument = target.id THEN searched.income
               WHEN income_instrument.rate > 0 AND target.rate > 0
                   THEN searched.income * income_instrument.rate / target.rate
           END AS report_income,
           CASE
               WHEN searched.outcome_instrument = target.id THEN searched.outcome
               WHEN outcome_instrument.rate > 0 AND target.rate > 0
                   THEN searched.outcome * outcome_instrument.rate / target.rate
           END AS report_outcome,
           CASE
               WHEN searched.income > 0
                AND searched.income_instrument <> target.id
                AND (income_instrument.rate IS NULL OR income_instrument.rate <= 0
                     OR target.rate IS NULL OR target.rate <= 0) THEN TRUE
               WHEN searched.outcome > 0
                AND searched.outcome_instrument <> target.id
                AND (outcome_instrument.rate IS NULL OR outcome_instrument.rate <= 0
                     OR target.rate IS NULL OR target.rate <= 0) THEN TRUE
               ELSE FALSE
           END AS invalid_rate
    FROM search_rows searched
    JOIN target_instrument target ON target.user_id = searched."user"::bigint
    LEFT JOIN instrument income_instrument
      ON income_instrument.id = searched.income_instrument
    LEFT JOIN instrument outcome_instrument
      ON outcome_instrument.id = searched.outcome_instrument
)
SELECT searched.id::text,
       to_char(searched.date, 'YYYY-MM-DD'),
       searched.created,
       CASE
           WHEN searched.income_account IS NOT DISTINCT FROM searched.outcome_account
            AND searched.income_account_type = 'debt'
            AND searched.income > 0 THEN 'debt_income'
           WHEN searched.income_account IS NOT DISTINCT FROM searched.outcome_account
            AND searched.income_account_type = 'debt'
            AND searched.outcome > 0 THEN 'debt_outcome'
           WHEN searched.income_account IS NOT DISTINCT FROM searched.outcome_account
            AND searched.income > 0 AND searched.outcome > 0 THEN 'mixed'
           WHEN searched.income_account IS NOT DISTINCT FROM searched.outcome_account
            AND searched.income > 0 THEN 'income'
           WHEN searched.income_account IS NOT DISTINCT FROM searched.outcome_account
            AND searched.outcome > 0 THEN 'outcome'
           WHEN searched.income_account_type = 'debt'
            AND searched.income > 0 THEN 'debt_income'
           WHEN searched.outcome_account_type = 'debt'
            AND searched.outcome > 0 THEN 'debt_outcome'
           WHEN searched.income_account IS DISTINCT FROM searched.outcome_account
               THEN 'transfer'
           ELSE 'invalid'
       END AS direction,
       (COALESCE(searched.report_income, 0) -
        COALESCE(searched.report_outcome, 0))::text AS amount,
       COALESCE(searched.report_income, 0)::text AS income,
       COALESCE(searched.report_outcome, 0)::text AS outcome,
       searched.income_account::text,
       searched.outcome_account::text,
       searched.account_title,
       COALESCE(categories.ids, ARRAY['category:uncategorized']::text[]),
       COALESCE(categories.titles, ARRAY['Uncategorized']::text[]),
       searched.merchant::text,
       searched.merchant_title,
       searched.hold,
       searched.invalid_rate
FROM converted_search_rows searched
LEFT JOIN LATERAL (
    SELECT array_agg(canonical.id ORDER BY canonical.id) AS ids,
           array_agg(canonical.title ORDER BY canonical.id) AS titles
    FROM (
        SELECT DISTINCT COALESCE(parent.id, category.id)::text AS id,
               COALESCE(NULLIF(BTRIM(parent.title), ''),
                        NULLIF(BTRIM(category.title), ''),
                        'Unnamed category') AS title
        FROM unnest(COALESCE(searched.tag, '{}'::uuid[])) AS tx_tag(id)
        JOIN tag category
          ON category.id = tx_tag.id
         AND category."user" = searched."user"
        LEFT JOIN tag parent
          ON parent.id = category.parent
         AND parent."user" = searched."user"
         AND parent.id <> category.id
    ) canonical
) categories ON TRUE
ORDER BY searched.date DESC, searched.created DESC, searched.id DESC
LIMIT $12::integer`
