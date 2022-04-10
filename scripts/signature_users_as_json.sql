WITH sigs AS (
  SELECT
    users.user_id AS id,
    users.username,
    users.discriminator,
    users.avatar_hash AS avatar
  FROM signatures
  INNER JOIN users
  ON signatures.user_id = users.user_id
  ORDER BY signatures.created_at ASC
)
SELECT JSON_AGG(sigs.*)
FROM sigs;
