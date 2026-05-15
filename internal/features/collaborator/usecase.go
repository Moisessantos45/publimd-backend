package collaborator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"publimd/internal/features/permissions"
	"publimd/internal/shared/models"
	"publimd/internal/shared/templates"
	"publimd/internal/shared/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

type CollaboratorUsecase struct {
	repo       CollaboratorRepository
	rd         *redis.Client
	mk         *utils.PasetoMaker
	checker    permissions.PostPermissionChecker
	postReader PostSlugResolver
}

func NewCollaboratorUsecase(
	repo CollaboratorRepository,
	rd *redis.Client,
	mk *utils.PasetoMaker,
	checker permissions.PostPermissionChecker,
	postReader PostSlugResolver,
) CollaboratorService {
	return &CollaboratorUsecase{repo: repo, rd: rd, mk: mk, checker: checker, postReader: postReader}
}

func (uc *CollaboratorUsecase) sendInviteEmail(ctx context.Context, postID uint64, targetUserID uint64) error {
	isProduction := os.Getenv("GO_ENV")
	host := os.Getenv("HOST_URL_PROD")
	if isProduction == "dev" {
		host = os.Getenv("HOST_URL_DEV")
	}

	info, err := uc.repo.GetCollaboratorInviteInfo(ctx, postID, targetUserID)
	if err != nil {
		return fmt.Errorf("error fetching invite info: %w", err)
	}

	token, payload, err := uc.mk.NewInviteToken(fmt.Sprintf("%d:%d", postID, targetUserID), 8*time.Hour)
	if err != nil {
		return fmt.Errorf("error generating invite token: %w", err)
	}

	inviteKey := fmt.Sprintf("invite:%s", payload.ID)
	if err := uc.rd.Set(ctx, inviteKey, token, 8*time.Hour).Err(); err != nil {
		return fmt.Errorf("error caching invite token: %w", err)
	}

	renderer, err := templates.NewEmailRenderer()
	if err != nil {
		return err
	}

	data := templates.InvitePostData{
		Name:       info.Name,
		PostTitle:  info.PostTitle,
		InviteLink: fmt.Sprintf("%s/invite/%s", host, token),
	}

	htmlContent, err := renderer.RenderInvitePost(data)
	if err != nil {
		return err
	}

	return utils.EnqueueEmail([]string{info.Email}, "Invitación a colaborar en un post", htmlContent)
}

func (uc *CollaboratorUsecase) resolveUserID(ctx context.Context, authID uint64) (uint64, error) {
	userInfo, err := uc.repo.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return 0, fmt.Errorf("error fetching user info: %w", err)
	}

	return userInfo.ID, nil
}

func (uc *CollaboratorUsecase) GetAllPermissions(ctx context.Context, authID uint64, slug string) ([]models.Permission, error) {
	log.Printf("GetAllPermissions called with authID: %d, slug: %s", authID, slug)
	authorID, err := uc.resolveUserID(ctx, authID)
	if err != nil {
		return nil, err
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("error fetching post: %w", err)
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postInfo.ID, authorID)
	if err != nil {
		return nil, fmt.Errorf("error checking author: %w", err)
	}

	isCollaborator, _, err := uc.repo.GetCollaboratorPermission(ctx, postInfo.ID, authorID)
	if err != nil {
		return nil, fmt.Errorf("error checking collaborator: %w", err)
	}

	if !isAuthor && !isCollaborator {
		return nil, fmt.Errorf("access denied")
	}

	return uc.repo.GetAllPermissions(ctx)
}

func (uc *CollaboratorUsecase) GetAll(ctx context.Context, authID uint64, slug string) (*PostCollaboratorsData, error) {
	userID, err := uc.resolveUserID(ctx, authID)
	if err != nil {
		return nil, err
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("error fetching post: %w", err)
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postInfo.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking author: %w", err)
	}

	isCollaborator, _, err := uc.repo.GetCollaboratorPermission(ctx, postInfo.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking collaborator: %w", err)
	}

	if !isAuthor && !isCollaborator {
		return nil, fmt.Errorf("access denied")
	}

	return uc.repo.GetAll(ctx, postInfo.ID, userID)
}

func (uc *CollaboratorUsecase) GetAllUserInfoBasic(ctx context.Context, authID uint64, slug string) ([]UserBasicInfo, error) {
	userID, err := uc.resolveUserID(ctx, authID)
	if err != nil {
		return nil, err
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("error fetching post: %w", err)
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postInfo.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking author: %w", err)
	}

	isCollaborator, _, err := uc.repo.GetCollaboratorPermission(ctx, postInfo.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("error checking collaborator: %w", err)
	}

	if !isAuthor && !isCollaborator {
		return nil, fmt.Errorf("access denied")
	}

	return uc.repo.GetAllUserInfoBasic(ctx, postInfo.ID)
}

func (uc *CollaboratorUsecase) GetUserByEmail(ctx context.Context, email string, authID uint64, slug string) (*CollaboratorInviteRequest, error) {
	authorID, err := uc.resolveUserID(ctx, authID)
	if err != nil {
		return nil, err
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("error fetching post: %w", err)
	}

	isAuthor, err := uc.repo.IsAuthor(ctx, postInfo.ID, authorID)
	if err != nil {
		return nil, fmt.Errorf("error checking author: %w", err)
	}

	isCollaborator, _, err := uc.repo.GetCollaboratorPermission(ctx, postInfo.ID, authorID)
	if err != nil {
		return nil, fmt.Errorf("error checking collaborator: %w", err)
	}

	if !isAuthor && !isCollaborator {
		return nil, fmt.Errorf("access denied")
	}

	return uc.repo.GetUserByEmail(ctx, email)
}

func (uc *CollaboratorUsecase) Create(
	ctx context.Context,
	authID uint64,
	slug string,
	targetUserID uint64,
	permissionID uint64,
) error {
	if authID == 0 || slug == "" || targetUserID == 0 || permissionID == 0 {
		return fmt.Errorf("invalid parameters")
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("error fetching post: %w", err)
	}

	allowed, err := uc.checker.CanManagePost(ctx, authID, postInfo.ID)
	if err != nil {
		return err
	}

	if !allowed {
		return fmt.Errorf("only the author or a collaborator with manage permission can add collaborators")
	}

	col := &models.Collaborator{
		UserID:       targetUserID,
		PostID:       postInfo.ID,
		PermissionID: permissionID,
		Confirmed:    false,
	}

	if err := uc.repo.Create(ctx, col); err != nil {
		return err
	}

	return uc.sendInviteEmail(ctx, postInfo.ID, targetUserID)
}

func (uc *CollaboratorUsecase) ConfirmInvitation(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("token inválido")
	}

	payload, err := uc.mk.VerifyToken(token)
	if err != nil {
		return fmt.Errorf("token inválido o expirado")
	}

	var postID, userID uint64
	if _, err := fmt.Sscanf(payload.UserID, "%d:%d", &postID, &userID); err != nil {
		return fmt.Errorf("token con formato inválido")
	}

	inviteKey := fmt.Sprintf("invite:%s", payload.ID)

	storedToken, err := uc.rd.Get(ctx, inviteKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("invitación expirada o ya utilizada")
		}
		return fmt.Errorf("error consultando invitación: %w", err)
	}

	if storedToken != token {
		return fmt.Errorf("token inválido")
	}

	if err := uc.repo.ConfirmCollaborator(ctx, postID, userID); err != nil {
		return fmt.Errorf("error confirming invitation: %w", err)
	}

	if err := uc.rd.Del(ctx, inviteKey).Err(); err != nil {
		return fmt.Errorf("colaborador confirmado, pero no se pudo invalidar la invitación: %w", err)
	}

	return nil
}

func (uc *CollaboratorUsecase) ResendInvitation(ctx context.Context, authID uint64, slug string, targetUserID uint64) error {
	if authID == 0 || slug == "" || targetUserID == 0 {
		return fmt.Errorf("invalid parameters")
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("error fetching post: %w", err)
	}

	allowed, err := uc.checker.CanManagePost(ctx, authID, postInfo.ID)
	if err != nil {
		return err
	}

	if !allowed {
		return fmt.Errorf("only the author or a collaborator with manage permission can resend invitations")
	}

	info, err := uc.repo.GetCollaboratorInviteInfo(ctx, postInfo.ID, targetUserID)
	if err != nil {
		return fmt.Errorf("collaborator not found")
	}

	if info.Confirmed {
		return fmt.Errorf("collaborator has already confirmed the invitation")
	}

	return uc.sendInviteEmail(ctx, postInfo.ID, targetUserID)
}

func (uc *CollaboratorUsecase) UpdatePermission(
	ctx context.Context,
	authID uint64,
	slug string,
	targetUserID uint64,
	permissionID uint64,
) error {
	if authID == 0 || slug == "" || targetUserID == 0 {
		return fmt.Errorf("invalid parameters")
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("error fetching post: %w", err)
	}

	allowed, err := uc.checker.CanUpdatePermissions(ctx, authID, postInfo.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("only the post author can update collaborator permissions")
	}

	if permissionID == 0 {
		return uc.repo.Delete(ctx, postInfo.ID, targetUserID)
	}

	return uc.repo.UpdatePermission(ctx, postInfo.ID, targetUserID, permissionID)
}

func (uc *CollaboratorUsecase) Delete(
	ctx context.Context,
	authID uint64,
	slug string,
	targetUserID uint64,
) error {
	if authID == 0 || slug == "" || targetUserID == 0 {
		return fmt.Errorf("invalid parameters")
	}

	postInfo, err := uc.postReader.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("error fetching post: %w", err)
	}

	allowed, err := uc.checker.CanManagePost(ctx, authID, postInfo.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("only the author or a collaborator with manage permission can delete collaborators")
	}

	return uc.repo.Delete(ctx, postInfo.ID, targetUserID)
}
