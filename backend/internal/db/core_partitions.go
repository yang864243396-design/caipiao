package db

import (
	"context"
	"fmt"
)

const CorePartitionMonthsAhead = 12

func EnsureCoreOnlinePartitions(ctx context.Context, pool *Pool, monthsAhead int) (int, error) {
	if pool == nil {
		return 0, fmt.Errorf("core partition maintenance requires a database")
	}
	if monthsAhead < 0 || monthsAhead > 240 {
		return 0, fmt.Errorf("core partition months ahead must be between 0 and 240")
	}

	var created int
	err := pool.QueryRow(ctx, `
SELECT ensure_core_online_partitions(
    date_trunc('month', CURRENT_DATE)::date,
    $1
)`, monthsAhead).Scan(&created)
	if err != nil {
		return 0, fmt.Errorf("ensure core online partitions: %w", err)
	}
	return created, nil
}
