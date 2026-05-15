package permissions

import "context"

type PermissionRepository interface {
	GetBasicInfoByID(ctx context.Context, id uint64) (uint64, error)
	GetCollaboratorPermission(ctx context.Context, postID uint64, userID uint64) (bool, uint64, error)
	IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error)
}

type PostPermissionChecker interface {
	CanReadContent(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error)
	CanEditContent(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error)
	CanManagePost(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error)
	CanUpdatePermissions(ctx context.Context, actorAuthID uint64, postID uint64) (bool, error)
	IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error)
}
