-- db/migrations/0015_fix_pairing_tie_and_double_loss.sql
--
-- Limitless encodes winner = 0 as a tie and winner = -1 as a double loss.
-- Earlier ingestion stored 0 as 'unknown' and -1 as 'draw'. Fix both.
-- Order matters: fix -1 rows first so they aren't confused with the new draws.

UPDATE pairings
SET result = 'double_loss'
WHERE result = 'draw'
  AND player2_id IS NOT NULL
  AND raw_pairing->'winner' = '-1'::jsonb;

UPDATE pairings
SET result = 'draw'
WHERE result = 'unknown'
  AND player2_id IS NOT NULL
  AND raw_pairing->'winner' = '0'::jsonb;
