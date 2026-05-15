package user

import (
	"context"
	"fmt"
	"log"
	"publimd/internal/shared/models"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) UserRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) WithTransaction(fn func(repo *PostgresRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) GetDashboardMetrics(ctx context.Context, userID uint64) (*UserDashboardMetrics, error) {
	var metrics UserDashboardMetrics
	err := r.db.WithContext(ctx).Raw(`
    SELECT
        (SELECT COUNT(*) FROM posts WHERE author_id = ?) AS total_posts,
        (SELECT COUNT(*) FROM collaborators WHERE user_id = ?) AS total_collaborations,
        (SELECT COALESCE(SUM(l.id), 0) FROM likes l JOIN posts p ON l.post_id = p.id WHERE p.author_id = ?) AS total_likes,
        (SELECT COUNT(*) FROM comments c JOIN posts p ON c.post_id = p.id WHERE p.author_id = ?) AS total_comments
	`, userID, userID, userID, userID).Scan(&metrics).Error

	if err != nil {
		return nil, err
	}

	log.Printf("Métricas del dashboard para el usuario %d: %+v", userID, metrics)

	return &metrics, nil
}

func (r *PostgresRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con ID %d no encontrado", id)
		}

		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) GetByAuthID(ctx context.Context, id uint64) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("auth_id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con auth_id %d no encontrado", id)
		}

		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) GetBasicInfoByID(ctx context.Context, id uint64) (*UserBasicInfo, error) {
	var info UserBasicInfo
	if err := r.db.WithContext(ctx).Model(&models.User{}).Select("id, auth_id").Where("auth_id = ?", id).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con ID %d no encontrado", id)
		}

		return nil, err
	}

	return &info, nil
}

func (r *PostgresRepository) Update(ctx context.Context, authID uint64, data map[string]any) error {
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("auth_id = ?", authID).Updates(data).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("usuario con ID %d no encontrado", authID)
		}

		return err
	}

	return nil
}
