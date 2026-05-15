package user

import (
	"context"
	"fmt"
	"publimd/internal/shared/models"
	"strings"
)

type UserBasicInfo struct {
	ID     uint64 `json:"id" gorm:"column:id"`
	AuthID uint64 `json:"auth_id" gorm:"column:auth_id"`
}

type UserDashboardMetrics struct {
	TotalPosts          uint64 `json:"total_posts" gorm:"column:total_posts"`
	TotalCollaborations uint64 `json:"total_collaborations" gorm:"column:total_collaborations"`
	TotalLikes          uint64 `json:"total_likes" gorm:"column:total_likes"`
	TotalComments       uint64 `json:"total_comments" gorm:"column:total_comments"`
}

type UserRepository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetDashboardMetrics(ctx context.Context, userID uint64) (*UserDashboardMetrics, error)
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint64) (*models.User, error)
	GetByAuthID(ctx context.Context, id uint64) (*models.User, error)
	GetBasicInfoByID(ctx context.Context, id uint64) (*UserBasicInfo, error)
	Update(ctx context.Context, authID uint64, data map[string]any) error
}

type UserService interface {
	Create(ctx context.Context, user *models.User) error
	GetDashboardMetrics(ctx context.Context, authID uint64) (*UserDashboardMetrics, error)
	GetByID(ctx context.Context, id uint64) (*models.User, error)
	GetByAuthID(ctx context.Context, id uint64) (*models.User, error)
	GetBasicInfoByID(ctx context.Context, id uint64) (*UserBasicInfo, error)
	Update(ctx context.Context, authID uint64, user *models.User) error
}

func NewUser(data *models.User) (*models.User, error) {
	if data.AuthID == 0 {
		return nil, fmt.Errorf("el AuthID no puede ser cero")
	}

	if strings.TrimSpace(data.Name) == "" {
		return nil, fmt.Errorf("el nombre no puede estar vacío")
	}

	if strings.TrimSpace(data.LastName) == "" {
		return nil, fmt.Errorf("el apellido no puede estar vacío")
	}

	if strings.TrimSpace(data.Bio) == "" {
		return nil, fmt.Errorf("la biografía no puede estar vacía")
	}

	if strings.TrimSpace(data.Avatar) == "" {
		return nil, fmt.Errorf("el avatar no puede estar vacío")
	}

	if len(data.Name) > 100 {
		return nil, fmt.Errorf("el nombre no puede tener más de 100 caracteres")
	}

	if len(data.LastName) > 100 {
		return nil, fmt.Errorf("el apellido no puede tener más de 100 caracteres")
	}

	return data, nil
}

func BuildUserUpdateData(data *models.User) map[string]any {
	updateData := make(map[string]any)

	if strings.TrimSpace(data.Name) != "" {
		updateData["name"] = data.Name
	}

	if strings.TrimSpace(data.LastName) != "" {
		updateData["last_name"] = data.LastName
	}

	if strings.TrimSpace(data.Bio) != "" {
		updateData["bio"] = data.Bio
	}

	if strings.TrimSpace(data.Avatar) != "" {
		updateData["avatar"] = data.Avatar
	}

	return updateData
}
