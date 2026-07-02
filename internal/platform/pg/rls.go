package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	settingDocAccess = "app.doc_access"
	settingUserID    = "app.user_id"
	defaultDocAccess = "internal"
	defaultUserID    = "system"
)

const setRLSSQL = "select set_config($1, $2, true), set_config($3, $4, true)"

type Principal struct {
	UserID    string
	DocAccess string
}

func SetRLS(ctx context.Context, tx pgx.Tx, principal Principal) error {
	access, userID := normalizeRLS(principal)
	if _, err := tx.Exec(ctx, setRLSSQL, settingDocAccess, access, settingUserID, userID); err != nil {
		return fmt.Errorf("set rls context: %w", err)
	}
	return nil
}

func normalizeRLS(principal Principal) (string, string) {
	access := principal.DocAccess
	if access == "" {
		access = defaultDocAccess
	}
	userID := principal.UserID
	if userID == "" {
		userID = defaultUserID
	}
	return access, userID
}
