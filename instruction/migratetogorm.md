COMPLETE GORM REFACTORING BLUEPRINT - ARCHITECTURAL GUIDE
PROJECT STRUCTURE OVERVIEW
ai-notetaking-be/
│
├── cmd/
│ └── api/
│ └── main.go 
│
├── internal/
│ │
│ ├── entity/ [REFACTOR - Remove GORM tags]
│ │ ├── user.go
│ │ ├── notebook.go
│ │ ├── note.go
│ │ ├── note_embedding.go
│ │ ├── chat_session.go
│ │ ├── chat_message.go
│ │ ├── chat_message_raw.go
│ │ ├── subscription.go
│ │ ├── billing.go
│ │ └── example.go
│ │
│ ├── model/ [CREATE NEW - GORM models]
│ │ ├── user_model.go
│ │ ├── notebook_model.go
│ │ ├── note_model.go
│ │ ├── note_embedding_model.go
│ │ ├── chat_session_model.go
│ │ ├── chat_message_model.go
│ │ ├── chat_message_raw_model.go
│ │ ├── subscription_model.go
│ │ ├── billing_model.go
│ │ └── example_model.go
│ │
│ ├── mapper/ [CREATE NEW - Translation layer]
│ │ ├── user_mapper.go
│ │ ├── notebook_mapper.go
│ │ ├── note_mapper.go
│ │ ├── note_embedding_mapper.go
│ │ ├── chat_mapper.go
│ │ ├── subscription_mapper.go
│ │ ├── billing_mapper.go
│ │ └── example_mapper.go
│ │
│ ├── repository/
│ │ │
│ │ ├── contract/ [CREATE NEW - Repository interfaces]
│ │ │ ├── user_repository.go
│ │ │ ├── notebook_repository.go
│ │ │ ├── note_repository.go
│ │ │ ├── note_embedding_repository.go
│ │ │ ├── chat_session_repository.go
│ │ │ ├── chat_message_repository.go
│ │ │ ├── chat_message_raw_repository.go
│ │ │ ├── subscription_repository.go
│ │ │ ├── billing_repository.go
│ │ │ └── example_repository.go
│ │ │
│ │ ├── implementation/ [CREATE NEW - GORM implementations]
│ │ │ ├── user_repository_impl.go
│ │ │ ├── notebook_repository_impl.go
│ │ │ ├── note_repository_impl.go
│ │ │ ├── note_embedding_repository_impl.go
│ │ │ ├── chat_session_repository_impl.go
│ │ │ ├── chat_message_repository_impl.go
│ │ │ ├── chat_message_raw_repository_impl.go
│ │ │ ├── subscription_repository_impl.go
│ │ │ ├── billing_repository_impl.go
│ │ │ └── example_repository_impl.go
│ │ │
│ │ ├── specification/ [CREATE NEW - Query conditions]
│ │ │ ├── specification.go (Base interface)
│ │ │ ├── common_specifications.go (NotDeleted, ByID, OrderBy)
│ │ │ ├── user_specifications.go (ByEmail, UserOwnedBy, ActiveUsers)
│ │ │ ├── notebook_specifications.go (ByParentID, ByIDs)
│ │ │ ├── note_specifications.go (ByNotebookID, ByIDs)
│ │ │ └── chat_specifications.go (ByChatSessionID)
│ │ │
│ │ ├── scope/ [CREATE NEW - GORM scopes]
│ │ │ ├── common_scopes.go (NotDeleted, Pagination)
│ │ │ ├── pagination_scope.go (WithPagination helper)
│ │ │ └── soft_delete_scope.go (Soft delete handling)
│ │ │
│ │ ├── unitofwork/ [CREATE NEW - Transaction mgmt]
│ │ │ ├── unit_of_work.go (Interface definition)
│ │ │ ├── unit_of_work_impl.go (Implementation)
│ │ │ └── repository_factory.go (Factory for creating repos)
│ │ │
│ │ └── vector/ [CREATE NEW - Special vector ops]
│ │ ├── vector_query_service.go (Raw SQL for pgvector)
│ │ ├── vector_specification.go (Vector-specific specs)
│ │ └── vector_helper.go (Vector conversion utilities)
│ │
│ ├── service/ [REFACTOR - Use Unit of Work]
│ │ ├── user_service.go
│ │ ├── notebook_service.go
│ │ ├── note_service.go
│ │ ├── chat_service.go
│ │ └── subscription_service.go
│ │
│ ├── controller/ [NO CHANGES]
│ │ └── ...
│ │
│ ├── dto/ [NO CHANGES]
│ │ └── ...
│ │
│ └── pkg/
│ └── serverutils/ [NO CHANGES]
│ └── errors.go
│
└── pkg/
└── database/
├── config.go [MINOR CHANGES]
├── gorm.go [CREATE NEW - Replace pgx]
└── migrations.go [GORM AutoMigrate]
ai-notetaking-be/cmd/api/main.go
ai-notetaking-be/internal/entity/user.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/notebook.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/note.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/note_embedding.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/chat_session.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/chat_message.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/chat_message_raw.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/subscription.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/billing.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/entity/example.go [REFACTOR - Remove GORM tags]
ai-notetaking-be/internal/model/user_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/notebook_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/note_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/note_embedding_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/chat_session_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/chat_message_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/chat_message_raw_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/subscription_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/billing_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/model/example_model.go [CREATE NEW - GORM models]
ai-notetaking-be/internal/mapper/user_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/notebook_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/note_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/note_embedding_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/chat_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/subscription_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/billing_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/mapper/example_mapper.go [CREATE NEW - Translation layer]
ai-notetaking-be/internal/repository/contract/user_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/notebook_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/note_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/note_embedding_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/chat_session_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/chat_message_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/chat_message_raw_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/subscription_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/billing_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/contract/example_repository.go [CREATE NEW - Repository interfaces]
ai-notetaking-be/internal/repository/implementation/user_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/notebook_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/note_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/note_embedding_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/chat_session_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/chat_message_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/chat_message_raw_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/subscription_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/billing_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/implementation/example_repository_impl.go [CREATE NEW - GORM implementations]
ai-notetaking-be/internal/repository/specification/specification.go [CREATE NEW - Query conditions] (Base interface)
ai-notetaking-be/internal/repository/specification/common_specifications.go [CREATE NEW - Query conditions] (NotDeleted, ByID, OrderBy)
ai-notetaking-be/internal/repository/specification/user_specifications.go [CREATE NEW - Query conditions] (ByEmail, UserOwnedBy, ActiveUsers)
ai-notetaking-be/internal/repository/specification/notebook_specifications.go [CREATE NEW - Query conditions] (ByParentID, ByIDs)
ai-notetaking-be/internal/repository/specification/note_specifications.go [CREATE NEW - Query conditions] (ByNotebookID, ByIDs)
ai-notetaking-be/internal/repository/specification/chat_specifications.go [CREATE NEW - Query conditions] (ByChatSessionID)
ai-notetaking-be/internal/repository/scope/common_scopes.go [CREATE NEW - GORM scopes]
ai-notetaking-be/internal/repository/scope/pagination_scope.go [CREATE NEW - GORM scopes] (WithPagination helper)
ai-notetaking-be/internal/repository/scope/soft_delete_scope.go [CREATE NEW - GORM scopes] (Soft delete handling)
ai-notetaking-be/internal/repository/unitofwork/unit_of_work.go [CREATE NEW - Transaction mgmt] (Interface definition)
ai-notetaking-be/internal/repository/unitofwork/unit_of_work_impl.go [CREATE NEW - Transaction mgmt] (Implementation)
ai-notetaking-be/internal/repository/unitofwork/repository_factory.go [CREATE NEW - Transaction mgmt] (Factory for creating repos)
ai-notetaking-be/internal/repository/vector/vector_query_service.go [CREATE NEW - Special vector ops] (Raw SQL for pgvector)
ai-notetaking-be/internal/repository/vector/vector_specification.go [CREATE NEW - Special vector ops] (Vector-specific specs)
ai-notetaking-be/internal/repository/vector/vector_helper.go [CREATE NEW - Special vector ops] (Vector conversion utilities)
ai-notetaking-be/internal/service/user_service.go [REFACTOR - Use Unit of Work]
ai-notetaking-be/internal/service/notebook_service.go [REFACTOR - Use Unit of Work]
ai-notetaking-be/internal/service/note_service.go [REFACTOR - Use Unit of Work]
ai-notetaking-be/internal/service/chat_service.go [REFACTOR - Use Unit of Work]
ai-notetaking-be/internal/service/subscription_service.go [REFACTOR - Use Unit of Work]
ai-notetaking-be/pkg/database/config.go [MINOR CHANGES]
ai-notetaking-be/pkg/database/gorm.go [CREATE NEW - Replace pgx]
ai-notetaking-be/pkg/database/migrations.go [GORM AutoMigrate]
Location: internal/repository/implementation/
Responsibility: Implement repository interfaces using GORM
Dependencies: GORM, Model, Mapper, Contract, Specification
Pattern: Repository pattern with specifications
Layer 6: Repository Contract (Interfaces)
Location: internal/repository/contract/
Responsibility: Define repository interface contracts
Dependencies: Entity, Specification
Pattern: Interface segregationLayer 5: Scope (GORM Scopes)
Location: internal/repository/scope/
Responsibility: Reusable GORM scope functions
Dependencies: GORM
Pattern: GORM scope functions
Layer 4: Specification (Query Conditions)
Location: internal/repository/specification/
Responsibility: Encapsulate reusable query logic
Dependencies: GORM
Pattern: Strategy pattern for queries
Layer 3: Mapper (Translation Layer)
Location: internal/mapper/
Responsibility: Convert between Entity and Model
Dependencies: Entity, Model
Pattern: Unidirectional conversion methods
Layer 2: Model (GORM Persistence)
Location: internal/model/
Responsibility: Database table representation with GORM tags
Dependencies: GORM
Purpose: Complete separation of persistence concerns
go
Layer 1: Entity (Domain Models)
Location: internal/entity/
Responsibility: Pure business domain representation
Dependencies: None
Changes: Remove all GORM tags, keep pure Go structs
LAYER RESPONSIBILITY MATRIX
LayerLocationResponsibilityDependenciesChanges RequiredEntityinternal/entity/Pure business domain representation, no infrastructure concernsNoneREFACTOR - Remove all GORM tags and database annotationsModelinternal/model/Database table structure with GORM tags, indexes, constraintsGORMCREATE NEW - Mirror entities with persistence metadataMapperinternal/mapper/Bidirectional conversion between Entity and ModelEntity, ModelCREATE NEW - Translate domain to persistence and backContractinternal/repository/contract/Repository interface definitions, method signaturesEntity, SpecificationCREATE NEW - Define what repositories can doImplementationinternal/repository/implementation/Actual GORM query implementationsGORM, Model, Mapper, ContractCREATE NEW - Replace all raw SQL with GORMSpecificationinternal/repository/specification/Reusable query conditions, composable filtersGORMCREATE NEW - Encapsulate query logicScopeinternal/repository/scope/Cross-cutting GORM scopes (pagination, soft delete)GORMCREATE NEW - Common query modifiersUnit of Workinternal/repository/unitofwork/Transaction lifecycle managementGORM, ContractsCREATE NEW - Replace UsingTx patternVectorinternal/repository/vector/pgvector-specific operationsGORM, pgvectorCREATE NEW - Handle vector similarity searchesServiceinternal/service/Business logic orchestrationUnit of Work, RepositoriesREFACTOR - Use UoW for transactionsDatabasepkg/database/Database connection managementGORMREFACTOR - Replace pgxpool with gorm.DB

ARCHITECTURAL FLOW DIAGRAMS
DATA FLOW (Read Operation)
Controller Request
│
├──> DTO Validation
│
▼
Service Layer
│
├──> Call Repository.FindAll(ctx, specifications...)
│
▼
Repository (Contract Interface)
│
▼
Repository (Implementation)
│
├──> Apply Specifications to GORM query
├──> Apply Scopes (NotDeleted, Pagination)
├──> Execute GORM query
├──> Get Model objects from database
│
▼
Mapper Layer
│
├──> Convert Model → Entity
│
▼
Service Layer
│
├──> Business logic processing
│
▼
Controller Response
│
└──> Return DTO to client
DATA FLOW (Write Operation with Transaction)
Controller Request
│
▼
Service Layer
│
├──> Create Unit of Work
├──> Begin Transaction
│
▼
Unit of Work
│
├──> Get Repository instances (with tx)
│
▼
Service Logic
│
├──> repo1.Create(entity1)
│ │
│ ├──> Mapper: Entity → Model
│ ├──> GORM: db.Create(model)
│ └──> Return error or nil
│
├──> repo2.Update(entity2)
│ │
│ ├──> Mapper: Entity → Model
│ ├──> GORM: db.Updates(model)
│ └──> Return error or nil
│
├──> If any error → UnitOfWork.Rollback()
│
└──> If success → UnitOfWork.Commit()
│
▼
Controller Response
SPECIFICATION COMPOSITION FLOW
Service calls:
repo.FindAll(ctx,
specification.UserOwnedBy{UserID: userId},
specification.NotDeleted{},
specification.OrderByCreatedDesc{})

       │
       ▼

Repository Implementation:
│
├──> query = db.WithContext(ctx)
│
├──> FOR EACH specification:
│ query = specification.Apply(query)
│
├──> Execute: query.Find(&models)
│
├──> Mapper: models → entities
│
└──> Return entities

LAYER CREATION SEQUENCE
PHASE 1: FOUNDATION (No code changes to existing files)
Step 1: Create Model Layer
What to do:

Create new files in internal/model/
For each entity, create corresponding model file
Add GORM struct tags: gorm:"column:xxx;type:xxx;primaryKey"
Add TableName() method for each model
Add indexes, constraints via GORM tags

Key considerations:

Use exact database column names
Map Go types to database types correctly
Handle NULL values with pointers
Add soft delete fields: DeletedAt, IsDeleted

Step 2: Create Mapper Layer
What to do:

Create new files in internal/mapper/
Define mapper interface for each domain entity
Implement ToEntity(model) method
Implement ToModel(entity) method
Implement batch conversion methods: ToEntities(), ToModels()

Key considerations:

Handle NULL pointer conversions safely
Convert enum strings to typed enums
Preserve all field values during conversion
Make mappers stateless and reusable

Step 3: Create Specification Layer
What to do:

Create internal/repository/specification/specification.go with base interface
Create common_specifications.go for universal filters
Create domain-specific specification files per entity
Implement Apply(db \*gorm.DB) method for each specification

Key considerations:

Each specification = one query condition
Specifications should be composable
Keep specifications immutable
Specifications should not depend on each other

Step 4: Create Scope Layer
What to do:

Create internal/repository/scope/common_scopes.go
Implement GORM scope functions
Create pagination helper
Create soft delete scope

Key considerations:

Scopes are functions that modify GORM query
Scopes are reusable across repositories
Scopes handle cross-cutting concerns

Step 5: Create Repository Contract Layer
What to do:

Create internal/repository/contract/ directory
For each existing repository interface, create new contract file
Define interface methods using entity types (not models)
Add specification-based methods: FindOne, FindAll, Count

Key considerations:

Contracts use entity types, not models
Contracts accept specifications as variadic parameters
Keep method signatures simple and consistent
Contract = what repository can do, not how

Step 6: Create Unit of Work Layer
What to do:

Create internal/repository/unitofwork/unit_of_work.go interface
Define Begin, Commit, Rollback methods
Define repository accessor methods
Create implementation file
Create repository factory

Key considerations:

Unit of Work manages transaction lifecycle
All repositories accessed through UoW share same transaction
Factory creates repository instances with correct DB/TX

PHASE 2: IMPLEMENTATION (Create new implementations)
Step 7: Create Repository Implementations
What to do:

Create internal/repository/implementation/ directory
For each contract, create implementation file
Implement all contract methods using GORM
Inject mapper and database dependencies
Use specifications in query methods

Key considerations:

Replace all raw SQL with GORM query builder
Use mapper to convert between model and entity
Apply specifications using loop
Handle GORM errors and map to domain errors
Use scopes for common filters

Step 8: Create Vector Query Service
What to do:

Create internal/repository/vector/ directory
Create vector_query_service.go for raw SQL vector operations
Create vector_specification.go for vector-specific filters
Create vector_helper.go for embedding conversions

Key considerations for vector operations:

pgvector operations may need raw SQL
Keep vector queries separate from standard GORM repos
Use custom GORM data type for vector fields
Provide similarity threshold specifications

PHASE 3: DATABASE LAYER (Replace pgx with GORM)
Step 9: Create GORM Connection Manager
What to do:

Create pkg/database/gorm.go
Implement connection factory with GORM
Configure connection pool settings
Setup logger integration
Handle connection lifecycle

Key considerations:

Replace pgxpool.Pool with gorm.DB
Configure max connections, idle connections
Setup GORM logger for SQL debugging
Add health check method

PHASE 4: SERVICE REFACTORING (Change service layer)
Step 10: Refactor Services to Use Unit of Work
What to do:

Remove all repo.UsingTx(ctx, tx) patterns
Inject Unit of Work factory into services
Replace transaction code with UoW Begin/Commit/Rollback
Access repositories through UoW instance

Transaction pattern change:
BEFORE:
tx := db.Begin()
defer tx.Rollback()
repo.UsingTx(ctx, tx).Create(...)
tx.Commit()

AFTER:
uow := uowFactory.Create()
uow.Begin(ctx)
defer uow.Rollback()
uow.UserRepository().Create(...)
uow.Commit()

PHASE 5: ENTITY CLEANUP (Remove GORM dependencies)
Step 11: Clean Entity Layer
What to do:

Remove all GORM tags from entity files
Remove GORM imports
Keep only business logic fields
Ensure entities are pure Go structs

SPECIAL HANDLING: VECTOR OPERATIONS
Problem Statement
pgvector operations require special SQL syntax:

Cosine similarity: 1 - (embedding <=> $1)
L2 distance: embedding <-> $1
Inner product: embedding <#> $1

GORM does not natively support pgvector operators well.
Solution Strategy
Option 1: Hybrid Approach (RECOMMENDED)
What:

Standard CRUD via GORM
Vector similarity searches via raw SQL in VectorQueryService
Results mapped back to entities

When to use:

Complex vector operations
Performance-critical vector searches
Need fine-grained control over vector queries

Structure:
VectorQueryService
├── SearchSimilar(embedding, threshold, limit) → []Entity
├── SearchByNoteEmbedding(noteId, limit) → []Entity
└── BulkInsertEmbeddings([]Entity) → error
Option 2: Custom GORM Type
What:

Create custom GORM data type for pgvector
Implement Scan() and Value() methods
Use GORM with custom WHERE clauses

When to use:

Simple vector operations
Prefer consistency with GORM pattern
Don't need complex vector queries

Limitation:

Still may need raw SQL for complex operations
Performance might be lower

Recommended Approach
Use Option 1 (Hybrid) because:

Vector operations are specialized
Performance is critical for embeddings
Raw SQL gives more control
Separation of concerns: standard queries vs vector queries

MIGRATION EXECUTION PLAN
Week 1: Foundation Setup
Tasks:

Create all model files
Create all mapper files
Create specification base and common specs
Create scope layer
Create unit of work interface
Write comprehensive tests for mappers

Deliverable: Complete foundation layers, zero impact on existing code
Week 2: Repository Contracts and Implementations
Tasks:

Create all repository contract interfaces
Implement User repository with GORM
Implement Notebook repository with GORM
Implement Note repository with GORM
Write integration tests

Deliverable: 3 core repositories working with GORM
Week 3: Remaining Repositories
Tasks:

Implement NoteEmbedding repository
Create Vector Query Service
Implement Chat repositories (Session, Message, MessageRaw)
Implement Subscription and Billing repositories
Write integration tests

Deliverable: All 10 repositories migrated to GORM
Week 4: Service Layer Refactoring
Tasks:

Refactor User service to use Unit of Work
Refactor Notebook service to use Unit of Work
Refactor Note service to use Unit of Work
Refactor Subscription service to use Unit of Work
Update dependency injection

Deliverable: All services using Unit of Work pattern
Week 5: Testing and Cleanup
Tasks:

End-to-end integration tests
Performance benchmarking (compare with pgx baseline)
Remove old repository files
Remove pgx dependencies
Update documentation
Code review and optimization

Deliverable: Production-ready GORM implementation

TESTING STRATEGY
Unit Tests
What to test:

Mapper conversions (Entity → Model → Entity)
Specification logic (correct WHERE clauses)
Repository methods in isolation (use test database)

Integration Tests
What to test:

Repository CRUD operations
Specification composition
Transaction rollback scenarios
Unit of Work lifecycle

Performance Tests
What to test:

Query execution time (GORM vs pgx baseline)
Memory usage
Connection pool behavior
Vector search performance

DEPENDENCY INJECTION CHANGES
Current Pattern (pgx)
Database (pgxpool.Pool)
↓
Repository (receives pool)
↓
Service (receives repository)
New Pattern (GORM + Unit of Work)
Database (gorm.DB)
↓
Mapper (stateless)
↓
Repository Implementation (receives gorm.DB + Mapper)
↓
Unit of Work Factory (creates UoW with repositories)
↓
Service (receives UoW Factory)

ROLLBACK STRATEGY
Phase Rollback Points
Each week's work can be rolled back independently:

Week 1: Delete new directories, no impact
Week 2: Keep old repositories, switch DI back
Week 3: Gradual migration per repository
Week 4: Service-by-service rollback possible

Gradual Migration Strategy
Run both systems in parallel:

Old services use pgx repositories
New services use GORM repositories
Migrate one domain at a time
Compare results for correctness

SUCCESS CRITERIA
Functional Requirements

All existing API endpoints work identically
All database operations produce same results
Transaction handling works correctly
Error handling is consistent

Non-Functional Requirements

Query performance within 10% of pgx baseline
Memory usage within acceptable limits
Code maintainability improved
Test coverage above 80%

Architectural Requirements

Single Responsibility Principle maintained
Each layer has clear boundaries
Dependencies flow in one direction
Code is modular and testable

RISK MITIGATION
RiskImpactMitigationPerformance regressionHIGHBenchmark early, optimize GORM queries, use raw SQL where neededComplex transactions failHIGHExtensive transaction testing, rollback verificationpgvector compatibilityMEDIUMUse hybrid approach, test vector operations thoroughlyBreaking changes in servicesHIGHMigrate incrementally, run parallel systemsTimeline overrunMEDIUMClear phase deliverables, weekly reviewsTeam learning curveMEDIUMDocumentation, code examples, pair programming

FINAL CHECKLIST
Before Starting Migration

Backup current codebase
Document current API behavior
Setup GORM in test environment
Benchmark current performance
Get team alignment on architecture

During Migration

Follow phase sequence strictly
Write tests before implementation
Review code after each phase
Keep old code until fully migrated
Document any deviations

After Migration

Verify all APIs work
Compare performance metrics
Update all documentation
Remove old pgx code
Deploy to staging first
Monitor production carefully

This blueprint provides complete guidance without touching any existing code. Follow the phases sequentially, and the migration will be clean, maintainable, and reversible.
