package member



import (

	"context"

	"errors"

	"fmt"

	"log/slog"

	"strings"



	"github.com/jackc/pgx/v5"

	"golang.org/x/crypto/bcrypt"



	"caipiao/backend/internal/db/sqlcdb"

	"caipiao/backend/internal/ws"

)



var (

	ErrInvalidOp       = errors.New("invalid member op")

	defaultResetSecret = "Reset123456"

)



type AdminMemberOpInput struct {

	Action string `json:"action"`

}



type AdminMemberOpResult struct {

	Action       string         `json:"action"`

	Member       AdminMemberRow `json:"member"`

	Message      string         `json:"message,omitempty"`

	TempPassword string         `json:"tempPassword,omitempty"`

}



func (s *Service) AdminApplyOp(ctx context.Context, memberID int64, in AdminMemberOpInput) (AdminMemberOpResult, error) {

	if s == nil || s.q == nil {

		return AdminMemberOpResult{}, ErrUnavailable

	}

	if memberID <= 0 {

		return AdminMemberOpResult{}, ErrNotFound

	}



	action := strings.TrimSpace(in.Action)

	switch action {

	case "reset_login_password":

		return s.adminResetPassword(ctx, memberID, "登录密码")

	case "toggle_freeze":

		return s.adminToggleFreeze(ctx, memberID)

	default:

		return AdminMemberOpResult{}, fmt.Errorf("%w: 未知 action %q", ErrInvalidOp, action)

	}

}



func (s *Service) adminResetPassword(ctx context.Context, memberID int64, label string) (AdminMemberOpResult, error) {

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultResetSecret), bcrypt.DefaultCost)

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	rows, err := s.q.AdminUpdateMemberPasswordByID(ctx, sqlcdb.AdminUpdateMemberPasswordByIDParams{

		ID: memberID, PasswordHash: string(hash),

	})

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	if rows == 0 {

		return AdminMemberOpResult{}, ErrNotFound

	}

	member, err := s.AdminGetMember(ctx, memberID)

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	return AdminMemberOpResult{

		Action:       "reset_login_password",

		Member:       member,

		Message:      fmt.Sprintf("已重置%s（演示环境统一写入同一哈希）", label),

		TempPassword: defaultResetSecret,

	}, nil

}



func (s *Service) adminToggleFreeze(ctx context.Context, memberID int64) (AdminMemberOpResult, error) {

	cur, err := s.q.GetMemberByID(ctx, memberID)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {

			return AdminMemberOpResult{}, ErrNotFound

		}

		return AdminMemberOpResult{}, err

	}

	next := "frozen"

	msg := "已禁用账号"

	if cur.Status == "frozen" {

		next = "active"

		msg = "已启用账号"

		rows, err := s.q.AdminUpdateMemberStatus(ctx, sqlcdb.AdminUpdateMemberStatusParams{

			ID: memberID, Status: next,

		})

		if err != nil {

			return AdminMemberOpResult{}, err

		}

		if rows == 0 {

			return AdminMemberOpResult{}, ErrNotFound

		}

		member, err := s.AdminGetMember(ctx, memberID)

		if err != nil {

			return AdminMemberOpResult{}, err

		}

		return AdminMemberOpResult{

			Action:  "toggle_freeze",

			Member:  member,

			Message: msg,

		}, nil

	}



	if s.pool == nil {

		return AdminMemberOpResult{}, ErrUnavailable

	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	defer tx.Rollback(ctx)



	qtx := sqlcdb.New(tx)

	rows, err := qtx.AdminUpdateMemberStatus(ctx, sqlcdb.AdminUpdateMemberStatusParams{

		ID: memberID, Status: next,

	})

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	if rows == 0 {

		return AdminMemberOpResult{}, ErrNotFound

	}

	paused, err := qtx.PauseRunningPendingInstancesByMember(ctx, cur.ID)

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	if err := tx.Commit(ctx); err != nil {

		return AdminMemberOpResult{}, err

	}

	s.notifyPausedSchemeInstances(cur.Account, paused)

	if n := len(paused); n > 0 {

		msg = fmt.Sprintf("已禁用账号，已暂停 %d 个方案", n)

		slog.Info("member disabled, schemes paused", "memberId", memberID, "account", cur.Account, "count", n)

	}



	member, err := s.AdminGetMember(ctx, memberID)

	if err != nil {

		return AdminMemberOpResult{}, err

	}

	return AdminMemberOpResult{

		Action:  "toggle_freeze",

		Member:  member,

		Message: msg,

	}, nil

}



func (s *Service) notifyPausedSchemeInstances(account string, paused []sqlcdb.PauseRunningPendingInstancesByMemberRow) {

	if s.hub == nil || account == "" || len(paused) == 0 {

		return

	}

	for _, inst := range paused {

		runMode := "real"

		if inst.SimBet {

			runMode = "sim"

		}

		ws.PublishSchemeInstance(s.hub, account, ws.SchemeInstancePayload{

			InstanceID: inst.ID,

			RunMode:    runMode,

			SimBet:     inst.SimBet,

			Status:     "paused",

			Reason:     "manual",

			Hint:       "refresh_running_list",

		})

	}

}

