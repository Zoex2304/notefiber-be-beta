// FILE: internal/service/consumer_service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/utils"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
)

type IConsumerService interface {
	Consume(ctx context.Context) error
}

type consumerService struct {
	pubSub            *gochannel.GoChannel
	topicName         string
	uowFactory        unitofwork.RepositoryFactory
	embeddingProvider embedding.EmbeddingProvider
}

func NewConsumerService(
	pubSub *gochannel.GoChannel,
	topicName string,
	uowFactory unitofwork.RepositoryFactory,
	embeddingProvider embedding.EmbeddingProvider,
) IConsumerService {
	return &consumerService{
		pubSub:            pubSub,
		topicName:         topicName,
		uowFactory:        uowFactory,
		embeddingProvider: embeddingProvider,
	}
}

func (cs *consumerService) Consume(ctx context.Context) error {
	messages, err := cs.pubSub.Subscribe(ctx, cs.topicName)
	if err != nil {
		return err
	}

	go func() {
		for msg := range messages {
			cs.processMessage(ctx, msg)
		}
	}()

	return nil
}

func (cs *consumerService) processMessage(ctx context.Context, msg *message.Message) {
	// CRITICAL FIX: Proper error handling without defer Nack
	var payload dto.PublishEmbedNoteMessage
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		log.Printf("[ERROR] Failed to unmarshal message: %v", err)
		msg.Ack() // Ack invalid messages to prevent infinite retry
		return
	}

	log.Printf("[INFO] Processing note embedding for NoteId: %s", payload.NoteId)

	uow := cs.uowFactory.NewUnitOfWork(ctx)

	// Fetch Note (Global, no user restrictions)
	note, err := uow.NoteRepository().FindOne(ctx, specification.ByID{ID: payload.NoteId})
	if err != nil {
		log.Printf("[ERROR] Failed to get note %s: %v", payload.NoteId, err)
		msg.Nack() // Nack for retriable errors
		return
	}
	if note == nil {
		log.Printf("[ERROR] Note not found: %s", payload.NoteId)
		msg.Ack() // Note deleted? Ack.
		return
	}

	// Fetch Notebook
	notebook, err := uow.NotebookRepository().FindOne(ctx, specification.ByID{ID: note.NotebookId})
	if err != nil {
		log.Printf("[ERROR] Failed to get notebook %s: %v", note.NotebookId, err)
		msg.Nack()
		return
	}
	// Notebook might be null if parent check fails? FindOne doesn't fail on null, just returns nil.
	// But note exists, so notebook MUST exist in valid state (FK).
	// Handling nil notebook gracefully
	notebookName := "Unknown"
	if notebook != nil {
		notebookName = notebook.Name
	} else {
		log.Printf("[WARN] Note %s has no notebook (implied id %s)", note.Id, note.NotebookId)
	}

	noteUpdatedAt := "-"
	if note.UpdatedAt != nil {
		noteUpdatedAt = note.UpdatedAt.Format(time.RFC3339)
	}

	content := fmt.Sprintf(`Note Title: %s
Notebook Title: %s

%s

Created At: %s
Updated At: %s`,
		note.Title,
		notebookName,
		note.Content,
		note.CreatedAt.Format(time.RFC3339),
		noteUpdatedAt,
	)

	log.Printf("[INFO] Generating embeddings for note %s (content length: %d)", payload.NoteId, len(content))

	// 1. Split Text
	// ChunkSize: 1500 chars (approx 375 tokens) - Ultra safe for context limits
	// Overlap: 200 chars
	chunks := utils.SplitText(content, 1500, 200)
	log.Printf("[INFO] Content split into %d chunks", len(chunks))

	var newEmbeddings []*entity.NoteEmbedding

	// 2. Process each chunk
	for i, chunk := range chunks {
		res, err := cs.embeddingProvider.Generate(chunk, "RETRIEVAL_DOCUMENT")
		if err != nil {
			log.Printf("[ERROR] Failed to generate embedding for chunk %d of note %s: %v", i, payload.NoteId, err)
			msg.Nack()
			return
		}

		newEmbeddings = append(newEmbeddings, &entity.NoteEmbedding{
			Id:             uuid.New(),
			Document:       chunk, // Store specific chunk
			EmbeddingValue: res.Embedding.Values,
			NoteId:         note.Id,
			ChunkIndex:     i,
			CreatedAt:      time.Now(),
		})
	}

	if err := uow.Begin(ctx); err != nil {
		log.Printf("[ERROR] Failed to begin transaction: %v", err)
		msg.Nack()
		return
	}
	defer uow.Rollback()

	log.Printf("[INFO] Deleting old embeddings for note %s", payload.NoteId)
	if err := uow.NoteEmbeddingRepository().DeleteByNoteId(ctx, note.Id); err != nil {
		log.Printf("[ERROR] Failed to delete old embeddings: %v", err)
		msg.Nack()
		return
	}

	log.Printf("[INFO] Creating %d new embeddings for note %s", len(newEmbeddings), payload.NoteId)
	if len(newEmbeddings) > 0 {
		if err := uow.NoteEmbeddingRepository().CreateBulk(ctx, newEmbeddings); err != nil {
			log.Printf("[ERROR] Failed to create bulk embeddings: %v", err)
			msg.Nack()
			return
		}
	}

	if err := uow.Commit(); err != nil {
		log.Printf("[ERROR] Failed to commit transaction: %v", err)
		msg.Nack()
		return
	}

	log.Printf("[SUCCESS] Note processed: %d chunks for NoteId: %s", len(newEmbeddings), payload.NoteId)
	msg.Ack()
}
