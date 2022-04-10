WITH ranked_by_referrals AS (
  SELECT
    referrer_id,
    RANK() OVER (
      ORDER BY ranked_by_referrals.rank ASC, signatures.created_at ASC;
    ),
    COUNT(referrer_id) AS referral_count
  FROM signatures
  GROUP BY referrer_id
)
SELECT
  ROW_NUMBER() OVER (
    ORDER BY ranked_by_referrals.
  ) AS position,
  ranked_by_referrals.referral_count,
  users.username,
  users.discriminator
FROM signatures
INNER JOIN users
ON signatures.user_id = users.user_id
INNER JOIN ranked_by_referrals
ON ranked_by_referrals.referrer_id = signatures.user_id
ORDER BY ranked_by_referrals.rank ASC, signatures.created_at ASC;
