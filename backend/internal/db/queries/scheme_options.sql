-- name: GetLotterySchemeOptionSet :one
SELECT lottery_code, run_types, play_types, sub_plays
FROM lottery_scheme_option_sets
WHERE lottery_code = $1;
