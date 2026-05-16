package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"publimd/internal/shared/embeddings"
	"publimd/internal/shared/models"

	"github.com/pgvector/pgvector-go"
)

type embeddingJobPayload struct {
	PostID           uint64 `json:"post_id"`
	EmbeddingVersion uint64 `json:"embedding_version"`
}

type OutboxWorker struct {
	repo PostRepository
	cl   *embeddings.Client
}

func NewOutboxWorker(repo PostRepository, cl *embeddings.Client) *OutboxWorker {
	return &OutboxWorker{repo: repo, cl: cl}
}

func (w *OutboxWorker) Run(ctx context.Context) error {
	log.Println("[worker] outbox worker started")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] outbox worker stopping:", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			if err := w.processBatch(ctx, 20); err != nil {
				log.Printf("[worker] batch error: %v", err)
			}
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context, limit int) error {
	jobs, err := w.repo.ClaimPendingOutboxJobs(ctx, "post.embedding.generate", limit)
	if err != nil {
		return fmt.Errorf("claim jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	log.Printf("[worker] claimed %d job(s) from outbox", len(jobs))

	for _, job := range jobs {
		if err := w.processOne(ctx, job); err != nil {
			log.Printf("[worker] job %s failed: %v", job.ID, err)
		}
	}

	return nil
}

func (w *OutboxWorker) processOne(ctx context.Context, job models.Outbox) error {
	log.Printf("[worker] processing job id=%s aggregate_id=%d attempts=%d",
		job.ID, job.AggregateID, job.Attempts)

	var payload embeddingJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		log.Printf("[worker] job %s has invalid payload, marking dead", job.ID)
		return w.repo.MarkOutboxDead(ctx, job.ID, "invalid payload: "+err.Error())
	}

	log.Printf("[worker] job %s → post_id=%d embedding_version=%d",
		job.ID, payload.PostID, payload.EmbeddingVersion)

	if err := w.repo.SetEmbeddingProcessing(ctx, payload.PostID); err != nil {
		return w.retryOrFail(ctx, job, fmt.Errorf("set processing: %w", err))
	}

	post, err := w.repo.GetInfoEmbeddingByID(ctx, payload.PostID)
	if err != nil {
		return w.retryOrFail(ctx, job, fmt.Errorf("get post: %w", err))
	}

	if post.EmbeddingVersion != payload.EmbeddingVersion {
		log.Printf("[worker] job %s outdated (post version=%d payload version=%d), skipping",
			job.ID, post.EmbeddingVersion, payload.EmbeddingVersion)
		return w.repo.MarkOutboxDone(ctx, job.ID)
	}

	log.Printf("[worker] job %s calling embedding API for post_id=%d", job.ID, payload.PostID)

	req := embeddings.NewPostEmbeddingRequest(
		post.ID,
		post.Title,
		post.ContentClean,
		post.Tags,
		post.Category,
	)

	resp, err := w.cl.GeneratePostEmbedding(ctx, req)
	if err != nil {
		log.Printf("[worker] job %s embedding API error: %v", job.ID, err)
		_ = w.repo.SetEmbeddingFailed(ctx, payload.PostID, err.Error())
		return w.retryOrFail(ctx, job, fmt.Errorf("generate embedding: %w", err))
	}

	log.Printf("[worker] job %s received embedding vector, saving to post_id=%d", job.ID, payload.PostID)

	vec := pgvector.NewVector(resp.Embedding)

	err = w.repo.WithTransaction(func(repo *PostgresRepository) error {
		meta, err := repo.GetEmbeddingMetaByID(ctx, payload.PostID)
		if err != nil {
			return err
		}

		if meta.EmbeddingVersion != payload.EmbeddingVersion {
			log.Printf("[worker] job %s version changed during API call, discarding", job.ID)
			return repo.MarkOutboxDoneTx(ctx, job.ID)
		}

		if err := repo.UpdateEmbeddingTx(ctx, payload.PostID, vec); err != nil {
			return err
		}

		if err := repo.SetEmbeddingReadyTx(ctx, payload.PostID); err != nil {
			return err
		}

		return repo.MarkOutboxDoneTx(ctx, job.ID)
	})

	if err != nil {
		return w.retryOrFail(ctx, job, err)
	}

	log.Printf("[worker] job %s done: post_id=%d embedding saved", job.ID, payload.PostID)
	return nil
}

func (w *OutboxWorker) retryOrFail(ctx context.Context, job models.Outbox, err error) error {
	const maxAttempts = 8
	nextAttempts := job.Attempts + 1

	if nextAttempts >= maxAttempts {
		log.Printf("[worker] job %s exhausted %d attempts, marking dead: %v",
			job.ID, maxAttempts, err)
		return w.repo.MarkOutboxDead(ctx, job.ID, err.Error())
	}

	shift := min(nextAttempts, 6)
	backoff := time.Duration(1<<shift) * time.Minute
	nextTime := time.Now().Add(backoff)

	log.Printf("[worker] job %s retry %d/%d in %s: %v",
		job.ID, nextAttempts, maxAttempts, backoff, err)

	return w.repo.RescheduleOutbox(ctx, job.ID, nextAttempts, nextTime, err.Error())
}
