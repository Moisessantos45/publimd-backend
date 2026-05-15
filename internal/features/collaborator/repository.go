package collaborator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"publimd/internal/shared/models"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) CollaboratorRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) IsAuthor(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Post{}).
		Where("id = ? AND author_id = ?", postID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostgresRepository) GetCollaboratorPermission(
	ctx context.Context,
	postID uint64,
	userID uint64,
) (bool, uint64, error) {
	var collaborator models.Collaborator

	err := r.db.WithContext(ctx).
		Select("permission_id").
		Where("post_id = ? AND user_id = ? AND confirmed = true", postID, userID).
		Take(&collaborator).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	return true, collaborator.PermissionID, nil
}

func (r *PostgresRepository) GetCollaboratorInviteInfo(ctx context.Context, postID uint64, userID uint64) (*CollaboratorInviteInfo, error) {
	var info CollaboratorInviteInfo
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.user_id, u.name, a.email, p.title AS post_title, c.confirmed
		FROM collaborators c
		JOIN users u ON c.user_id = u.id
		JOIN auths a ON u.auth_id = a.id
		JOIN posts p ON c.post_id = p.id
		WHERE c.post_id = ? AND c.user_id = ?
	`, postID, userID).Scan(&info).Error
	if err != nil {
		return nil, err
	}
	if info.UserID == 0 {
		return nil, fmt.Errorf("collaborator not found")
	}

	return &info, nil
}

func (r *PostgresRepository) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.WithContext(ctx).Model(&models.Permission{}).Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	return permissions, nil
}

func (r *PostgresRepository) GetAll(ctx context.Context, postID uint64, userID uint64) (*PostCollaboratorsData, error) {
	var collaborators []CollaboratorInfoBasic
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT c.user_id, u.name AS username,a.email, u.avatar, c.confirmed,c.permission_id
			FROM collaborators c
			JOIN users u ON c.user_id = u.id
			JOIN auths a ON u.auth_id = a.id
			WHERE c.post_id = ?
		`, postID).
		Scan(&collaborators).Error
	if err != nil {
		return nil, err
	}

	var currentUserInfo struct {
		IsAuthor     bool   `gorm:"column:is_author"`
		PermissionID uint64 `gorm:"column:permission_id"`
	}

	err = r.db.WithContext(ctx).
		Raw(`
			SELECT
				(author_id = ?) AS is_author,
				COALESCE((
					SELECT permission_id
					FROM collaborators
					WHERE post_id = p.id AND user_id = ?
					LIMIT 1
				), 4) AS permission_id
			FROM posts p
			WHERE p.id = ?
		`, userID, userID, postID).
		Scan(&currentUserInfo).Error
	if err != nil {
		return nil, err
	}

	log.Printf("Colaboradores encontrados para postID %d: %d", postID, len(collaborators))

	return &PostCollaboratorsData{
		IsAuthor:      currentUserInfo.IsAuthor,
		PermissionID:  currentUserInfo.PermissionID,
		Collaborators: collaborators,
	}, nil
}

func (r *PostgresRepository) GetAllUserInfoBasic(ctx context.Context, postID uint64) ([]UserBasicInfo, error) {
	var users []UserBasicInfo
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT u.id AS user_id, u.avatar
			FROM collaborators c
			JOIN users u ON c.user_id = u.id
			WHERE c.post_id = ?
			LIMIT 4
		`, postID).
		Scan(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*CollaboratorInviteRequest, error) {
	var user CollaboratorInviteRequest
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT a.id, u.name, a.email
			FROM auths a
			JOIN users u ON a.id = u.auth_id
			WHERE a.email = ? AND a.email_confirmed=true AND a.full_profile=true
		`, email).
		Scan(&user).Error
	if err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, fmt.Errorf("usuario con email %s no encontrado", email)
	}

	return &user, nil
}

func (r *PostgresRepository) GetBasicInfoByID(ctx context.Context, id uint64) (*models.UserBasicInfo, error) {
	var info models.UserBasicInfo
	if err := r.db.WithContext(ctx).Model(&models.User{}).Select("id, auth_id").Where("auth_id = ?", id).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con ID %d no encontrado", id)
		}

		return nil, err
	}

	return &info, nil
}

func (r *PostgresRepository) Create(ctx context.Context, cols *models.Collaborator) error {
	return r.db.WithContext(ctx).Create(&cols).Error
}

func (r *PostgresRepository) ConfirmCollaborator(ctx context.Context, postID uint64, userID uint64) error {
	return r.db.WithContext(ctx).
		Model(&models.Collaborator{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Update("confirmed", true).Error
}

func (r *PostgresRepository) UpdatePermission(ctx context.Context, postID uint64, userID uint64, permissionID uint64) error {
	err := r.db.WithContext(ctx).
		Model(&models.Collaborator{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Update("permission_id", permissionID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, postID uint64, userID uint64) error {
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&models.Collaborator{}).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}
