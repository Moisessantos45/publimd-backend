package post

import (
	"context"
	"fmt"
	"publimd/internal/shared/models"
	"publimd/internal/shared/utils"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

type PostInfo struct {
	ID              uint64    `json:"id" gorm:"column:id"`
	Slug            string    `json:"slug" gorm:"column:slug"`
	Title           string    `json:"title" gorm:"column:title"`
	Content         string    `json:"content" gorm:"column:content"`
	AuthorID        uint64    `json:"author_id" gorm:"column:author_id"`
	Tags            string    `json:"tags" gorm:"column:tags"`
	Category        string    `json:"category" gorm:"column:category"`
	StateID         uint64    `json:"state_id" gorm:"column:state_id"`
	IsCollaborative bool      `json:"is_collaborative" gorm:"column:is_collaborative"`
	PermissionID    uint64    `json:"permission_id" gorm:"column:permission_id"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at"`
}

type PostInfoBasic struct {
	ID        uint64    `json:"id" gorm:"column:id"`
	Slug      string    `json:"slug" gorm:"column:slug"`
	Title     string    `json:"title" gorm:"column:title"`
	Content   string    `json:"content" gorm:"column:content"`
	Author    string    `json:"author" gorm:"column:author"`
	Tags      string    `json:"tags" gorm:"column:tags"`
	Category  string    `json:"category" gorm:"column:category"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

type PaginatedPosts struct {
	Data     []PostInfoBasic   `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type PostInfoGeneric struct {
	ID              uint64 `json:"id" gorm:"column:id"`
	Slug            string `json:"slug" gorm:"column:slug"`
	Title           string `json:"title" gorm:"column:title"`
	Tags            string `json:"tags" gorm:"column:tags"`
	Category        string `json:"category" gorm:"column:category"`
	State           string `json:"state" gorm:"column:state"`
	StateID         uint64 `json:"state_id" gorm:"column:state_id"`
	IsCollaborative bool   `json:"is_collaborative" gorm:"column:is_collaborative"`
	IsVectorized    bool   `json:"is_vectorized" gorm:"column:is_vectorized"`

	CollaboratorPermissionID uint64    `json:"collaborator_permission_id" gorm:"column:collaborator_permission_id"`
	CreatedAt                time.Time `json:"created_at" gorm:"column:created_at"`
}

type PaginatedPostsGeneric struct {
	Data     []PostInfoGeneric `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type PostInfoDetailed struct {
	ID        uint64    `json:"id" gorm:"column:id"`
	Slug      string    `json:"slug" gorm:"column:slug"`
	Title     string    `json:"title" gorm:"column:title"`
	Content   string    `json:"content" gorm:"column:content"`
	Tags      string    `json:"tags" gorm:"column:tags"`
	Category  string    `json:"category" gorm:"column:category"`
	Author    string    `json:"author" gorm:"column:author"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

type PostInfoRecent struct {
	ID        uint64    `json:"id" gorm:"column:id"`
	Slug      string    `json:"slug" gorm:"column:slug"`
	Title     string    `json:"title" gorm:"column:title"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

type PostTrainData struct {
	Embedding  *pgvector.Vector `json:"embedding" gorm:"column:embedding"`
	TargetText string           `json:"target_text" gorm:"column:content_clean"`
}

type PostRepository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetTrainData(ctx context.Context) ([]PostTrainData, error)
	GetAllStates(ctx context.Context) ([]models.StatePost, error)
	GetAllPublic(ctx context.Context, offset int, limit int, query string, embedding []float32) ([]PostInfoBasic, int64, error)
	GetAll(ctx context.Context, userID uint64, offset int, limit int) ([]PostInfoGeneric, int64, error)
	GetAllRecent(ctx context.Context, userID uint64) ([]PostInfoRecent, error)
	Create(ctx context.Context, post *models.Post) error
	GetByID(ctx context.Context, id uint64) (*models.Post, error)
	GetInfoEmbeddingByID(ctx context.Context, id uint64) (*models.PostInfoEmbedding, error)
	GetBasicInfoByID(ctx context.Context, id uint64) (*models.PostInfoBasic, error)
	GetBySlug(ctx context.Context, slug string) (*models.Post, error)
	GetBasicInfoBySlug(ctx context.Context, slug string) (*models.PostInfoBasic, error)
	GetBySlugPublic(ctx context.Context, slug string) (*PostInfoDetailed, error)
	GetBySlugPrivate(ctx context.Context, slug string, userID uint64) (*PostInfo, error)
	Update(ctx context.Context, id uint64, data map[string]any) error
	UpdateState(ctx context.Context, id uint64, stateID uint64) error
	UpdateEmbedding(ctx context.Context, id uint64, embedding any) error
	InsertOutbox(ctx context.Context, event *models.Outbox) error
	MarkEmbeddingPendingAndBumpVersion(ctx context.Context, postID uint64) error
	GetEmbeddingMetaByID(ctx context.Context, postID uint64) (*models.PostEmbeddingMeta, error)
	ClaimPendingOutboxJobs(ctx context.Context, topic string, limit int) ([]models.Outbox, error)
	MarkOutboxDone(ctx context.Context, jobID string) error
	MarkOutboxDoneTx(ctx context.Context, jobID string) error
	RescheduleOutbox(ctx context.Context, jobID string, attempts int, nextTime time.Time, errMsg string) error
	MarkOutboxDead(ctx context.Context, jobID string, errMsg string) error
	UpdateEmbeddingTx(ctx context.Context, postID uint64, vec any) error
	SetEmbeddingReadyTx(ctx context.Context, postID uint64) error
	SetEmbeddingFailed(ctx context.Context, postID uint64, errMsg string) error
	SetEmbeddingProcessing(ctx context.Context, postID uint64) error
}

type PostService interface {
	GetTrainData(ctx context.Context) ([]PostTrainData, error)
	GetAllStates(ctx context.Context) ([]models.StatePost, error)
	GetAllPublic(ctx context.Context, offset int, limit int, query string) (*PaginatedPosts, error)
	GetAll(ctx context.Context, authID uint64, offset int, limit int) (*PaginatedPostsGeneric, error)
	GetAllRecent(ctx context.Context, userID uint64) ([]PostInfoRecent, error)
	Create(ctx context.Context, authID uint64, post *models.Post) error
	GetByID(ctx context.Context, authID uint64) (*models.Post, error)
	GetBasicInfoBySlug(ctx context.Context, slug string) (*models.PostInfoBasic, error)
	GetBySlugPublic(ctx context.Context, slug string) (*PostInfoDetailed, error)
	GetBySlugPrivate(ctx context.Context, slug string, userID uint64) (*PostInfo, error)
	Update(ctx context.Context, id uint64, authID uint64, post *models.Post) error
	UpdateState(ctx context.Context, id uint64, authID uint64, stateID uint64) error
	UpdateEmbedding(ctx context.Context, authID uint64, slug string) error
}

func generateSlug(title string) string {
	slugNormalized := utils.NormalizeText(title)
	slugBase := strings.ToLower(strings.TrimSpace(slugNormalized))
	slugBase = strings.ReplaceAll(slugBase, " ", "-")

	suffix := fmt.Sprintf("-%d", time.Now().UnixNano())

	const maxSlugLen = 55
	maxBaseLen := maxSlugLen - len(suffix)
	if maxBaseLen < 1 {
		maxBaseLen = 1
	}

	if len(slugBase) > maxBaseLen {
		slugBase = slugBase[:maxBaseLen]
		slugBase = strings.TrimRight(slugBase, "-")
	}

	return slugBase + suffix
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func compareStrings(a string, b string) bool {
	if strings.TrimSpace(a) == "" && strings.TrimSpace(b) == "" {
		return true
	}

	return utils.NormalizeText(a) == utils.NormalizeText(b)
}

func NewPost(data *models.Post) (*models.Post, error) {
	if data.AuthorID == 0 {
		return nil, fmt.Errorf("author_id is required")
	}

	if data.StateID == 0 {
		return nil, fmt.Errorf("state_id is required")
	}

	if strings.TrimSpace(data.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}

	if strings.TrimSpace(data.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	if len(data.Tags) == 0 {
		return nil, fmt.Errorf("tags is required")
	}

	if strings.TrimSpace(data.Category) == "" {
		return nil, fmt.Errorf("category is required")
	}

	data.Slug = generateSlug(data.Title)

	post := &models.Post{
		Slug:         data.Slug,
		Title:        utils.NormalizeText(data.Title),
		Content:      data.Content,
		AuthorID:     data.AuthorID,
		Tags:         data.Tags,
		Category:     data.Category,
		StateID:      data.StateID,
		ContentClean: utils.CleanMarkdownForSearch(data.Content, false),
	}

	return post, nil
}

func BuildPostUpdateData(data *models.Post, changeTitle bool) map[string]any {
	updateData := make(map[string]any)

	if strings.TrimSpace(data.Title) != "" && !changeTitle {
		updateData["title"] = utils.NormalizeText(data.Title)
		updateData["slug"] = generateSlug(data.Title)
	}

	if strings.TrimSpace(data.Content) != "" {
		updateData["content"] = data.Content
		updateData["content_clean"] = utils.CleanMarkdownForSearch(data.Content, false)
	}

	if len(data.Tags) > 0 {
		updateData["tags"] = data.Tags
	}

	if strings.TrimSpace(data.Category) != "" {
		updateData["category"] = data.Category
	}

	return updateData
}
