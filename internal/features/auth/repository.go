package auth

import (
	"log"
	"publimd/internal/shared/errorsx"
	"publimd/internal/shared/models"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) AuthRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Register(user *models.Auth) error {
	return r.db.Create(user).Error
}

func (r *PostgresRepository) GetByEmail(email string) (*models.Auth, error) {
	var user models.Auth
	err := r.db.Where("email = ?", email).First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) GetBasicByID(id uint64) (*UserBasic, error) {
	var user UserBasic
	err := r.db.Model(&models.Auth{}).Where("id = ?", id).Take(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, err
}

func (r *PostgresRepository) GetBasicByEmail(email string) (*UserBasic, error) {
	var user UserBasic
	err := r.db.Model(&models.Auth{}).Where("email = ?", email).Take(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, err
}

func (r *PostgresRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Auth{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) GetActiveUserByEmail(email string) (*models.Auth, error) {
	var user models.Auth
	err := r.db.
		Where("email = ?", email).First(&user).Error

	if err != nil {
		return nil, err
	}

	if user.EmailConfirmed == false {
		log.Printf("Intento de inicio de sesión con email no verificado: %s", email)
		return nil, errorsx.ErrCodeEmailNotVerified
	}

	return &user, nil
}

func (r *PostgresRepository) GetByID(userId uint64) (*models.Auth, error) {
	var user models.Auth
	err := r.db.Where("id = ?", userId).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateEmailConfirm(userId uint64) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("email_confirmed", true).Error
}

func (r *PostgresRepository) UpdatePassword(userId uint64, newPassword string) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("password_hash", newPassword).Error
}

func (r *PostgresRepository) ChangeCompletProfile(userId uint64, complete bool) error {
	return r.db.Model(&models.Auth{}).Where("id = ?", userId).Update("full_profile", complete).Error
}

func (r *PostgresRepository) SetTwoFactor(userId uint64, enabled bool, secret string) error {
	updates := map[string]any{
		"two_factor_enabled": enabled,
		"two_factor_secret":  "",
	}

	if enabled {
		updates["two_factor_secret"] = secret
	}

	return r.db.Model(&models.Auth{}).
		Where("id = ?", userId).
		Updates(updates).Error
}
