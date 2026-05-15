package auth

import (
	"context"
	"fmt"
	"publimd/internal/shared/models"
	"publimd/internal/shared/utils"

	"strings"
)

type UserBasic struct {
	ID    string `gorm:"column:id"`
	Email string `gorm:"column:email"`
}

type AuthRepository interface {
	Register(user *models.Auth) error
	GetByEmail(email string) (*models.Auth, error)
	ExistsByEmail(email string) (bool, error)
	GetActiveUserByEmail(email string) (*models.Auth, error)
	GetByID(userId uint64) (*models.Auth, error)
	GetBasicByID(userId uint64) (*UserBasic, error)
	GetBasicByEmail(email string) (*UserBasic, error)

	UpdateEmailConfirm(userId uint64) error
	UpdatePassword(userId uint64, newPassword string) error
	SetTwoFactor(userId uint64, enabled bool, secret string) error
	ChangeCompletProfile(userId uint64, complete bool) error
}

type AuthService interface {
	Register(ctx context.Context, email string, password string) error
	Login(ctx context.Context, email string, password string) (*models.Auth, string, error)
	GetSession(userId uint64) (*models.Auth, error)
	RefreshToken(ctx context.Context, token string) (string, error)
	VerifyEmail(email string) error
	ExistsByEmail(email string) (bool, error)
	ForwardEmailVerification(ctx context.Context, email string) error
	ConfirmAccount(ctx context.Context, userId uint64, token string) error
	SendPasswordReset(ctx context.Context, email string) error
	ResetPassword(userId uint64, newPassword string) error
	UpdatePassword(userId uint64, currentPassword string, newPassword string) error
	UpdateCompletProfile(userId uint64) error
	ChangeCompletProfile(userId uint64, complete bool) error
	EnableTWOFA(userId uint64) (string, string, error)
	VerifyTOTP(userId uint64, code string) (bool, error)
	DisableTWOFA(userId uint64) error
	Verify2FALogin(ctx context.Context, sessionID string, code string) (*models.Auth, string, error)
	Logout(ctx context.Context, token string) error
}

func NewUser(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		Email:        email,
		PasswordHash: password,
	}, nil
}

func NewUserAdmin(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		Email:          email,
		PasswordHash:   password,
		FullProfile:    false,
		EmailConfirmed: false,
		FirstSession:   false,
	}, nil
}
