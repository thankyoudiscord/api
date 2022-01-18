-- psql -qAtX < scripts/ranked_users.sql > ranked_users.json

SELECT
  JSON_AGG(v.*)
FROM (
  SELECT
  users.user_id,
  users.username,
  users.discriminator,
  users.avatar_hash,
  COALESCE(ranked.referral_count, 0) AS referral_count,
  ROW_NUMBER() OVER (
    ORDER BY ranked.rank ASC, signatures.created_at ASC
  ) - 1 AS position
  FROM users
  LEFT JOIN signatures
  ON signatures.user_id = users.user_id
  LEFT JOIN (
    SELECT
    referrer_id,
    RANK() OVER (
      ORDER BY COUNT(referrer_id) DESC
    ),
    COUNT(referrer_id) AS referral_count
    FROM signatures
    GROUP BY referrer_id
  ) AS ranked
  ON users.user_id = ranked.referrer_id
  ORDER BY ranked.rank ASC, signatures.created_at ASC
) AS v;

-- vim:et ts=2 sw=2
