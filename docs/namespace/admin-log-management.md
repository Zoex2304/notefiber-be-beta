# Dokumentasi Fitur: Admin Logging Management

> **Fokus Domain:** BACKEND  
> **Konteks:** Trace Upstream ke Downstream secara Semantik  
> **Scope:** System logging dengan Zap + Lumberjack dan admin dashboard access

---

## Alur Data Semantik (Scope: BACKEND)

```
=== LOG WRITING (From Services) ===
[Service: Admin/Auth/Payment]  
    -> [Logger.Info/Warn/Error]  
    -> [Zap Logger: Format as JSON]  
    -> [Lumberjack: Write to File]  
    -> [Auto-Rotation: Size/Age/Backups]

=== GET LOGS (Admin Dashboard) ===
[HTTP GET /api/admin/logs]  
    -> [Admin Middleware: Validate JWT + Admin Role]  
    -> [Controller: Parse Query Params]  
    -> [Service: GetSystemLogs]  
        -> [Logger.GetLogs(level, limit, offset)]  
            -> [Read Log File]  
            -> [Parse JSON Lines]  
            -> [Filter by Level]  
            -> [Generate MD5 ID]  
            -> [Reverse (Newest First)]  
            -> [Paginate]  
    -> [HTTP Response dengan Log Entries]

=== GET LOG DETAIL (Admin Dashboard) ===
[HTTP GET /api/admin/logs/:id]  
    -> [Admin Middleware: Validate JWT + Admin Role]  
    -> [Controller: Parse Log ID]  
    -> [Service: GetLogDetail]  
        -> [Logger.GetLogById(id)]  
            -> [Scan Logs]  
            -> [Match MD5 ID]  
    -> [HTTP Response dengan Log Details]
```

---

## A. Laporan Implementasi Fitur Admin Logging Management

### Deskripsi Fungsional

Fitur ini menyediakan sistem logging terintegrasi menggunakan **Zap Logger** dengan rotasi file via **Lumberjack**. Administrator dapat melihat dan filter logs melalui dashboard. Sistem mengimplementasikan:

1. **Log Writing**: Services memanggil logger untuk audit trail (Info, Warn, Error)
2. **Log Storage**: JSON-formatted lines disimpan ke file dengan rotasi otomatis
3. **GetLogs**: Fetch logs dengan pagination dan filter by level
4. **GetLogDetail**: Fetch single log entry by MD5 hash ID
5. **Log Rotation**: Automatic rotation berdasarkan size/age/backups

Log ID menggunakan **MD5 hash** dari content, bukan UUID.

### Visualisasi

**Log List Response:**
```json
{
    "success": true,
    "code": 200,
    "message": "System logs",
    "data": [
        {
            "id": "5d41402abc4b2a76b9719d911017c592",
            "level": "INFO",
            "module": "ADMIN",
            "message": "Updated user status",
            "created_at": "2024-12-25T01:00:00Z"
        },
        {
            "id": "7d793037a0760186574b0282f2f435e7",
            "level": "ERROR",
            "module": "PAYMENT",
            "message": "Midtrans webhook failed",
            "created_at": "2024-12-25T00:55:00Z"
        }
    ]
}
```

**Log Detail Response:**
```json
{
    "success": true,
    "code": 200,
    "message": "Log detail",
    "data": {
        "id": "5d41402abc4b2a76b9719d911017c592",
        "level": "INFO",
        "module": "ADMIN",
        "message": "Updated user status",
        "created_at": "2024-12-25T01:00:00Z",
        "details": {
            "userId": "550e8400-e29b-41d4-a716-446655440000",
            "status": "blocked",
            "admin": "system"
        }
    }
}
```
*Caption: Gambar 1: Response untuk Log List dan Log Detail.*

---

## B. Bedah Arsitektur & Komponen

Berikut adalah rincian 12 komponen yang menyusun fitur ini di sisi BACKEND.

---

### [internal/server/server.go](file:///d:/notetaker/notefiber-BE/internal/server/server.go)
**Layer Terdeteksi:** `HTTP Server & Route Registration`

**Narasi Operasional:**
Server mendaftarkan [AdminController](file:///d:/notetaker/notefiber-BE/internal/controller/admin_controller.go#17-56) yang menangani log endpoints.

```go
func registerRoutes(app *fiber.App, c *bootstrap.Container) {
	api := app.Group("/api")
	c.AdminController.RegisterRoutes(api)
}
```
*Caption: Snippet 1: Registrasi AdminController.*

---

### [internal/bootstrap/container.go](file:///d:/notetaker/notefiber-BE/internal/bootstrap/container.go)
**Layer Terdeteksi:** `Dependency Injection Container`

**Narasi Operasional:**
[ZapLogger](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go#25-29) diinisialisasi dengan path ke log file dan mode production. Logger kemudian di-inject ke [AdminService](file:///d:/notetaker/notefiber-BE/internal/service/admin_service.go#25-72).

```go
func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	// Initialize Logger
	logFilePath := cfg.LogFilePath // e.g., "logs/app.log"
	isProd := cfg.Environment == "production"
	zapLogger := logger.NewZapLogger(logFilePath, isProd)

	// Services with Logger
	adminService := service.NewAdminService(uowFactory, zapLogger)

	return &Container{
		AdminController: controller.NewAdminController(adminService, authService),
	}
}
```
*Caption: Snippet 2: Konstruksi ZapLogger dan injection ke AdminService.*

---

### [internal/dto/admin_log_dto.go](file:///d:/notetaker/notefiber-BE/internal/dto/admin_log_dto.go)
**Layer Terdeteksi:** `Data Transfer Object (DTO)`

**Narasi Operasional:**
File ini mendefinisikan kontrak data untuk log operations. [LogListResponse](file:///d:/notetaker/notefiber-BE/internal/dto/admin_log_dto.go#40-47) menggunakan `string` untuk ID karena log ID adalah MD5 hash. [LogDetailResponse](file:///d:/notetaker/notefiber-BE/internal/dto/admin_log_dto.go#48-52) extends list response dengan `details` map.

```go
// LogListResponse uses string for Id because log IDs are MD5 hashes, not UUIDs
type LogListResponse struct {
	Id        string    `json:"id"` // MD5 hash, not UUID
	Level     string    `json:"level"`
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type LogDetailResponse struct {
	LogListResponse
	Details map[string]interface{} `json:"details"`
}
```
*Caption: Snippet 3: DTO untuk Log dengan MD5 ID.*

---

### [internal/controller/admin_controller.go](file:///d:/notetaker/notefiber-BE/internal/controller/admin_controller.go)
**Layer Terdeteksi:** `Interface / Controller Layer`

**Narasi Operasional:**
Controller menangani log endpoints dengan pagination dan level filter dari query params. Log ID diterima sebagai string (bukan UUID).

```go
func (c *adminController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/admin")
	h.Use(c.adminMiddleware)

	// Logs
	h.Get("/logs", c.GetLogs)
	h.Get("/logs/:id", c.GetLogDetail)
}

func (c *adminController) GetLogs(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	level := ctx.Query("level", "") // INFO, WARN, ERROR, or empty for all

	logs, err := c.service.GetSystemLogs(ctx.Context(), page, limit, level)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("System logs", logs))
}

func (c *adminController) GetLogDetail(ctx *fiber.Ctx) error {
	logId := ctx.Params("id") // Log ID is a string (MD5 hash), not UUID

	l, err := c.service.GetLogDetail(ctx.Context(), logId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "Log not found"))
	}
	return ctx.JSON(serverutils.SuccessResponse("Log detail", l))
}
```
*Caption: Snippet 4: Controller dengan pagination dan level filter.*

---

### [internal/service/admin_service.go](file:///d:/notetaker/notefiber-BE/internal/service/admin_service.go)
**Layer Terdeteksi:** `Business Logic / Service Layer`

**Narasi Operasional:**
Service memanggil logger interface untuk read operations. Service juga menggunakan logger untuk write operations (audit trail).

```go
type adminService struct {
	uowFactory unitofwork.RepositoryFactory
	logger     logger.ILogger
}

func NewAdminService(uowFactory unitofwork.RepositoryFactory, logger logger.ILogger) IAdminService {
	return &adminService{
		uowFactory: uowFactory,
		logger:     logger,
	}
}

// Reading logs for Admin Dashboard
func (s *adminService) GetSystemLogs(ctx context.Context, page, limit int, level string) ([]*dto.LogListResponse, error) {
	logs, err := s.logger.GetLogs(level, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	var res []*dto.LogListResponse
	for _, l := range logs {
		ts, _ := time.Parse(time.RFC3339, l.Timestamp)
		res = append(res, &dto.LogListResponse{
			Id:        l.Id, // MD5 hash
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		})
	}
	return res, nil
}

func (s *adminService) GetLogDetail(ctx context.Context, logId string) (*dto.LogDetailResponse, error) {
	l, err := s.logger.GetLogById(logId)
	if err != nil {
		return nil, err
	}

	ts, _ := time.Parse(time.RFC3339, l.Timestamp)
	return &dto.LogDetailResponse{
		LogListResponse: dto.LogListResponse{
			Id:        logId,
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		},
		Details: l.Details,
	}, nil
}

// Writing logs (Audit Trail)
func (s *adminService) UpdateUserStatus(ctx context.Context, userId uuid.UUID, status string) error {
	// Audit log
	s.logger.Info("ADMIN", "Updated user status", map[string]interface{}{
		"userId": userId.String(),
		"status": status,
		"admin":  "system",
	})
	
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return uow.UserRepository().UpdateStatus(ctx, userId, status)
}
```
*Caption: Snippet 5: Service dengan log reading dan audit writing.*

---

### [internal/pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go)
**Layer Terdeteksi:** `Logger Interface`

**Narasi Operasional:**
Interface mendefinisikan kontrak untuk logging operations: write methods (Debug, Info, Warn, Error) dan read methods (GetLogs, GetLogById).

```go
type ILogger interface {
	// Write Methods (for Services)
	Debug(module, message string, details map[string]interface{})
	Info(module, message string, details map[string]interface{})
	Warn(module, message string, details map[string]interface{})
	Error(module, message string, details map[string]interface{})
	Sync() error
	
	// Read Methods (for Admin Dashboard)
	GetLogs(level string, limit, offset int) ([]LogEntry, error)
	GetLogById(id string) (*LogEntry, error)
}
```
*Caption: Snippet 6: Logger interface dengan write dan read methods.*

---

### [internal/pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) (Implementation)
**Layer Terdeteksi:** `Zap Logger Implementation`

**Narasi Operasional:**
Implementasi menggunakan **Zap** untuk high-performance logging dan **Lumberjack** untuk file rotation. Output ke file (JSON) dan console (development mode).

```go
type ZapLogger struct {
	logger   *zap.Logger
	filePath string
}

func NewZapLogger(logFilePath string, isProd bool) *ZapLogger {
	// 1. Configure Rotation (Lumberjack)
	rotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // Megabytes
		MaxBackups: 5,    // Files to keep
		MaxAge:     30,   // Days
		Compress:   true, // gzip old logs
	}

	// 2. Configure Encoder (JSON for file)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// 3. File Core (JSON, INFO level)
	fileCore := zapcore.NewCore(
		jsonEncoder,
		zapcore.AddSync(rotator),
		zap.InfoLevel,
	)

	// 4. Console Core (Dev mode uses console format)
	var consoleEncoder zapcore.Encoder
	if isProd {
		consoleEncoder = jsonEncoder
	} else {
		consoleEncoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}

	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.Lock(os.Stdout),
		zap.DebugLevel,
	)

	// 5. Join Cores (Tee output)
	core := zapcore.NewTee(fileCore, consoleCore)

	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &ZapLogger{
		logger:   l,
		filePath: logFilePath,
	}
}

// Write Methods
func (l *ZapLogger) Info(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	l.logger.Info(message, zap.String("module", module), zap.Any("details", details))
}

func (l *ZapLogger) Error(module, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	// Extract error for stacktrace if exists
	if err, ok := details["error"]; ok {
		l.logger.Error(message, zap.String("module", module), zap.Any("details", details), zap.Any("error_ref", err))
	} else {
		l.logger.Error(message, zap.String("module", module), zap.Any("details", details))
	}
}
```
*Caption: Snippet 7: Zap Logger dengan Lumberjack rotation.*

---

### [internal/pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) (Log Reading)
**Layer Terdeteksi:** `Log Reading Implementation`

**Narasi Operasional:**
Log reading membaca file secara sequential, parse JSON lines, filter by level, generate MD5 ID, reverse untuk newest first, dan paginate.

```go
type LogEntry struct {
	Id        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Module    string                 `json:"module,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (l *ZapLogger) GetLogs(level string, limit, offset int) ([]LogEntry, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry LogEntry
		
		// Parse JSON line
		if err := json.Unmarshal(line, &entry); err == nil {
			// Filter by level if requested
			if level != "" && entry.Level != level {
				continue
			}
			
			// Generate ID if missing (MD5 hash of content)
			if entry.Id == "" {
				entry.Id = fmt.Sprintf("%x", md5.Sum(line))
			}
			
			entries = append(entries, entry)
		}
	}

	// Reverse to show newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Pagination
	start := offset
	end := offset + limit
	if start >= len(entries) {
		return []LogEntry{}, nil
	}
	if end > len(entries) {
		end = len(entries)
	}

	return entries[start:end], nil
}

func (l *ZapLogger) GetLogById(id string) (*LogEntry, error) {
	logs, err := l.GetLogs("", 10000, 0) // Scan last 10k logs
	if err != nil {
		return nil, err
	}
	
	for _, log := range logs {
		if log.Id == id {
			return &log, nil
		}
	}
	
	return nil, fmt.Errorf("log not found")
}
```
*Caption: Snippet 8: Log reading dengan MD5 ID generation.*

---

## C. Ringkasan Layer Arsitektur

| No | Layer | File | Tanggung Jawab |
|----|-------|------|----------------|
| 1 | HTTP Server | [server/server.go](file:///d:/notetaker/notefiber-BE/internal/server/server.go) | Route registration |
| 2 | DI Container | [bootstrap/container.go](file:///d:/notetaker/notefiber-BE/internal/bootstrap/container.go) | Logger construction |
| 3 | DTO | [dto/admin_log_dto.go](file:///d:/notetaker/notefiber-BE/internal/dto/admin_log_dto.go) | LogListResponse, LogDetailResponse |
| 4 | Controller | [controller/admin_controller.go](file:///d:/notetaker/notefiber-BE/internal/controller/admin_controller.go) | Log endpoints |
| 5 | Middleware | [controller/admin_controller.go](file:///d:/notetaker/notefiber-BE/internal/controller/admin_controller.go) | Admin protection |
| 6 | Service | [service/admin_service.go](file:///d:/notetaker/notefiber-BE/internal/service/admin_service.go) | Log access + audit writing |
| 7 | Logger Interface | [pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) | ILogger contract |
| 8 | Zap Logger | [pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) | Write implementation |
| 9 | Log Reader | [pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) | GetLogs, GetLogById |
| 10 | LogEntry | [pkg/logger/zap_logger.go](file:///d:/notetaker/notefiber-BE/internal/pkg/logger/zap_logger.go) | Log structure |
| 11 | Lumberjack | (external) | File rotation |
| 12 | Log File | `logs/app.log` | JSON storage |

---

## D. Endpoint API Reference

| Method | Endpoint | Deskripsi | Auth |
|--------|----------|-----------|------|
| `GET` | `/api/admin/logs` | List system logs | Admin JWT |
| `GET` | `/api/admin/logs/:id` | Get log detail | Admin JWT |

### Query Parameters (GET /logs)

| Parameter | Type | Default | Deskripsi |
|-----------|------|---------|-----------|
| `page` | int | 1 | Page number |
| `limit` | int | 10 | Items per page |
| `level` | string | - | Filter: INFO, WARN, ERROR |

---

## E. Log File Format

```json
{"level":"INFO","timestamp":"2024-12-25T01:00:00+07:00","caller":"service/admin_service.go:225","message":"Updated user status","module":"ADMIN","details":{"userId":"550e8400-...","status":"blocked","admin":"system"}}
{"level":"ERROR","timestamp":"2024-12-25T00:55:00+07:00","caller":"service/payment_service.go:180","message":"Midtrans webhook failed","module":"PAYMENT","details":{"orderId":"...","error":"signature mismatch"}}
```

**Log Structure:**

| Field | Type | Deskripsi |
|-------|------|-----------|
| `level` | string | INFO, WARN, ERROR, DEBUG |
| `timestamp` | string | ISO8601 format |
| `caller` | string | Source file:line (auto by Zap) |
| `message` | string | Log message |
| `module` | string | Service/feature name |
| `details` | object | Additional context |

---

## F. Log Rotation Configuration

```go
rotator := &lumberjack.Logger{
    Filename:   "logs/app.log",
    MaxSize:    10,   // Rotate when file reaches 10MB
    MaxBackups: 5,    // Keep 5 old files
    MaxAge:     30,   // Delete files older than 30 days
    Compress:   true, // Compress rotated files (.gz)
}
```

**Rotation Files:**
```
logs/
â”œâ”€â”€ app.log              # Current active log
â”œâ”€â”€ app-2024-12-24.log.gz # Rotated + compressed
â”œâ”€â”€ app-2024-12-23.log.gz
â””â”€â”€ ...
```

---

## G. Log Level Hierarchy

| Level | Numeric | Use Case |
|-------|---------|----------|
| `DEBUG` | 0 | Development only, verbose |
| `INFO` | 1 | Normal operations, audit trail |
| `WARN` | 2 | Potential issues |
| `ERROR` | 3 | Errors requiring attention |

> [!NOTE]
> File log hanya menyimpan INFO dan above. DEBUG hanya muncul di console saat development.

---

## H. Audit Trail Usage

| Service | Module | Example Messages |
|---------|--------|------------------|
| AdminService | `ADMIN` | Updated user status, Deleted User |
| AdminService | `ADMIN` | Upgraded User Subscription |
| AdminService | `ADMIN` | Approved Refund Request |
| PaymentService | `PAYMENT` | Midtrans webhook failed |
| AuthService | `AUTH` | Login failed, Token refresh |

**Usage Pattern:**
```go
s.logger.Info("ADMIN", "Updated user status", map[string]interface{}{
    "userId": userId.String(),
    "status": status,
    "admin":  "system",
})
```

---

## I. Log ID Generation

Log ID menggunakan **MD5 hash** dari raw JSON line content:

```go
entry.Id = fmt.Sprintf("%x", md5.Sum(line))
// Result: "5d41402abc4b2a76b9719d911017c592"
```

> [!WARNING]
> MD5 is used for identification only, not security. IDs may collide on identical log content.

---

## J. Performance Considerations

| Aspect | Implementation | Note |
|--------|----------------|------|
| Write Performance | Zap (zero allocation) | High throughput |
| Read Performance | Sequential scan | Limited to ~100MB for MVP |
| Memory Usage | Line-by-line scan | Buffer 1MB max per line |
| File Size | Lumberjack rotation | Max 10MB per file |

> [!TIP]
> For production with large log volume, consider implementing reverse file reading or external log aggregator (ELK, Loki).

---

*Dokumen ini di-generate dalam mode READ-ONLY tanpa modifikasi terhadap kode sumber.*
