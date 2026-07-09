package member

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrWalletConflict = errors.New("wallet version conflict")

var ledgerSeq atomic.Uint64

// NewLedgerNo 生成全局唯一帐变流水号：前缀 + 纳秒时间戳 + 进程内自增序列。
// 纳秒精度叠加单调自增序列，确保同一会员同一秒内多笔（如批量派奖）也不会
// 撞 uq_wallet_ledger_ledger_no 唯一约束。长度 ≤ 27，满足 VARCHAR(32)。
func NewLedgerNo(prefix string) string {
	return fmt.Sprintf("%s%d%04d", prefix, time.Now().UTC().UnixNano(), ledgerSeq.Add(1)%10000)
}

// DebitWalletForBet locks the wallet, debits stake, and writes a bet_debit ledger row.
func DebitWalletForBet(ctx context.Context, qtx *sqlcdb.Queries, memberID int64, orderNo string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("invalid bet amount")
	}
	wallet, err := qtx.LockMemberWallet(ctx, memberID)
	if err != nil {
		return err
	}
	balance := RoundMoney(wallet.Balance)
	if balance < amount {
		return ErrInsufficientFunds
	}
	newBalance := RoundMoney(balance - amount)
	rows, err := qtx.UpdateMemberWalletBalances(ctx, sqlcdb.UpdateMemberWalletBalancesParams{
		MemberID:      memberID,
		Balance:       NumericFromFloat(newBalance),
		FrozenBalance: NumericFromFloat(wallet.FrozenBalance),
		Version:       wallet.Version,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("wallet version conflict")
	}
	ledgerNo := NewLedgerNo("MLBD")
	return qtx.InsertWalletLedger(ctx, sqlcdb.InsertWalletLedgerParams{
		LedgerNo:     ledgerNo,
		MemberID:     memberID,
		TxnType:      "bet_debit",
		DeltaAmount:  NumericFromFloat(-amount),
		BalanceAfter: NumericFromFloat(newBalance),
		OrderRef:     pgtype.Text{String: orderNo, Valid: true},
	})
}

func NumericFromFloat(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", v))
	return n
}

func RoundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// CreditWalletPayout credits winnings after a settled winning bet (stake already debited at place time).
func CreditWalletPayout(ctx context.Context, qtx *sqlcdb.Queries, memberID int64, orderNo string, payout float64) error {
	if payout <= 0 {
		return nil
	}
	wallet, err := qtx.LockMemberWallet(ctx, memberID)
	if err != nil {
		return err
	}
	newBalance := RoundMoney(wallet.Balance + payout)
	rows, err := qtx.UpdateMemberWalletBalances(ctx, sqlcdb.UpdateMemberWalletBalancesParams{
		MemberID:      memberID,
		Balance:       NumericFromFloat(newBalance),
		FrozenBalance: NumericFromFloat(wallet.FrozenBalance),
		Version:       wallet.Version,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("wallet version conflict")
	}
	ledgerNo := NewLedgerNo("MLPY")
	return qtx.InsertWalletLedger(ctx, sqlcdb.InsertWalletLedgerParams{
		LedgerNo:     ledgerNo,
		MemberID:     memberID,
		TxnType:      "payout",
		DeltaAmount:  NumericFromFloat(payout),
		BalanceAfter: NumericFromFloat(newBalance),
		OrderRef:     pgtype.Text{String: orderNo, Valid: true},
	})
}

// CreditWalletDeposit credits balance after a demo/sandbox recharge (instant paid).
func CreditWalletDeposit(ctx context.Context, qtx *sqlcdb.Queries, memberID int64, orderNo string, amount float64) (balance, frozen float64, err error) {
	if amount <= 0 {
		return 0, 0, fmt.Errorf("invalid deposit amount")
	}
	wallet, err := qtx.LockMemberWallet(ctx, memberID)
	if err != nil {
		return 0, 0, err
	}
	newBalance := RoundMoney(wallet.Balance + amount)
	frozen = RoundMoney(wallet.FrozenBalance)
	rows, err := qtx.UpdateMemberWalletBalances(ctx, sqlcdb.UpdateMemberWalletBalancesParams{
		MemberID:      memberID,
		Balance:       NumericFromFloat(newBalance),
		FrozenBalance: NumericFromFloat(frozen),
		Version:       wallet.Version,
	})
	if err != nil {
		return 0, 0, err
	}
	if rows == 0 {
		return 0, 0, ErrWalletConflict
	}
	ledgerNo := NewLedgerNo("MLDP")
	err = qtx.InsertWalletLedger(ctx, sqlcdb.InsertWalletLedgerParams{
		LedgerNo:     ledgerNo,
		MemberID:     memberID,
		TxnType:      "deposit",
		DeltaAmount:  NumericFromFloat(amount),
		BalanceAfter: NumericFromFloat(newBalance),
		OrderRef:     pgtype.Text{String: orderNo, Valid: true},
	})
	if err != nil {
		return 0, 0, err
	}
	return newBalance, frozen, nil
}

// PayoutGross returns stake + net profit credited on win.
func PayoutGross(stake, pnl float64) float64 {
	return RoundMoney(stake + pnl)
}

// MirrorRealLedger 写第三方 real 行为镜像流水（T5）：不改本地 member_wallets，
// balance_after 存第三方主币种余额快照（B1：满足 >=0 NOT NULL）。
//   - txnType：'bet_debit'（接单）或 'payout'（派奖）
//   - delta：投注为负、派奖为正
//   - balanceSnapshot：下注/派奖后第三方主币种可用余额（<0 则按 0 记，避免违反约束）
func MirrorRealLedger(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	memberID int64,
	orderNo string,
	txnType string,
	delta float64,
	balanceSnapshot float64,
	guajiAccountID int64,
	currency string,
) error {
	if balanceSnapshot < 0 {
		balanceSnapshot = 0
	}
	prefix := "MLMR"
	if txnType == "payout" {
		prefix = "MLMP"
	}
	return qtx.InsertWalletLedgerMirror(ctx, sqlcdb.InsertWalletLedgerMirrorParams{
		LedgerNo:       NewLedgerNo(prefix),
		MemberID:       memberID,
		TxnType:        txnType,
		DeltaAmount:    NumericFromFloat(RoundMoney(delta)),
		BalanceAfter:   NumericFromFloat(RoundMoney(balanceSnapshot)),
		OrderRef:       pgtype.Text{String: orderNo, Valid: orderNo != ""},
		GuajiAccountID: pgtype.Int8{Int64: guajiAccountID, Valid: guajiAccountID != 0},
		Currency:       pgtype.Text{String: currency, Valid: currency != ""},
		Remark:         pgtype.Text{String: "第三方镜像", Valid: true},
	})
}
