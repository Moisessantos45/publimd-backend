package errorsx

import "errors"

var (
	ErrCodeEmailNotVerified = errors.New("EMAIL_NOT_VERIFIED")
	ErrEmailNotVerified     = errors.New("email no verificado")
	ErrPending2FA           = errors.New("PENDING_2FA")
)
