package permissions

import (
	"context"
	"fmt"
)

type PermissionUseCase struct {
	repo PermissionRepository
}

func NewPermissionUseCase(repo PermissionRepository) PostPermissionChecker {
	return &PermissionUseCase{repo: repo}
}

func (uc *PermissionUseCase) resolveUserID(ctx context.Context, actorAuthID uint64) (uint64, error) {
	userID, err := uc.repo.GetBasicInfoByID(ctx, actorAuthID)
	if err != nil {
		return 0, fmt.Errorf("error fetching user info: %w", err)
	}
	return userID, nil
}

func (uc *PermissionUseCase) IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	if postID == 0 || userID == 0 {
		return false, fmt.Errorf("postID and userID cannot be zero")
	}

	return uc.repo.IsAuthor(ctx, postID, userID)
}

func (uc *PermissionUseCase) CanReadContent(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error) {
	userID, err := uc.resolveUserID(ctx, actorAuthID)
	if err != nil {
		return false, err
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking author: %w", err)
	}
	if isAuthor {
		return true, nil
	}

	isCollaborator, _, err := uc.repo.GetCollaboratorPermission(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking collaborator permission: %w", err)
	}

	return isCollaborator, nil
}

func (uc *PermissionUseCase) CanEditContent(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error) {
	userID, err := uc.resolveUserID(ctx, actorAuthID)
	if err != nil {
		return false, err
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking author: %w", err)
	}
	if isAuthor {
		return true, nil
	}

	isCollaborator, permissionID, err := uc.repo.GetCollaboratorPermission(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking collaborator permission: %w", err)
	}

	return isCollaborator && permissionID >= 2, nil
}

func (uc *PermissionUseCase) CanManagePost(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error) {
	userID, err := uc.resolveUserID(ctx, actorAuthID)
	if err != nil {
		return false, err
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking author: %w", err)
	}

	if isAuthor {
		return true, nil
	}

	isCollaborator, permissionID, err := uc.repo.GetCollaboratorPermission(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking collaborator permission: %w", err)
	}

	return isCollaborator && permissionID >= 3, nil
}

func (uc *PermissionUseCase) CanUpdatePermissions(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error) {
	userID, err := uc.resolveUserID(ctx, actorAuthID)
	if err != nil {
		return false, err
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking author: %w", err)
	}

	if isAuthor {
		return true, nil
	}

	isCollaborator, permissionID, err := uc.repo.GetCollaboratorPermission(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("error checking collaborator permission: %w", err)
	}

	return isCollaborator && permissionID >= 2, nil
}
