package post

import (
	"context"
	"fmt"
	"publimd/internal/shared/models"
	"strings"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) PostRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) WithTransaction(fn func(repo *PostgresRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) GetTrainData(ctx context.Context) ([]PostTrainData, error) {
	var data []PostTrainData
	err := r.db.WithContext(ctx).Raw(`
		SELECT embedding,content_clean
		FROM posts
	`).Scan(&data).Error

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *PostgresRepository) GetAllStates(ctx context.Context) ([]models.StatePost, error) {
	var states []models.StatePost
	err := r.db.WithContext(ctx).Find(&states).Error
	if err != nil {
		return nil, err
	}
	return states, nil
}

func (r *PostgresRepository) GetAllPublic(
	ctx context.Context,
	offset int,
	limit int,
	query string,
	embedding []float32,
) ([]PostInfoBasic, int64, error) {
	var posts []PostInfoBasic
	var total int64

	query = strings.TrimSpace(query)
	hasQuery := query != ""
	hasEmbedding := len(embedding) > 0

	if !hasQuery && !hasEmbedding {
		baseQuery := r.db.WithContext(ctx).
			Model(&models.Post{}).
			Joins("JOIN users ON users.id = posts.author_id").
			Where("posts.state_id = ?", 2)

		if err := baseQuery.Count(&total).Error; err != nil {
			return nil, 0, err
		}

		err := baseQuery.
			Select(`
				posts.id,
				posts.slug,
				posts.title,
				LEFT(posts.content, 200) AS content,
				users.name AS author,
				posts.tags,
				posts.category,
				posts.created_at
			`).
			Order("posts.created_at DESC").
			Offset(offset).
			Limit(limit).
			Scan(&posts).Error
		if err != nil {
			return nil, 0, err
		}

		return posts, total, nil
	}

	const (
		rrfK            = 60
		candidateLimit  = 100
		semanticMaxDist = 0.85
		fuzzyMinScore   = 0.08
	)

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
			panic(rec)
		}
	}()

	if hasEmbedding {
		efSearch := max(candidateLimit, limit*4)
		if err := tx.Exec(fmt.Sprintf("SET LOCAL hnsw.ef_search = %d", efSearch)).Error; err != nil {
			tx.Rollback()
			return nil, 0, err
		}
	}

	var querySQL string
	var countSQL string
	var args []any
	var countArgs []any

	switch {
	case hasQuery && hasEmbedding:
		vec := pgvector.NewVector(embedding)

		countSQL = `
			WITH fulltext AS (
				SELECT p.id
				FROM posts p
				WHERE p.state_id = 2
				  AND p.search_vector @@ websearch_to_tsquery('spanish_unaccent', ?)
				ORDER BY ts_rank_cd(
					p.search_vector,
					websearch_to_tsquery('spanish_unaccent', ?),
					32
				) DESC, p.created_at DESC
				LIMIT ?
			),
			semantic AS (
				SELECT p.id
				FROM posts p
				WHERE p.state_id = 2
				  AND p.embedding IS NOT NULL
				  AND (p.embedding <=> ?) <= ?
				ORDER BY p.embedding <=> ?, p.created_at DESC
				LIMIT ?
			),
			fuzzy AS (
				SELECT p.id
				FROM posts p
				WHERE p.state_id = 2
				  AND p.fuzzy_short IS NOT NULL
				  AND btrim(p.fuzzy_short) <> ''
				  AND word_similarity(?, p.fuzzy_short) >= ?
				ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
				LIMIT ?
			),
			combined AS (
				SELECT id FROM fulltext
				UNION
				SELECT id FROM semantic
				UNION
				SELECT id FROM fuzzy
			)
			SELECT COUNT(*) FROM combined
		`
		countArgs = []any{
			query, query, candidateLimit,
			vec, semanticMaxDist, vec, candidateLimit,
			query, fuzzyMinScore, query, candidateLimit,
		}

		querySQL = `
			WITH fulltext AS (
				SELECT
					p.id,
					ROW_NUMBER() OVER (
						ORDER BY ts_rank_cd(
							p.search_vector,
							websearch_to_tsquery('spanish_unaccent', ?),
							32
						) DESC,
						p.created_at DESC
					) AS rank
				FROM posts p
				WHERE p.state_id = 2
				  AND p.search_vector @@ websearch_to_tsquery('spanish_unaccent', ?)
				ORDER BY ts_rank_cd(
					p.search_vector,
					websearch_to_tsquery('spanish_unaccent', ?),
					32
				) DESC, p.created_at DESC
				LIMIT ?
			),
			semantic AS (
				SELECT
					p.id,
					ROW_NUMBER() OVER (
						ORDER BY p.embedding <=> ?, p.created_at DESC
					) AS rank
				FROM posts p
				WHERE p.state_id = 2
				  AND p.embedding IS NOT NULL
				  AND (p.embedding <=> ?) <= ?
				ORDER BY p.embedding <=> ?, p.created_at DESC
				LIMIT ?
			),
			fuzzy AS (
				SELECT
					p.id,
					ROW_NUMBER() OVER (
						ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
					) AS rank
				FROM posts p
				WHERE p.state_id = 2
				  AND p.fuzzy_short IS NOT NULL
				  AND btrim(p.fuzzy_short) <> ''
				  AND word_similarity(?, p.fuzzy_short) >= ?
				ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
				LIMIT ?
			),
			rrf AS (
				SELECT id, 1.0 / (? + rank) AS score FROM fulltext
				UNION ALL
				SELECT id, 1.0 / (? + rank) AS score FROM semantic
				UNION ALL
				SELECT id, 1.0 / (? + rank) AS score FROM fuzzy
			),
			ranked AS (
				SELECT id, SUM(score) AS rrf_score
				FROM rrf
				GROUP BY id
			)
			SELECT
				p.id,
				p.slug,
				p.title,
				LEFT(p.content, 200) AS content,
				u.name AS author,
				p.tags,
				p.category,
				p.created_at
			FROM ranked r
			JOIN posts p ON p.id = r.id
			JOIN users u ON u.id = p.author_id
			ORDER BY r.rrf_score DESC, p.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []any{
			query, query, query, candidateLimit,
			vec, vec, semanticMaxDist, vec, candidateLimit,
			query, query, fuzzyMinScore, query, candidateLimit,
			rrfK, rrfK, rrfK,
			limit, offset,
		}

	case hasQuery:
		countSQL = `
			WITH fulltext AS (
				SELECT p.id
				FROM posts p
				WHERE p.state_id = 2
				  AND p.search_vector @@ websearch_to_tsquery('spanish_unaccent', ?)
				ORDER BY ts_rank_cd(
					p.search_vector,
					websearch_to_tsquery('spanish_unaccent', ?),
					32
				) DESC, p.created_at DESC
				LIMIT ?
			),
			fuzzy AS (
				SELECT p.id
				FROM posts p
				WHERE p.state_id = 2
				  AND p.fuzzy_short IS NOT NULL
				  AND btrim(p.fuzzy_short) <> ''
				  AND word_similarity(?, p.fuzzy_short) >= ?
				ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
				LIMIT ?
			),
			combined AS (
				SELECT id FROM fulltext
				UNION
				SELECT id FROM fuzzy
			)
			SELECT COUNT(*) FROM combined
		`
		countArgs = []any{
			query, query, candidateLimit,
			query, fuzzyMinScore, query, candidateLimit,
		}

		querySQL = `
			WITH fulltext AS (
				SELECT
					p.id,
					ROW_NUMBER() OVER (
						ORDER BY ts_rank_cd(
							p.search_vector,
							websearch_to_tsquery('spanish_unaccent', ?),
							32
						) DESC,
						p.created_at DESC
					) AS rank
				FROM posts p
				WHERE p.state_id = 2
				  AND p.search_vector @@ websearch_to_tsquery('spanish_unaccent', ?)
				ORDER BY ts_rank_cd(
					p.search_vector,
					websearch_to_tsquery('spanish_unaccent', ?),
					32
				) DESC, p.created_at DESC
				LIMIT ?
			),
			fuzzy AS (
				SELECT
					p.id,
					ROW_NUMBER() OVER (
						ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
					) AS rank
				FROM posts p
				WHERE p.state_id = 2
				  AND p.fuzzy_short IS NOT NULL
				  AND btrim(p.fuzzy_short) <> ''
				  AND word_similarity(?, p.fuzzy_short) >= ?
				ORDER BY word_similarity(?, p.fuzzy_short) DESC, p.created_at DESC
				LIMIT ?
			),
			rrf AS (
				SELECT id, 1.0 / (? + rank) AS score FROM fulltext
				UNION ALL
				SELECT id, 1.0 / (? + rank) AS score FROM fuzzy
			),
			ranked AS (
				SELECT id, SUM(score) AS rrf_score
				FROM rrf
				GROUP BY id
			)
			SELECT
				p.id,
				p.slug,
				p.title,
				LEFT(p.content, 200) AS content,
				u.name AS author,
				p.tags,
				p.category,
				p.created_at
			FROM ranked r
			JOIN posts p ON p.id = r.id
			JOIN users u ON u.id = p.author_id
			ORDER BY r.rrf_score DESC, p.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []any{
			query, query, query, candidateLimit,
			query, query, fuzzyMinScore, query, candidateLimit,
			rrfK, rrfK,
			limit, offset,
		}

	case hasEmbedding:
		vec := pgvector.NewVector(embedding)

		countSQL = `
			SELECT COUNT(*)
			FROM posts p
			WHERE p.state_id = 2
			  AND p.embedding IS NOT NULL
			  AND (p.embedding <=> ?) <= ?
		`
		countArgs = []any{vec, semanticMaxDist}

		querySQL = `
			SELECT
				p.id,
				p.slug,
				p.title,
				LEFT(p.content, 200) AS content,
				u.name AS author,
				p.tags,
				p.category,
				p.created_at
			FROM posts p
			JOIN users u ON u.id = p.author_id
			WHERE p.state_id = 2
			  AND p.embedding IS NOT NULL
			  AND (p.embedding <=> ?) <= ?
			ORDER BY p.embedding <=> ?, p.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = []any{vec, semanticMaxDist, vec, limit, offset}
	}

	if err := tx.Raw(countSQL, countArgs...).Scan(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err := tx.Raw(querySQL, args...).Scan(&posts).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *PostgresRepository) GetAll(ctx context.Context, userID uint64, offset int, limit int) ([]PostInfoGeneric, int64, error) {
	var posts []PostInfoGeneric
	var total int64

	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT p.id)
		FROM posts p
		JOIN state_posts sp ON sp.id = p.state_id
		LEFT JOIN collaborators c ON c.post_id = p.id AND c.user_id = ? AND c.confirmed=true
		WHERE p.author_id = ? OR c.user_id IS NOT NULL
	`, userID, userID).Scan(&total).Error

	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).Raw(`
		SELECT
		  p.id,
		  p.slug,
		  p.title,
		  p.tags,
		  p.category,
		  sp.name AS state,
		  sp.id AS state_id,
		  CASE WHEN c.user_id IS NOT NULL THEN 1 ELSE 0 END AS is_collaborative,
		  COALESCE(p.embedding IS NOT NULL, false) AS is_vectorized,
		  COALESCE(c.permission_id, 4) AS collaborator_permission_id,
		  p.created_at
		FROM posts p
		JOIN state_posts sp ON sp.id = p.state_id
		LEFT JOIN collaborators c
		  ON c.post_id = p.id
		 AND c.user_id = ? AND c.confirmed=true
		WHERE p.author_id = ? OR c.user_id IS NOT NULL
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, userID, limit, offset).Scan(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *PostgresRepository) GetAllRecent(ctx context.Context, userID uint64) ([]PostInfoRecent, error) {
	var posts []PostInfoRecent
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slug, title, created_at
		FROM posts
		WHERE author_id = ? OR EXISTS (
			SELECT 1 FROM collaborators c
			WHERE c.post_id = posts.id AND c.user_id = ?
		)
		ORDER BY created_at DESC
		LIMIT 5
	`, userID, userID).Scan(&posts).Error

	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uint64) (*models.Post, error) {
	var post models.Post
	err := r.db.WithContext(ctx).Preload("State").First(&post, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetInfoEmbeddingByID(ctx context.Context, id uint64) (*models.PostInfoEmbedding, error) {
	var post models.PostInfoEmbedding

	err := r.db.WithContext(ctx).Raw(`
		SELECT id, title, tags, category, content_clean
		FROM posts
		WHERE id = ?
	`, id).Scan(&post).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	var post models.Post
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetBasicInfoBySlug(ctx context.Context, slug string) (*models.PostInfoBasic, error) {
	var post models.PostInfoBasic
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slug, title
		FROM posts
		WHERE slug = ?
	`, slug).Scan(&post).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetBasicInfoByID(ctx context.Context, id uint64) (*models.PostInfoBasic, error) {
	var post models.PostInfoBasic
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slug, title
		FROM posts
		WHERE id = ?
	`, id).Scan(&post).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetBySlugPrivate(ctx context.Context, slug string, userID uint64) (*PostInfo, error) {
	var post PostInfo
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			p.id,
			p.slug,
			p.title,
			p.content,
			p.tags,
			p.category,
			p.created_at,
			p.state_id,
			EXISTS (
				SELECT 1 FROM collaborators c
				WHERE c.post_id = p.id AND c.user_id = ?
			) AS is_collaborative,
			COALESCE((
				SELECT c.permission_id
				FROM collaborators c
				WHERE c.post_id = p.id AND c.user_id = ?
				LIMIT 1
			), 4) AS permission_id
		FROM posts p
		WHERE p.slug = ?
	`, userID, userID, slug).Scan(&post).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) GetBySlugPublic(ctx context.Context, slug string) (*PostInfoDetailed, error) {
	var post PostInfoDetailed
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			p.id, p.slug, p.title, p.content, p.tags, p.category,p.created_at,
			CONCAT(u.name, ' ', u.last_name) AS author
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.slug = ? AND p.state_id = 2 OR p.state_id = 5
	`, slug).Scan(&post).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostgresRepository) Create(ctx context.Context, post *models.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *PostgresRepository) Update(ctx context.Context, id uint64, data map[string]any) error {
	err := r.db.WithContext(ctx).Model(&models.Post{}).Where("id = ?", id).Updates(data).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}

func (r *PostgresRepository) UpdateEmbedding(ctx context.Context, id uint64, embedding any) error {
	err := r.db.WithContext(ctx).Model(&models.Post{}).Where("id = ?", id).Update("embedding", embedding).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}

func (r *PostgresRepository) UpdateState(ctx context.Context, id uint64, stateID uint64) error {
	err := r.db.WithContext(ctx).Model(&models.Post{}).Where("id = ?", id).Update("state_id", stateID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return nil
}
