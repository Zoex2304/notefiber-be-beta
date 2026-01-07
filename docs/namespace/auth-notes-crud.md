# Dokumentasi Fitur: CRUD Note

> **Fokus Domain:** BACKEND  
> **Konteks:** Trace Upstream ke Downstream secara Semantik

---

## Alur Data Semantik (Scope: BACKEND)

```
=== CREATE NOTE ===
[HTTP POST /api/note/v1]  
    -> [JWT Middleware: Ekstraksi User ID]  
    -> [Controller: Parsing & Validasi]  
    -> [Service: Konstruksi Entity]  
        -> [Repository: Persistensi Note]  
        -> [Publisher: Trigger Embedding Generation (Async)]  
    -> [HTTP Response dengan Note ID]

=== READ NOTE ===
[HTTP GET /api/note/v1/:id]  
    -> [JWT Middleware: Ekstraksi User ID]  
    -> [Controller: Parsing ID]  
    -> [Service: Query dengan Ownership Check]  
        -> [Repository: FindOne dengan Specifications]  
    -> [HTTP Response dengan Detail Note]

=== UPDATE NOTE ===
[HTTP PUT /api/note/v1/:id]  
    -> [JWT Middleware: Ekstraksi User ID]  
    -> [Controller: Parsing & Validasi]  
    -> [Service: Fetch -> Modify -> Persist]  
        -> [Repository: FindOne + Update]  
        -> [Publisher: Trigger Re-Embedding (Async)]  
    -> [HTTP Response dengan Note ID]

=== DELETE NOTE ===
[HTTP DELETE /api/note/v1/:id]  
    -> [JWT Middleware: Ekstraksi User ID]  
    -> [Controller: Parsing ID]  
    -> [Service: Ownership Check -> Transactional Delete]  
        -> [Repository: Delete Note + Delete Embeddings]  
    -> [HTTP Response Success]
```

---

## A. Laporan Implementasi Fitur CRUD Note

### Deskripsi Fungsional

Fitur ini menyediakan manajemen catatan (notes) lengkap untuk pengguna aplikasi NoteFiber. Setiap note terikat pada satu notebook dan satu user, dengan isolasi data yang ketat berdasarkan kepemilikan. Sistem mengimplementasikan:

1. **Create**: Pembuatan note baru dengan trigger otomatis untuk generasi embedding (untuk fitur semantic search)
2. **Read**: Pengambilan detail note dengan validasi kepemilikan
3. **Update**: Modifikasi konten note dengan re-generation embedding
4. **Delete**: Penghapusan note beserta embedding terkait dalam satu transaksi atomik
5. **Move**: Pemindahan note antar notebook
6. **Semantic Search**: Pencarian berbasis AI dengan validasi subscription plan

Semua endpoint dilindungi JWT middleware dan menerapkan prinsip multi-tenancy via [UserOwnedBy](file:///d:/notetaker/notefiber-BE/internal/repository/specification/user_specifications.go#17-20) specification.

### Visualisasi

**Create Note Response:**
```json
{
    "success": true,
    "code": 200,
    "message": "Success create note",
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000"
    }
}
```

**Show Note Response:**
```json
{
    "success": true,
    "code": 200,
    "message": "Success show note",
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "title": "My First Note",
        "content": "This is the content of my note...",
        "notebook_id": "660e8400-e29b-41d4-a716-446655440000",
        "created_at": "2024-12-24T10:00:00Z",
        "updated_at": "2024-12-24T15:30:00Z"
    }
}
```
*Caption: Gambar 1: Struktur JSON Response untuk operasi Create dan Show.*

---

## B. Bedah Arsitektur & Komponen

Berikut adalah rincian 17 komponen yang menyusun fitur ini di sisi BACKEND.

---

### [internal/server/server.go](file:///d:/notetaker/notefiber-BE/internal/server/server.go)
**Layer Terdeteksi:** `HTTP Server & Route Registration`

**Narasi Operasional:**
Komponen ini menginisialisasi instance server HTTP berbasis Fiber dan mendaftarkan seluruh controller. Untuk fitur Note, [NoteController](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go#12-21) didaftarkan pada grup `/api` dengan path prefix `/note/v1`. Semua endpoint note dilindungi oleh JWT middleware yang di-apply pada level grup.

```go
func registerRoutes(app *fiber.App, c *bootstrap.Container) {
	api := app.Group("/api")

	c.AuthController.RegisterRoutes(api)
	c.NotebookController.RegisterRoutes(api)
	c.NoteController.RegisterRoutes(api)
	c.ChatbotController.RegisterRoutes(api)
	// ... other controllers
}
```
*Caption: Snippet 1: Registrasi NoteController ke grup API.*

---

### [internal/bootstrap/container.go](file:///d:/notetaker/notefiber-BE/internal/bootstrap/container.go)
**Layer Terdeteksi:** `Dependency Injection Container`

**Narasi Operasional:**
File ini mengorkestrasi konstruksi dan injeksi dependensi. [NoteService](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go#20-28) diinisialisasi dengan dua dependensi: `uowFactory` (untuk akses repository) dan [publisherService](file:///d:/notetaker/notefiber-BE/internal/service/publisher_service.go#15-20) (untuk trigger embedding generation via message queue). Service ini kemudian diinjeksikan ke [NoteController](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go#12-21).

```go
func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	// 1. Core Facades
	uowFactory := unitofwork.NewRepositoryFactory(db)

	// 2. Event Bus
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, watermillLogger)

	// 3. Services
	publisherService := service.NewPublisherService(cfg.Keys.ExampleTopic, pubSub)
	noteService := service.NewNoteService(uowFactory, publisherService)

	// 4. Controllers
	return &Container{
		NoteController: controller.NewNoteController(noteService),
		// ...
	}
}
```
*Caption: Snippet 2: Konstruksi NoteService dengan Repository Factory dan Publisher.*

---

### [internal/dto/note_dto.go](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go)
**Layer Terdeteksi:** `Data Transfer Object (DTO)`

**Narasi Operasional:**
File ini mendefinisikan kontrak data untuk semua operasi Note. [CreateNoteRequest](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go#9-14) memerlukan `title` dan `notebook_id` dengan validasi required. [UpdateNoteRequest](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go#28-33) menerima `title` dan `content` untuk modifikasi. [MoveNoteRequest](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go#38-42) hanya memerlukan `notebook_id` tujuan. Response DTO mencakup metadata seperti `created_at` dan `updated_at` untuk audit.

```go
type CreateNoteRequest struct {
	Title      string    `json:"title" validate:"required"`
	Content    string    `json:"content"`
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}

type CreateNoteResponse struct {
	Id uuid.UUID `json:"id"`
}

type ShowNoteResponse struct {
	Id         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	NotebookId uuid.UUID  `json:"notebook_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type UpdateNoteRequest struct {
	Id      uuid.UUID
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type MoveNoteRequest struct {
	Id         uuid.UUID
	NotebookId uuid.UUID `json:"notebook_id" validate:"required"`
}
```
*Caption: Snippet 3: Definisi DTO untuk operasi CRUD Note.*

---

### [internal/controller/note_controller.go](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go)
**Layer Terdeteksi:** `Interface / Controller Layer`

**Narasi Operasional:**
Komponen ini menangani siklus Request-Response HTTP untuk semua endpoint Note. Setiap handler mengekstrak `user_id` dari JWT token (disimpan di `ctx.Locals` oleh middleware), mem-parsing parameter dan body request, memvalidasi input, lalu mendelegasikan ke Service. Handler [SemanticSearch](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go#19-20) juga menangani error khusus untuk akses fitur premium (403 Forbidden).

```go
func (c *noteController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/note/v1")
	h.Use(serverutils.JwtMiddleware) // PROTECTED: Wajib login
	h.Get("semantic-search", c.SemanticSearch)
	h.Post("", c.Create)
	h.Get(":id", c.Show)
	h.Put(":id", c.Update)
	h.Put(":id/move", c.MoveNote)
	h.Delete(":id", c.Delete)
}

func (c *noteController) Create(ctx *fiber.Ctx) error {
	// 1. Ambil User ID dari Token
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.CreateNoteRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := serverutils.ValidateRequest(req)
	if err != nil {
		return err
	}

	// 2. Kirim userId ke Service
	res, err := c.noteService.Create(ctx.Context(), userId, &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success create note", res))
}

func (c *noteController) Show(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)
	
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	res, err := c.noteService.Show(ctx.Context(), userId, id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success show note", res))
}

func (c *noteController) Delete(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	err := c.noteService.Delete(ctx.Context(), userId, id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Success delete note", nil))
}
```
*Caption: Snippet 4: Controller dengan ekstraksi User ID dan delegasi ke Service.*

---

### [internal/service/note_service.go](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go)
**Layer Terdeteksi:** `Business Logic / Service Layer`

**Narasi Operasional:**
Komponen ini mengenkapsulasi logika bisnis untuk semua operasi Note. 

**Create**: Konstruksi entity Note dengan User ID, persistensi via repository, dan publish message untuk embedding generation.

**Show**: Query dengan kombinasi specification [ByID](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#11-14) + [UserOwnedBy](file:///d:/notetaker/notefiber-BE/internal/repository/specification/user_specifications.go#17-20) untuk validasi kepemilikan otomatis.

**Update**: Fetch existing note dengan ownership check, modifikasi field, persist, dan trigger re-embedding.

**Delete**: Operasi transaksional yang menghapus Note dan embedding terkait dalam satu atomic operation.

**MoveNote**: Update `notebook_id` dengan ownership validation.

**SemanticSearch**: Fitur premium yang memvalidasi subscription plan sebelum melakukan vector similarity search.

```go
func (c *noteService) Create(ctx context.Context, userId uuid.UUID, req *dto.CreateNoteRequest) (*dto.CreateNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	note := entity.Note{
		Id:         uuid.New(),
		Title:      req.Title,
		Content:    req.Content,
		NotebookId: req.NotebookId,
		UserId:     userId,
		CreatedAt:  time.Now(),
	}

	err := uow.NoteRepository().Create(ctx, &note)
	if err != nil {
		return nil, err
	}

	// Trigger Embedding Generation via Message Queue
	msgPayload := dto.PublishEmbedNoteMessage{NoteId: note.Id}
	msgJson, _ := json.Marshal(msgPayload)
	c.publisherService.Publish(ctx, msgJson)

	return &dto.CreateNoteResponse{Id: note.Id}, nil
}

func (c *noteService) Show(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*dto.ShowNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)
	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId}, // Ownership Check
	)
	if err != nil || note == nil {
		return nil, err
	}

	return &dto.ShowNoteResponse{
		Id: note.Id, Title: note.Title, Content: note.Content,
		NotebookId: note.NotebookId, CreatedAt: note.CreatedAt, UpdatedAt: note.UpdatedAt,
	}, nil
}

func (c *noteService) Update(ctx context.Context, userId uuid.UUID, req *dto.UpdateNoteRequest) (*dto.UpdateNoteResponse, error) {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	note, err := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: req.Id},
		specification.UserOwnedBy{UserID: userId},
	)
	if err != nil || note == nil {
		return nil, err
	}

	now := time.Now()
	note.Title = req.Title
	note.Content = req.Content
	note.UpdatedAt = &now

	uow.NoteRepository().Update(ctx, note)

	// Trigger Re-Embedding
	payload := dto.PublishEmbedNoteMessage{NoteId: note.Id}
	payloadJson, _ := json.Marshal(payload)
	c.publisherService.Publish(ctx, payloadJson)

	return &dto.UpdateNoteResponse{Id: note.Id}, nil
}

func (c *noteService) Delete(ctx context.Context, userId uuid.UUID, id uuid.UUID) error {
	uow := c.uowFactory.NewUnitOfWork(ctx)

	note, _ := uow.NoteRepository().FindOne(ctx,
		specification.ByID{ID: id},
		specification.UserOwnedBy{UserID: userId},
	)
	if note == nil {
		return nil
	}

	// Transactional Delete: Note + Embeddings
	uow.Begin(ctx)
	defer uow.Rollback()

	uow.NoteRepository().Delete(ctx, id)
	uow.NoteEmbeddingRepository().DeleteByNoteId(ctx, id)

	return uow.Commit()
}
```
*Caption: Snippet 5: Service dengan ownership validation, transaction, dan event publishing.*

---

### [internal/service/publisher_service.go](file:///d:/notetaker/notefiber-BE/internal/service/publisher_service.go)
**Layer Terdeteksi:** `Event Publishing / Message Queue`

**Narasi Operasional:**
Komponen ini mengabstraksi pengiriman message ke event bus (Watermill GoChannel). Digunakan oleh NoteService untuk trigger embedding generation secara asynchronous setelah Create/Update. Consumer service yang terpisah akan memproses message ini untuk menghasilkan vector embedding.

```go
type IPublisherService interface {
	Publish(ctx context.Context, payload []byte) error
}

type publisherService struct {
	pubSub    *gochannel.GoChannel
	topicName string
}

func (ps *publisherService) Publish(ctx context.Context, payload []byte) error {
	err := ps.pubSub.Publish(
		ps.topicName,
		message.NewMessage(watermill.NewUUID(), payload),
	)
	return err
}

func NewPublisherService(topicName string, pubSub *gochannel.GoChannel) IPublisherService {
	return &publisherService{topicName: topicName, pubSub: pubSub}
}
```
*Caption: Snippet 6: Publisher service untuk async event publishing.*

---

### [internal/repository/unitofwork/repository_factory.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/repository_factory.go)
**Layer Terdeteksi:** `Factory Interface`

**Narasi Operasional:**
File ini mendefinisikan kontrak untuk pembuatan instance Unit of Work yang digunakan NoteService untuk akses repository.

```go
type RepositoryFactory interface {
	NewUnitOfWork(ctx context.Context) UnitOfWork
}
```
*Caption: Snippet 7: Interface factory untuk pembuatan Unit of Work.*

---

### [internal/repository/unitofwork/unit_of_work.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work.go)
**Layer Terdeteksi:** `Unit of Work Interface`

**Narasi Operasional:**
File ini mendefinisikan kontrak Unit of Work. Untuk Note CRUD, tiga repository digunakan: [NoteRepository](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work.go#16-17) untuk operasi note, [NoteEmbeddingRepository](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work_impl.go#71-74) untuk penghapusan embedding saat delete, dan [SubscriptionRepository](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work_impl.go#87-90) untuk validasi fitur premium pada semantic search.

```go
type UnitOfWork interface {
	Begin(ctx context.Context) error
	Commit() error
	Rollback() error

	NoteRepository() contract.NoteRepository
	NoteEmbeddingRepository() contract.NoteEmbeddingRepository
	SubscriptionRepository() contract.SubscriptionRepository
	// ... other repositories
}
```
*Caption: Snippet 8: Interface Unit of Work dengan akses ke Note dan Embedding repository.*

---

### [internal/repository/unitofwork/unit_of_work_impl.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work_impl.go)
**Layer Terdeteksi:** `Unit of Work Implementation`

**Narasi Operasional:**
Komponen ini mengimplementasikan pola Unit of Work dengan GORM. Untuk operasi Delete, transaksi eksplisit digunakan untuk menjamin atomisitas penghapusan Note dan Embedding.

```go
func (u *UnitOfWorkImpl) Begin(ctx context.Context) error {
	u.tx = u.db.WithContext(ctx).Begin()
	return u.tx.Error
}

func (u *UnitOfWorkImpl) NoteRepository() contract.NoteRepository {
	return implementation.NewNoteRepository(u.getDB())
}

func (u *UnitOfWorkImpl) NoteEmbeddingRepository() contract.NoteEmbeddingRepository {
	return implementation.NewNoteEmbeddingRepository(u.getDB())
}
```
*Caption: Snippet 9: Implementasi Unit of Work dengan transaksi.*

---

### [internal/repository/contract/note_repository.go](file:///d:/notetaker/notefiber-BE/internal/repository/contract/note_repository.go)
**Layer Terdeteksi:** `Repository Interface / Contract`

**Narasi Operasional:**
File ini mendefinisikan kontrak standar untuk operasi data Note: [Create](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go#43-66), [Update](file:///d:/notetaker/notefiber-BE/internal/repository/contract/user_repository.go#14-15), [Delete](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go#142-171), [FindOne](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go#58-69), [FindAll](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go#70-78), dan [Count](file:///d:/notetaker/notefiber-BE/internal/repository/contract/note_repository.go#18-19). Semua metode query menerima variadic specifications untuk membangun query secara deklaratif.

```go
type NoteRepository interface {
	Create(ctx context.Context, note *entity.Note) error
	Update(ctx context.Context, note *entity.Note) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Note, error)
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Note, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
}
```
*Caption: Snippet 10: Kontrak repository CRUD untuk Note.*

---

### [internal/repository/specification/common_specifications.go](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go)
**Layer Terdeteksi:** `Common Specification Implementation`

**Narasi Operasional:**
File ini menyediakan specification generik yang digunakan lintas domain. [ByID](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#11-14) memfilter berdasarkan primary key, [ByIDs](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#20-23) untuk multiple IDs (semantic search), [OrderBy](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#29-33) untuk sorting, dan [Pagination](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#52-56) untuk limit/offset. Specifications ini dikomposisikan dengan specification domain-specific.

```go
type ByID struct {
	ID uuid.UUID
}

func (s ByID) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("id = ?", s.ID)
}

type ByIDs struct {
	IDs []uuid.UUID
}

func (s ByIDs) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("id IN ?", s.IDs)
}

type Pagination struct {
	Limit  int
	Offset int
}

func (s Pagination) Apply(db *gorm.DB) *gorm.DB {
	return db.Limit(s.Limit).Offset(s.Offset)
}
```
*Caption: Snippet 11: Specification generik untuk query building.*

---

### [internal/repository/specification/note_specifications.go](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go)
**Layer Terdeteksi:** `Domain Specification Implementation`

**Narasi Operasional:**
File ini menyediakan specification khusus domain Note. [ByNotebookID](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go#8-11) memfilter notes berdasarkan notebook parent, [ByNotebookIDs](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go#16-19) untuk multiple notebooks. Specification [UserOwnedBy](file:///d:/notetaker/notefiber-BE/internal/repository/specification/user_specifications.go#17-20) (dari user_specifications) digunakan untuk ownership validation.

```go
type ByNotebookID struct {
	NotebookID uuid.UUID
}

func (s ByNotebookID) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("notebook_id = ?", s.NotebookID)
}

type ByNotebookIDs struct {
	NotebookIDs []uuid.UUID
}

func (s ByNotebookIDs) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("notebook_id IN ?", s.NotebookIDs)
}
```
*Caption: Snippet 12: Specification untuk filter berdasarkan Notebook.*

---

### [internal/repository/implementation/note_repository_impl.go](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go)
**Layer Terdeteksi:** `Repository Implementation`

**Narasi Operasional:**
Komponen ini mengimplementasikan kontrak [NoteRepository](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work.go#16-17) dengan GORM. Setiap operasi menggunakan Mapper untuk transformasi Entity-Model. [FindOne](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go#58-69) dan [FindAll](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go#70-78) menerapkan specifications secara iteratif. Delete menggunakan soft-delete bawaan GORM melalui field `DeletedAt`.

```go
func (r *NoteRepositoryImpl) Create(ctx context.Context, note *entity.Note) error {
	m := r.mapper.ToModel(note)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	*note = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteRepositoryImpl) Update(ctx context.Context, note *entity.Note) error {
	m := r.mapper.ToModel(note)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	*note = *r.mapper.ToEntity(m)
	return nil
}

func (r *NoteRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Note{}, id).Error
}

func (r *NoteRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.Note, error) {
	var m model.Note
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.ToEntity(&m), nil
}

func (r *NoteRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.Note, error) {
	var models []*model.Note
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToEntities(models), nil
}
```
*Caption: Snippet 13: Implementasi repository CRUD dengan specification pattern.*

---

### [internal/entity/note_entity.go](file:///d:/notetaker/notefiber-BE/internal/entity/note_entity.go)
**Layer Terdeteksi:** `Domain Entity`

**Narasi Operasional:**
File ini mendefinisikan entity [Note](file:///d:/notetaker/notefiber-BE/internal/model/note_model.go#10-20) yang merepresentasikan konsep bisnis catatan. Atribut kunci mencakup relasi ke `NotebookId` dan `UserId` untuk isolasi data, serta metadata audit (`CreatedAt`, `UpdatedAt`, `DeletedAt`).

```go
type Note struct {
	Id         uuid.UUID
	Title      string
	Content    string
	NotebookId uuid.UUID
	UserId     uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	DeletedAt  *time.Time
	IsDeleted  bool
}
```
*Caption: Snippet 14: Entity domain untuk Note.*

---

### [internal/model/note_model.go](file:///d:/notetaker/notefiber-BE/internal/model/note_model.go)
**Layer Terdeteksi:** `Database Model (ORM)`

**Narasi Operasional:**
Model [Note](file:///d:/notetaker/notefiber-BE/internal/model/note_model.go#10-20) dipetakan ke tabel `notes` dengan konfigurasi kolom GORM. Indeks pada `notebook_id` dan `user_id` mempercepat query filter. Soft-delete diaktifkan via `gorm.DeletedAt` yang secara otomatis mengeksklusi record yang dihapus dari query normal.

```go
type Note struct {
	Id         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title      string         `gorm:"type:varchar(255);not null"`
	Content    string         `gorm:"type:text"`
	NotebookId uuid.UUID      `gorm:"type:uuid;not null;index"`
	UserId     uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (Note) TableName() string {
	return "notes"
}
```
*Caption: Snippet 15: Model ORM dengan soft-delete support.*

---

### [internal/mapper/note_mapper.go](file:///d:/notetaker/notefiber-BE/internal/mapper/note_mapper.go)
**Layer Terdeteksi:** `Data Mapper`

**Narasi Operasional:**
Komponen ini menyediakan transformasi bidirectional antara Entity dan Model untuk Note. Perhatian khusus diberikan pada konversi field nullable: `DeletedAt` dan `UpdatedAt` di-handle dengan null-check untuk menghindari nil pointer dereference.

```go
func (m *NoteMapper) ToEntity(n *model.Note) *entity.Note {
	if n == nil {
		return nil
	}

	var deletedAt *time.Time
	if n.DeletedAt.Valid {
		t := n.DeletedAt.Time
		deletedAt = &t
	}

	var updatedAt *time.Time
	if !n.UpdatedAt.IsZero() {
		t := n.UpdatedAt
		updatedAt = &t
	}

	return &entity.Note{
		Id:         n.Id,
		Title:      n.Title,
		Content:    n.Content,
		NotebookId: n.NotebookId,
		UserId:     n.UserId,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  updatedAt,
		DeletedAt:  deletedAt,
		IsDeleted:  n.DeletedAt.Valid,
	}
}

func (m *NoteMapper) ToModel(n *entity.Note) *model.Note {
	if n == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if n.DeletedAt != nil {
		deletedAt = gorm.DeletedAt{Time: *n.DeletedAt, Valid: true}
	}

	return &model.Note{
		Id:         n.Id,
		Title:      n.Title,
		Content:    n.Content,
		NotebookId: n.NotebookId,
		UserId:     n.UserId,
		CreatedAt:  n.CreatedAt,
		DeletedAt:  deletedAt,
	}
}
```
*Caption: Snippet 16: Mapper dengan handling nullable fields.*

---

## C. Ringkasan Layer Arsitektur

| No | Layer | File | Tanggung Jawab |
|----|-------|------|----------------|
| 1 | HTTP Server | [server/server.go](file:///d:/notetaker/notefiber-BE/internal/server/server.go) | Inisialisasi Fiber, route registration |
| 2 | DI Container | [bootstrap/container.go](file:///d:/notetaker/notefiber-BE/internal/bootstrap/container.go) | Dependency wiring dengan Publisher |
| 3 | DTO | [dto/note_dto.go](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go) | Kontrak data CRUD operations |
| 4 | Controller | [controller/note_controller.go](file:///d:/notetaker/notefiber-BE/internal/controller/note_controller.go) | HTTP handler dengan JWT extraction |
| 5 | Service | [service/note_service.go](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go) | Business logic & event publishing |
| 6 | Publisher | [service/publisher_service.go](file:///d:/notetaker/notefiber-BE/internal/service/publisher_service.go) | Async message queue publishing |
| 7 | Factory Interface | [unitofwork/repository_factory.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/repository_factory.go) | Kontrak pembuatan Unit of Work |
| 8 | Factory Impl | [unitofwork/repository_factory_impl.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/repository_factory_impl.go) | Implementasi factory |
| 9 | UoW Interface | [unitofwork/unit_of_work.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work.go) | Kontrak transaksi & akses repository |
| 10 | UoW Impl | [unitofwork/unit_of_work_impl.go](file:///d:/notetaker/notefiber-BE/internal/repository/unitofwork/unit_of_work_impl.go) | Manajemen transaksi GORM |
| 11 | Repository Contract | [contract/note_repository.go](file:///d:/notetaker/notefiber-BE/internal/repository/contract/note_repository.go) | Interface CRUD operations |
| 12 | Common Specs | [specification/common_specifications.go](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go) | [ByID](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#11-14), [ByIDs](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#20-23), [Pagination](file:///d:/notetaker/notefiber-BE/internal/repository/specification/common_specifications.go#52-56) |
| 13 | Note Specs | [specification/note_specifications.go](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go) | [ByNotebookID](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go#8-11), [ByNotebookIDs](file:///d:/notetaker/notefiber-BE/internal/repository/specification/note_specifications.go#16-19) |
| 14 | Repository Impl | [implementation/note_repository_impl.go](file:///d:/notetaker/notefiber-BE/internal/repository/implementation/note_repository_impl.go) | GORM CRUD implementation |
| 15 | Entity | [entity/note_entity.go](file:///d:/notetaker/notefiber-BE/internal/entity/note_entity.go) | Domain object |
| 16 | Model | [model/note_model.go](file:///d:/notetaker/notefiber-BE/internal/model/note_model.go) | Database table mapping |
| 17 | Mapper | [mapper/note_mapper.go](file:///d:/notetaker/notefiber-BE/internal/mapper/note_mapper.go) | Entity â†” Model transformation |

---

## D. Endpoint API Reference

| Method | Endpoint | Deskripsi | Auth |
|--------|----------|-----------|------|
| `POST` | `/api/note/v1` | Create new note | JWT Required |
| `GET` | `/api/note/v1/:id` | Get note by ID | JWT Required |
| `PUT` | `/api/note/v1/:id` | Update note | JWT Required |
| `PUT` | `/api/note/v1/:id/move` | Move note to another notebook | JWT Required |
| `DELETE` | `/api/note/v1/:id` | Delete note (soft-delete) | JWT Required |
| `GET` | `/api/note/v1/semantic-search?q=query` | AI-powered search (Pro Plan) | JWT Required |

---

## E. Fitur Keamanan & Isolasi Data

| Aspek | Implementasi |
|-------|--------------|
| **Authentication** | JWT middleware wajib untuk semua endpoint |
| **Authorization** | [UserOwnedBy](file:///d:/notetaker/notefiber-BE/internal/repository/specification/user_specifications.go#17-20) specification di setiap query |
| **Multi-tenancy** | User ID dari token, bukan dari request body |
| **Soft-delete** | GORM `DeletedAt` untuk data recovery |
| **Transactional** | Atomik delete Note + Embeddings |
| **Premium Guard** | Subscription check untuk Semantic Search |

---

*Dokumen ini di-generate dalam mode READ-ONLY tanpa modifikasi terhadap kode sumber.*
