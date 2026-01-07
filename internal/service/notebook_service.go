// FILE: internal/service/notebook_service.go
package service

import (
	"context"
	"encoding/json"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

type INotebookService interface {
	GetAll(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllNotebookResponse, error)
	Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error)
	Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNotebookResponse, error)
	Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error)
	Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error
	MoveNotebook(ctx context.Context, userId uuid.UUID, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error)
}

type notebookService struct {
	uowFactory       unitofwork.RepositoryFactory
	publisherService IPublisherService
}

func NewNotebookService(
	uowFactory unitofwork.RepositoryFactory,
	publisherService IPublisherService,
) INotebookService {
	return &notebookService{
		uowFactory:       uowFactory,
		publisherService: publisherService,
	}
}

func (c *notebookService) GetAll(ctx context.Context, userId uuid.UUID) ([]*dto.GetAllNotebookResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	// Fetch Notebooks
	notebooks, err := uow.NotebookRepository().FindAll(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0)
	result := make([]*dto.GetAllNotebookResponse, 0)
	for _, notebook := range notebooks {
		res := dto.GetAllNotebookResponse{
			Id:        notebook.Id,
			Name:      notebook.Name,
			ParentId:  notebook.ParentId,
			CreatedAt: notebook.CreatedAt,
			UpdatedAt: notebook.UpdatedAt,
			Notes:     make([]*dto.GetAllNotebookResponseNote, 0),
		}

		// Prepend to result (Legacy behavior reversed order?)
		// Legacy: result = append([]*dto...{&res}, result...) -> Prepend.
		// Assuming notebooks returned by Repo are ordered by CreatedAt? Mapped behavior.
		// Let's keep prepend if that's what legacy did, or just append if repo handles order.
		// Legacy `GetAll` usually ordered by created_at.
		result = append([]*dto.GetAllNotebookResponse{&res}, result...)
		ids = append(ids, notebook.Id)
	}

	if len(ids) == 0 {
		return result, nil
	}

	// Fetch Notes
	// Filter by NotebookIDs AND UserId (redundant but safe)
	notes, err := uow.NoteRepository().FindAll(ctx,
		specification.ByNotebookIDs{NotebookIDs: ids},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}

	// Map Notes to structure
	for i := 0; i < len(result); i++ {
		for j := 0; j < len(notes); j++ {
			if notes[j].NotebookId == result[i].Id {
				result[i].Notes = append(result[i].Notes, &dto.GetAllNotebookResponseNote{
					Id:        notes[j].Id,
					Title:     notes[j].Title,
					Content:   notes[j].Content,
					CreatedAt: notes[j].CreatedAt,
					UpdatedAt: notes[j].UpdatedAt,
				})
			}
		}
	}

	return result, nil
}

func (c *notebookService) Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	notebook := entity.Notebook{
		Id:        uuid.New(),
		Name:      req.Name,
		ParentId:  req.ParentId,
		UserId:    userId,
		CreatedAt: time.Now(),
	}

	err := uow.NotebookRepository().Create(ctx, &notebook)
	if err != nil {
		return nil, err
	}

	return &dto.CreateNotebookResponse{
		Id: notebook.Id,
	}, nil
}

func (c *notebookService) Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNotebookResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	// Check strictly by ID and UserID
	notebook, err := uow.NotebookRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if notebook == nil {
		// return specific not found error or just nil/err
		return nil, nil // Or throw "not found"
	}

	res := dto.ShowNotebookResponse{
		Id:        notebook.Id,
		Name:      notebook.Name,
		ParentId:  notebook.ParentId,
		CreatedAt: notebook.CreatedAt,
		UpdatedAt: notebook.UpdatedAt,
	}

	return &res, nil
}

func (c *notebookService) Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	// Fetch first to check ownership
	notebook, err := uow.NotebookRepository().FindOne(ctx,
		specification.ByID{ID: req.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if notebook == nil {
		return nil, nil
	} // Not found

	now := time.Now()
	notebook.Name = req.Name
	notebook.UpdatedAt = &now

	if err := uow.NotebookRepository().Update(ctx, notebook); err != nil {
		return nil, err
	}

	// Fetch Notes specifically for embedding update
	// "GetByNotebookIds" logic
	notes, err := uow.NoteRepository().FindAll(ctx,
		specification.ByNotebookID{NotebookID: notebook.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}

	for _, note := range notes {
		msg := dto.PublishEmbedNoteMessage{
			NoteId: note.Id,
		}
		msgJson, _ := json.Marshal(msg)
		// Publisher logic
		if err := c.publisherService.Publish(ctx, msgJson); err != nil {
			return nil, err
		}
	}

	return &dto.UpdateNotebookResponse{
		Id: notebook.Id,
	}, nil
}

func (c *notebookService) Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	// Check ownership
	notebook, err := uow.NotebookRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return err
	}
	if notebook == nil {
		return nil
	} // Not found?

	// Transaction
	if err := uow.Begin(ctx); err != nil {
		return err
	}
	defer uow.Rollback()

	// 1. Delete Notebook
	if err := uow.NotebookRepository().Delete(ctx, id); err != nil {
		return err
	}

	// 2. Delete Note Embeddings by NotebookID
	if err := uow.NoteEmbeddingRepository().DeleteByNotebookId(ctx, id); err != nil {
		return err
	}

	// 3. Nullify Parent ID for child notebooks?
	// Legacy: notebookRepo.NullifyParentById(ctx, id, userId)
	// We need logic: UPDATE notebooks SET parent_id = NULL WHERE parent_id = ? AND user_id = ?
	children, err := uow.NotebookRepository().FindAll(ctx,
		specification.ByParentID{ParentID: &id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err == nil {
		for _, child := range children {
			child.ParentId = nil
			if err := uow.NotebookRepository().Update(ctx, child); err != nil {
				return err
			}
		}
	} else {
		return err
	}

	// 4. Delete Notes
	// Legacy: noteRepo.DeleteByNotebookId(ctx, id, userId)
	// Fetch notes in notebook and delete them?
	// Finding them first allows using Delete(id).
	// But Delete(id) implies 1 by 1.
	// Can we use DeleteByNotebookID? NoteRepo doesn't have it in Contract.
	// Generic Delete doesn't support "Delete WHERE notebook_id=X".
	// Efficient way: Fetch IDs, Delete Loop.

	notes, err := uow.NoteRepository().FindAll(ctx, specification.ByNotebookID{NotebookID: id})
	if err != nil {
		return err
	}

	for _, note := range notes {
		if err := uow.NoteRepository().Delete(ctx, note.Id); err != nil {
			return err
		}
	}

	return uow.Commit()
}

func (c *notebookService) MoveNotebook(ctx context.Context, userId uuid.UUID, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	notebook, err := uow.NotebookRepository().FindOne(ctx,
		specification.ByID{ID: req.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil {
		return nil, err
	}
	if notebook == nil {
		return nil, nil
	}

	if req.ParentId != nil {
		// Check parent ownership
		parent, err := uow.NotebookRepository().FindOne(ctx,
			specification.ByID{ID: *req.ParentId},
			specification.UserOwnedBy{UserID: userId},
		)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, nil
		}
	}

	notebook.ParentId = req.ParentId
	err = uow.NotebookRepository().Update(ctx, notebook)
	if err != nil {
		return nil, err
	}

	return &dto.MoveNotebookResponse{
		Id: req.Id,
	}, nil
}
