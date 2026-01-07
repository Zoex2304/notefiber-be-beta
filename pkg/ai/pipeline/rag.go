package pipeline

import (
	"context"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/llm"
	"ai-notetaking-be/pkg/rag/executor"

	"github.com/google/uuid"
)

// RAGResult contains the result of RAG execution
type RAGResult struct {
	Reply        string
	Citations    []dto.CitationDTO
	SessionState string
}

// RAGPipeline wraps the existing RAG executor for consistent interface
type RAGPipeline struct {
	executor *executor.PipelineExecutor
}

// NewRAGPipeline creates a new RAG pipeline wrapper
func NewRAGPipeline(executor *executor.PipelineExecutor) *RAGPipeline {
	return &RAGPipeline{
		executor: executor,
	}
}

// Execute runs the full 3-phase RAG pipeline
func (p *RAGPipeline) Execute(
	ctx context.Context,
	userId uuid.UUID,
	sessionId uuid.UUID,
	query string,
	history []llm.Message,
	uow unitofwork.UnitOfWork,
) (*RAGResult, error) {

	result, err := p.executor.Execute(ctx, userId, sessionId, query, history, uow)
	if err != nil {
		return nil, err
	}

	return &RAGResult{
		Reply:        result.Reply,
		Citations:    result.Citations,
		SessionState: result.SessionState,
	}, nil
}
