package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"publimd/internal/features/permissions"
	"publimd/internal/features/user"
	"publimd/internal/shared/embeddings"
	"publimd/internal/shared/models"
	"publimd/internal/shared/utils"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

type PostUseCase struct {
	repo    PostRepository
	ucUser  user.UserService
	checker permissions.PostPermissionChecker
	cl      *embeddings.Client
}

func NewPostUseCase(repo PostRepository, ucUser user.UserService, checker permissions.PostPermissionChecker, cl *embeddings.Client) PostService {
	return &PostUseCase{repo: repo, ucUser: ucUser, checker: checker, cl: cl}
}

func (uc *PostUseCase) GetTrainData(ctx context.Context) ([]PostTrainData, error) {
	return uc.repo.GetTrainData(ctx)
}
func (uc *PostUseCase) GetAllStates(ctx context.Context) ([]models.StatePost, error) {
	return uc.repo.GetAllStates(ctx)
}

func (uc *PostUseCase) GetAll(ctx context.Context, authID uint64, page int, pageSize int) (*PaginatedPostsGeneric, error) {
	if authID == 0 {
		return nil, fmt.Errorf("authID is required")
	}

	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	user, err := uc.ucUser.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %v", err)
	}

	offset := (page - 1) * pageSize
	post, total, err := uc.repo.GetAll(ctx, user.ID, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("error fetching posts: %v", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	if page > totalPages {
		return nil, fmt.Errorf("page %d exceeds total pages %d (total items: %d)", page, totalPages, total)
	}

	result := &PaginatedPostsGeneric{
		Data: post,
		Paginate: models.Pagination{
			Total:      total,
			TotalPages: totalPages,
			Page:       page,
			PageSize:   pageSize,
		},
	}

	return result, nil
}

func (uc *PostUseCase) GetAllPublic(ctx context.Context, page int, pageSize int, query string) (*PaginatedPosts, error) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	var embedding []float32
	if strings.TrimSpace(query) != "" {
		log.Printf("Generating embedding for query: %s", query)
		response, err := uc.cl.GenerateQueryEmbedding(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("error generating query embedding: %v", err)
		}

		embedding = response.Embedding
	}

	offset := (page - 1) * pageSize
	post, total, err := uc.repo.GetAllPublic(ctx, offset, pageSize, query, embedding)
	if err != nil {
		log.Printf("Error fetching public posts: %v", err)
		return nil, fmt.Errorf("error fetching public posts: %v", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	if page > totalPages {
		return nil, fmt.Errorf("page %d exceeds total pages %d (total items: %d)", page, totalPages, total)
	}

	for i := range post {
		post[i].Content = utils.CleanMarkdownForSearch(post[i].Content, false)
	}

	result := &PaginatedPosts{
		Data: post,
		Paginate: models.Pagination{
			Total:      total,
			TotalPages: totalPages,
			Page:       page,
			PageSize:   pageSize,
		},
	}

	return result, nil
}

func (uc *PostUseCase) GetAllRecent(ctx context.Context, authID uint64) ([]PostInfoRecent, error) {
	if authID == 0 {
		return nil, fmt.Errorf("authID is required")
	}

	user, err := uc.ucUser.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %v", err)
	}

	return uc.repo.GetAllRecent(ctx, user.ID)
}

func (uc *PostUseCase) GetByID(ctx context.Context, authID uint64) (*models.Post, error) {
	if authID == 0 {
		return nil, fmt.Errorf("authID is required")
	}

	user, err := uc.ucUser.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %v", err)
	}

	return uc.repo.GetByID(ctx, user.ID)
}

func (uc *PostUseCase) GetBySlugPrivate(ctx context.Context, slug string, authID uint64) (*PostInfo, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("slug is required")
	}

	if authID == 0 {
		return nil, fmt.Errorf("authID is required")
	}

	userInfo, err := uc.ucUser.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("error fetching user info: %v", err)
	}

	post, err := uc.repo.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("error fetching post: %v", err)
	}

	allowed, err := uc.checker.CanReadContent(ctx, authID, post.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking permissions: %v", err)
	}

	if !allowed {
		return nil, fmt.Errorf("only the author or a collaborator with manage permission or higher can access this post")
	}

	return uc.repo.GetBySlugPrivate(ctx, slug, userInfo.ID)
}

func (uc *PostUseCase) GetBySlugPublic(ctx context.Context, slug string) (*PostInfoDetailed, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("slug is required")
	}

	return uc.repo.GetBySlugPublic(ctx, slug)
}

func (uc *PostUseCase) GetBasicInfoBySlug(ctx context.Context, slug string) (*models.PostInfoBasic, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("slug is required")
	}

	return uc.repo.GetBasicInfoBySlug(ctx, slug)
}

func (uc *PostUseCase) Create(ctx context.Context, authID uint64, post *models.Post) error {
	if authID == 0 {
		return fmt.Errorf("authID is required")
	}

	user, err := uc.ucUser.GetBasicInfoByID(ctx, authID)
	if err != nil {
		return fmt.Errorf("error fetching user: %v", err)
	}

	post.AuthorID = user.ID

	newPost, err := NewPost(post)
	if err != nil {
		return fmt.Errorf("error validating post data: %v", err)
	}

	newPost.EmbeddingStatus = "pending"
	newPost.EmbeddingVersion = 1

	err = uc.repo.WithTransaction(func(repo *PostgresRepository) error {
		if err := repo.Create(ctx, newPost); err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"post_id":           newPost.ID,
			"embedding_version": newPost.EmbeddingVersion,
		})
		if err != nil {
			return err
		}

		outbox := &models.Outbox{
			Topic:       "post.embedding.generate",
			AggregateID: newPost.ID,
			Payload:     payload,
			State:       "pending",
			AvailableAt: time.Now(),
		}

		if err := repo.InsertOutbox(ctx, outbox); err != nil {
			return err
		}

		log.Printf("[outbox] job enqueued: post_id=%d embedding_version=%d op=create",
			newPost.ID, newPost.EmbeddingVersion)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error creating post: %v", err)
	}

	*post = *newPost

	return nil
}

func (uc *PostUseCase) Update(ctx context.Context, id uint64, authID uint64, post *models.Post) error {
	if authID == 0 {
		return fmt.Errorf("authID is required")
	}

	if id == 0 {
		return fmt.Errorf("post ID is required")
	}

	allowed, err := uc.checker.CanEditContent(ctx, authID, id)
	if err != nil {
		return fmt.Errorf("error checking permissions: %v", err)
	}

	if !allowed {
		return fmt.Errorf("only the author or a collaborator with write permission or higher can update this post")
	}

	findPost, err := uc.repo.GetBasicInfoByID(ctx, id)
	if err != nil {
		return fmt.Errorf("error fetching post: %v", err)
	}

	isEqualTitle := compareStrings(findPost.Title, post.Title)

	updateData := BuildPostUpdateData(post, isEqualTitle)

	err = uc.repo.WithTransaction(func(repo *PostgresRepository) error {
		if err := repo.Update(ctx, id, updateData); err != nil {
			return err
		}

		if err := repo.MarkEmbeddingPendingAndBumpVersion(ctx, id); err != nil {
			return err
		}

		meta, err := repo.GetEmbeddingMetaByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error fetching embedding meta: %v", err)
		}

		payload, err := json.Marshal(map[string]any{
			"post_id":           id,
			"embedding_version": meta.EmbeddingVersion,
		})
		if err != nil {
			return err
		}

		outbox := &models.Outbox{
			Topic:       "post.embedding.generate",
			AggregateID: id,
			Payload:     payload,
			State:       "pending",
			AvailableAt: time.Now(),
		}

		if err := repo.InsertOutbox(ctx, outbox); err != nil {
			return err
		}

		log.Printf("[outbox] job enqueued: post_id=%d embedding_version=%d op=update",
			id, meta.EmbeddingVersion)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error updating post: %v", err)
	}

	return nil
}

func (uc *PostUseCase) UpdateState(ctx context.Context, id uint64, authID uint64, stateID uint64) error {
	if authID == 0 {
		return fmt.Errorf("authID is required")
	}

	if stateID == 0 {
		return fmt.Errorf("stateID is required")
	}

	allowed, err := uc.checker.CanManagePost(ctx, authID, id)
	if err != nil {
		return fmt.Errorf("error checking permissions: %v", err)
	}

	if !allowed {
		return fmt.Errorf("only the author or a collaborator with manage permission or higher can change the post state")
	}

	return uc.repo.UpdateState(ctx, id, stateID)
}

func (uc *PostUseCase) UpdateEmbedding(ctx context.Context, authID uint64, slug string) error {
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("slug is required")
	}

	post, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("error fetching post: %v", err)
	}

	allowed, err := uc.checker.CanManagePost(ctx, authID, post.ID)
	if err != nil {
		return fmt.Errorf("error checking permissions: %v", err)
	}

	if !allowed {
		return fmt.Errorf("only the author or a collaborator with manage permission or higher can change the post state")
	}

	data := embeddings.NewPostEmbeddingRequest(post.ID, post.Title, post.Content, post.Tags, post.Category)

	response, err := uc.cl.GeneratePostEmbedding(ctx, data)
	if err != nil {
		return fmt.Errorf("error generating embedding: %v", err)
	}

	vec := pgvector.NewVector(response.Embedding)

	return uc.repo.UpdateEmbedding(ctx, post.ID, vec)
}
