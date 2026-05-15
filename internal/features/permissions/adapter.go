package permissions

import (
	"context"
	"publimd/internal/shared/models"
)

type CollaboratorRepoAdapter struct {
	repo interface {
		GetBasicInfoByID(ctx context.Context, id uint64) (*models.UserBasicInfo, error)
		GetCollaboratorPermission(ctx context.Context, postID uint64, userID uint64) (bool, uint64, error)
		IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error)
	}
}

func NewCollaboratorRepoAdapter(repo interface {
	GetBasicInfoByID(ctx context.Context, id uint64) (*models.UserBasicInfo, error)
	GetCollaboratorPermission(ctx context.Context, postID uint64, userID uint64) (bool, uint64, error)
	IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error)
}) PermissionRepository {
	return &CollaboratorRepoAdapter{repo: repo}
}

func (a *CollaboratorRepoAdapter) GetBasicInfoByID(ctx context.Context, id uint64) (uint64, error) {
	info, err := a.repo.GetBasicInfoByID(ctx, id)
	if err != nil {
		return 0, err
	}
	return info.ID, nil
}

func (a *CollaboratorRepoAdapter) GetCollaboratorPermission(ctx context.Context, postID uint64, userID uint64) (bool, uint64, error) {
	return a.repo.GetCollaboratorPermission(ctx, postID, userID)
}

func (a *CollaboratorRepoAdapter) IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	return a.repo.IsAuthor(ctx, postID, userID)
}
