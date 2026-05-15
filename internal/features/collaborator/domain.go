package collaborator

import (
	"context"
	"publimd/internal/shared/models"
)

type CollaboratorInfoBasic struct {
	UserID       uint64 `json:"user_id" gorm:"column:user_id"`
	Username     string `json:"username" gorm:"column:username"`
	Email        string `json:"email" gorm:"column:email"`
	Avatar       string `json:"avatar" gorm:"column:avatar"`
	PermissionID uint64 `json:"permission_id" gorm:"column:permission_id"`
	Confirmed    bool   `json:"confirmed" gorm:"column:confirmed"`
}

type PostCollaboratorsData struct {
	IsAuthor     bool   `json:"is_author" gorm:"column:is_author"`
	PermissionID uint64 `json:"permission_id" gorm:"column:permission_id"`

	Collaborators []CollaboratorInfoBasic `json:"collaborators" gorm:"-"`
}

type CollaboratorInviteRequest struct {
	ID    uint64 `json:"id" gorm:"column:id"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type CollaboratorInviteInfo struct {
	UserID    uint64 `gorm:"column:user_id"`
	Name      string `gorm:"column:name"`
	Email     string `gorm:"column:email"`
	PostTitle string `gorm:"column:post_title"`
	Confirmed bool   `gorm:"column:confirmed"`
}

type UserBasicInfo struct {
	UserID uint64 `json:"user_id" gorm:"column:user_id"`
	Avatar string `json:"avatar" gorm:"column:avatar"`
}

type PostSlugResolver interface {
	GetBasicInfoBySlug(ctx context.Context, slug string) (*models.PostInfoBasic, error)
}

type CollaboratorRepository interface {
	GetBasicInfoByID(ctx context.Context, id uint64) (*models.UserBasicInfo, error)
	GetCollaboratorPermission(ctx context.Context, postID uint64, userID uint64) (bool, uint64, error)
	IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error)

	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetAll(ctx context.Context, postID uint64, userID uint64) (*PostCollaboratorsData, error)
	GetAllUserInfoBasic(ctx context.Context, postID uint64) ([]UserBasicInfo, error)
	GetUserByEmail(ctx context.Context, email string) (*CollaboratorInviteRequest, error)
	GetCollaboratorInviteInfo(ctx context.Context, postID uint64, userID uint64) (*CollaboratorInviteInfo, error)
	Create(ctx context.Context, cols *models.Collaborator) error
	Delete(ctx context.Context, postID uint64, userID uint64) error
	UpdatePermission(ctx context.Context, postID uint64, userID uint64, permissionID uint64) error
	ConfirmCollaborator(ctx context.Context, postID uint64, userID uint64) error
}

type CollaboratorService interface {
	GetAllPermissions(ctx context.Context, authID uint64, slug string) ([]models.Permission, error)
	GetAll(ctx context.Context, authID uint64, slug string) (*PostCollaboratorsData, error)
	GetAllUserInfoBasic(ctx context.Context, authID uint64, slug string) ([]UserBasicInfo, error)
	GetUserByEmail(ctx context.Context, email string, authID uint64, slug string) (*CollaboratorInviteRequest, error)
	Create(ctx context.Context, actorAuthID uint64, slug string, targetUserID uint64, permissionID uint64) error
	UpdatePermission(ctx context.Context, actorAuthID uint64, slug string, targetUserID uint64, permissionID uint64) error
	Delete(ctx context.Context, actorAuthID uint64, slug string, targetUserID uint64) error
	ConfirmInvitation(ctx context.Context, token string) error
	ResendInvitation(ctx context.Context, actorAuthID uint64, slug string, targetUserID uint64) error
}
