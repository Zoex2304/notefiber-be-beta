# Deep Linking Backend Analysis

> **Analysis Date**: 2025-12-29  
> **Scope**: Backend implementation alignment with `deeplinking.md` paradigm principles  
> **Decision**: Current UUID implementation retained (UUID v4, not v7)  
> **Status**: ✅ **IMPLEMENTED** — Breadcrumb support added for deep linking

---

## Executive Summary

The current backend implementation **strongly aligns** with the Deep Linking paradigm principles. All core entities use UUID-based identification, ownership verification is consistently applied, and event-driven patterns are in place for state synchronization.

**✅ Deep Linking Implementation Complete**: The `GET /api/note/v1/:id` endpoint now includes a `breadcrumb` field containing the full notebook ancestry path, enabling frontend to display breadcrumbs and auto-expand sidebar when navigating directly to a note via URL.

---

## Paradigm Principles Assessment

### 1. UUID-Based Resource Identification ✅

**Status**: Fully Compliant

All primary entities use `uuid.UUID` as primary keys with database-level UUID generation:

| Entity | ID Type | Default Strategy |
|--------|---------|------------------|
| `User` | `uuid.UUID` | `gen_random_uuid()` |
| `Note` | `uuid.UUID` | `gen_random_uuid()` |
| `Notebook` | `uuid.UUID` | `gen_random_uuid()` |
| `ChatSession` | `uuid.UUID` | `gen_random_uuid()` |
| `PasswordResetToken` | `uuid.UUID` | `gen_random_uuid()` |
| `UserProvider` | `uuid.UUID` | `gen_random_uuid()` |
| `EmailVerificationToken` | `uuid.UUID` | `gen_random_uuid()` |
| `UserRefreshToken` | `uuid.UUID` | `gen_random_uuid()` |

**Code Reference**:
```go
// internal/model/note_model.go
type Note struct {
    Id uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    // ...
}
```

**Paradigm Alignment**:
- ✅ No sequential integer IDs exposed
- ✅ Prevents enumeration attacks
- ✅ Uses `github.com/google/uuid` package
- ℹ️ Uses PostgreSQL `gen_random_uuid()` (UUID v4) — accepted per decision

---

### 2. Authorization as Trust Boundary ✅

**Status**: Fully Compliant

The backend enforces ownership verification independently at the service layer:

#### Ownership Specification Pattern

All user-owned resources are queried with `UserOwnedBy` specification:

```go
// internal/repository/specification/user_specifications.go
type UserOwnedBy struct {
    UserID uuid.UUID
}

func (s UserOwnedBy) Apply(db *gorm.DB) *gorm.DB {
    return db.Where("user_id = ?", s.UserID)
}
```

**Applied Consistently**:
- [note_service.go](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go#L84-L87) - `Show()`, `Update()`, `Delete()`
- [notebook_service.go](file:///d:/notetaker/notefiber-BE/internal/service/notebook_service.go#L126-L129) - `Show()`, `Update()`, `Delete()`

**Example**:
```go
// internal/service/note_service.go
note, err := uow.NoteRepository().FindOne(ctx,
    specification.ByID{ID: id},
    specification.UserOwnedBy{UserID: userId},
)
if note == nil {
    return nil, nil // Returns nil (becomes 404 via middleware)
}
```

#### 404 vs 403 Pattern

| Scenario | HTTP Status | Implementation |
|----------|-------------|----------------|
| Resource not found | `404 Not Found` | Via `ErrNotFound` in error middleware |
| Resource owned by another user | `404 Not Found` | Query returns `nil` → treated as not found |
| Feature requires Pro Plan | `403 Forbidden` | Explicit feature gating |
| Admin-only endpoint | `403 Forbidden` | Role-based access control |

**Paradigm Alignment**:
- ✅ Non-owned resources return 404 (no information disclosure)
- ✅ Backend independently verifies ownership
- ✅ JWT middleware enforces authentication

---

### 3. Frontend/Backend Route Separation ✅

**Status**: Fully Compliant

API routes are under `/api` prefix with versioned resource paths:

| Domain | API Route Pattern | Description |
|--------|-------------------|-------------|
| Notes | `/api/note/v1/:id` | Note CRUD operations |
| Notebooks | `/api/notebook/v1/:id` | Notebook CRUD operations |
| Auth | `/api/auth/v1/*` | Authentication endpoints |
| User | `/api/user/v1/*` | User profile endpoints |
| Admin | `/api/admin/v1/*` | Administrative endpoints |
| Payments | `/api/payment/v1/*` | Subscription/billing |

**Route Registration**:
```go
// internal/server/server.go
func registerRoutes(app *fiber.App, c *bootstrap.Container) {
    api := app.Group("/api")
    c.NoteController.RegisterRoutes(api)
    c.NotebookController.RegisterRoutes(api)
    // ...
}
```

**Paradigm Alignment**:
- ✅ Clear `/api` prefix separates backend endpoints from frontend routes
- ✅ Versioning in path (`/v1/`) enables API evolution
- ✅ RESTful resource naming

---

### 4. Event-Driven Architecture ✅

**Status**: Implemented

The backend uses a publish/subscribe pattern for asynchronous operations:

**Publisher Service**:
- Note creation triggers embedding job: `publisherService.Publish(ctx, msgJson)`
- Note update triggers re-embedding
- Notebook update triggers batch note re-embedding

**Code Reference**:
```go
// internal/service/note_service.go
msgPayload := dto.PublishEmbedNoteMessage{
    NoteId: note.Id,
}
msgJson, err := json.Marshal(msgPayload)
err = c.publisherService.Publish(ctx, msgJson)
```

**Paradigm Alignment**:
- ✅ Causal event triggering (not polling)
- ✅ Asynchronous processing for expensive operations (embeddings)
- ✅ Clear event payload structure

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (SPA)                          │
│        Routes: /app/notes/:uuid, /app/notebooks/:uuid       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    JWT Authentication                        │
│              serverutils.JwtMiddleware                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Controllers                             │
│   /api/note/v1/:id, /api/notebook/v1/:id                    │
│   Extract userId from JWT, forward to service               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Services                               │
│   Owner verification via UserOwnedBy specification          │
│   Returns nil if not owned → 404 response                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Repositories                             │
│       FindOne(ctx, ByID{}, UserOwnedBy{})                   │
│       UUID-based queries                                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      PostgreSQL                              │
│           UUID primary keys, user_id foreign keys           │
└─────────────────────────────────────────────────────────────┘
```

---

## Compliance Matrix

| Paradigm Principle | Current Status | Notes |
|-------------------|----------------|-------|
| UUID-based identification | ✅ Compliant | All entities use UUID v4 |
| No sequential ID exposure | ✅ Compliant | No integer PKs |
| Authorization as trust boundary | ✅ Compliant | `UserOwnedBy` specification |
| 404 for non-owned resources | ✅ Compliant | Query returns nil → 404 |
| Frontend/Backend route separation | ✅ Compliant | `/api` prefix |
| Event-driven patterns | ✅ Compliant | Publisher service for async ops |
| Blocking verification (frontend) | N/A | Frontend responsibility |

---

## Database Schema Alignment

**No database changes required.**

The current schema already uses UUID primary keys across all tables. The `gen_random_uuid()` function is PostgreSQL's native UUID v4 generator.

---

## Recommendations (Optional Enhancements)

### 1. Consistent Error Response for Non-Owned Resources

Currently, when a resource is not found OR not owned, the service returns `nil, nil`:

```go
if note == nil {
    return nil, nil // Becomes 404
}
```

**Recommendation**: Consider returning a sentinel error for clearer tracing:

```go
if note == nil {
    return nil, serverutils.ErrNotFound // Explicit not found
}
```

This is currently handled implicitly but could improve debuggability.

### 2. Deep Link Documentation for Frontend

Create a routing contract document specifying:
- `/app/notes/:uuid` → requires `GET /api/note/v1/:uuid`
- `/app/notebooks/:uuid` → requires `GET /api/notebook/v1/:uuid`

This enables frontend direct navigation without intermediate API discovery.

---

## Implementation: Breadcrumb Support for Deep Linking

### Changes Made

#### 1. DTO Layer

**File**: [note_dto.go](file:///d:/notetaker/notefiber-BE/internal/dto/note_dto.go)

```go
// BreadcrumbItem represents a single notebook in the ancestry path
type BreadcrumbItem struct {
    Id   uuid.UUID `json:"id"`
    Name string    `json:"name"`
}

type ShowNoteResponse struct {
    Id         uuid.UUID        `json:"id"`
    Title      string           `json:"title"`
    Content    string           `json:"content"`
    NotebookId uuid.UUID        `json:"notebook_id"`
    Breadcrumb []BreadcrumbItem `json:"breadcrumb"` // NEW: Ancestry path
    CreatedAt  time.Time        `json:"created_at"`
    UpdatedAt  *time.Time       `json:"updated_at"`
}
```

#### 2. Service Layer

**File**: [note_service.go](file:///d:/notetaker/notefiber-BE/internal/service/note_service.go)

Added `buildBreadcrumb()` helper that traverses the `parent_id` chain:

```go
func (c *noteService) buildBreadcrumb(ctx context.Context, uow unitofwork.UnitOfWork, notebookId uuid.UUID, userId uuid.UUID) ([]dto.BreadcrumbItem, error) {
    // Traverses parent_id chain from note's parent to root
    // Returns array in root-first order
}
```

### API Response Example

```json
{
  "success": true,
  "data": {
    "id": "note-uuid",
    "title": "Meeting Notes",
    "notebook_id": "grandchild-uuid",
    "breadcrumb": [
      { "id": "root-uuid", "name": "Work" },
      { "id": "child-uuid", "name": "Projects" },
      { "id": "grandchild-uuid", "name": "Q1 Planning" }
    ]
  }
}
```

### Frontend Usage

When handling deep link navigation to `/app/note/:uuid`:

1. Fetch the note via `GET /api/note/v1/:uuid`
2. Use `breadcrumb` array to set expanded state of sidebar tree
3. Display breadcrumb navigation using the `name` field

```typescript
// Example sidebar expansion
const expandedIds = new Set(note.breadcrumb.map(b => b.id));
setExpandedNotebooks(expandedIds);
```

---

## Conclusion

The backend implementation **fully aligns** with the Deep Linking paradigm. All resources are UUID-addressable, ownership verification is consistently enforced, and event-driven patterns are in place for asynchronous operations.

**Breadcrumb support has been implemented** to enable frontend deep linking. The `GET /api/note/v1/:id` endpoint now returns the full notebook ancestry path.

The decision to use UUID v4 (current implementation) instead of UUID v7 is documented and accepted. The core security benefits (unpredictability, enumeration prevention) are preserved.
