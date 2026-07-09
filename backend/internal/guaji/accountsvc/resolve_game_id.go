package accountsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// resolveGameID 以 lottery_catalog.outbound_lottery_code 为下单 game_id 唯一来源。
func (s *Service) resolveGameID(ctx context.Context, lotteryCode, fallbackGameID string) (int, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode != "" && s != nil && s.pool != nil {
		var outbound string
		err := s.pool.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(outbound_lottery_code), ''), code)
FROM lottery_catalog
WHERE code = $1`, lotteryCode).Scan(&outbound)
		if err != nil {
			return 0, fmt.Errorf("resolve game_id for %s: %w", lotteryCode, err)
		}
		gameID, err := strconv.Atoi(strings.TrimSpace(outbound))
		if err != nil || gameID <= 0 {
			return 0, fmt.Errorf("resolve game_id for %s: invalid outbound %q", lotteryCode, outbound)
		}
		return gameID, nil
	}
	gameID, err := strconv.Atoi(strings.TrimSpace(fallbackGameID))
	if err != nil || gameID <= 0 {
		return 0, fmt.Errorf("resolve game_id: missing lotteryCode and invalid game_id %q", fallbackGameID)
	}
	return gameID, nil
}
