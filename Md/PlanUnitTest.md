# Unit Test Plan

Dokumen ini adalah rencana unit test backend setelah rework besar pada policy, workflow, audit, SLA, cache, outbox, file upload, dan WebSocket. Tujuannya bukan sekadar mengejar jumlah test, tetapi memastikan business rule penting terbukti, regression mudah ditangkap, dan klaim teknis untuk jurnal lebih kuat.

## 1. Status Test Saat Ini

Test yang sudah ada:

```text
internal/http/internal_middleware/rate_limit_test.go
internal/module/monitoring/service/wbs_validation_test.go
internal/module/realtime/websocket/controller/handler_test.go
internal/module/realtime/websocket/service/hub_test.go
internal/module/realtime/websocket/service/message_test.go
internal/module/realtime/websocket/service/publisher_test.go
internal/module/realtime/websocket/service/topic_test.go
internal/module/support/token/service/refresh_token_test.go
internal/outbox/realtime_test.go
internal/workflow/finance_test.go
internal/workflow/monitoring_test.go
internal/workflow/tracker_test.go
```

Status terakhir:

```text
go test ./... PASS
```

Status eksekusi saat ini:

```text
2026-06-27:
- menambahkan unit test policy utama dengan fake store berbasis entity.Store;
- menambahkan unit test audit service helper dan audit middleware;
- menambahkan unit test access token dan refresh token edge case;
- menambahkan unit test FSM/workflow edge case;
- menambahkan unit test SLA service;
- menambahkan unit test dashboard cache disabled/key;
- menambahkan unit test uploader helper;
- menambahkan unit test outbox payload/realtime mapper edge case;
- menambahkan unit test validator utility;
- mengubah WebSocket controller test agar skip saat environment sandbox tidak mengizinkan httptest listener.

2026-06-28:
- refactor service module utama dari concrete `*entity.SQLStore` ke interface `entity.Store`;
- service yang direfactor: project, monitoring, tracker, finance, dashboard, uploader, dan organization;
- menambahkan finance service tests untuk payment summary, payment detail mapping, dan validasi approval sebelum transaksi;
- menambahkan project service tests untuk list accessible project, configure settings validation, dan assign vendor validation;
- menambahkan monitoring service tests untuk WBS validation, leaf task mapping, delete task guard, dan parent validation;
- menambahkan tracker service tests untuk ticket list/detail mapping dan assign ticket validation;
- menambahkan dashboard service tests untuk S-Curve, stats, dan domain error mapping.
- menambahkan auth service tests untuk check email, login, set password, refresh token, logout/revoke-all guard, dan helper key/TTL refresh token;
- refactor auth controller agar bergantung interface lokal sehingga bisa dibuat handler contract test tanpa DB;
- menambahkan auth controller contract tests untuk check email, login, validation error, dan service error;
- menambahkan auth middleware tests untuk bearer token, missing/invalid token, context claims, dan token extraction;
- menambahkan rate limiter tests untuk key generation, body restore, case-insensitive email, window reset, dan client IP priority;
- menambahkan outbox processor tests untuk publish success, publish failure, dan skip event yang sudah diklaim;
- menambahkan hash utility test;
- menambahkan WBS edge tests untuk empty tree, orphan task, dan sibling overweight;
- menambahkan policy tests tambahan untuk dashboard/monitoring/tracker/finance view, pending validations, create payment request, dan close ticket.
- refactor project, finance, dan tracker controller agar dependency service berupa interface lokal;
- menambahkan project controller contract tests untuk list, get detail, create, validation error, dan assign vendor;
- menambahkan finance controller contract tests untuk summary, create payment request, validation error, dan auditor approval;
- menambahkan tracker controller contract tests untuk list ticket, create ticket, validation error, assign ticket, dan add log.
```

Catatan:

- Test WebSocket memakai `httptest`, sehingga pada environment sandbox perlu izin bind ke loopback port.
- Coverage sekarang sudah ada untuk helper/domain logic tertentu, tetapi service bisnis utama masih banyak yang belum punya unit test.

## 2. Standar Unit Test yang Diinginkan

Setiap unit test sebaiknya memenuhi standar berikut:

- table-driven untuk rule dengan banyak variasi;
- nama test menjelaskan behavior, bukan implementasi;
- test success path dan failure path;
- error code domain dicek eksplisit;
- tidak bergantung ke database/network untuk unit test murni;
- Redis/DB/external service diganti fake/mock;
- edge case wajib diuji untuk authorization, workflow transition, race/concurrency, dan data integrity;
- test tidak hanya mengecek `err == nil`, tetapi juga output penting;
- test harus bisa jalan dengan `go test ./...`;
- integration test boleh dibuat terpisah jika butuh DB/Redis nyata.

## 3. Testability Refactor yang Disarankan

Beberapa service saat ini masih menyimpan dependency sebagai concrete type `*entity.SQLStore`. Ini menyulitkan unit test karena fake store tidak bisa diinject langsung.

Refactor yang disarankan sebelum membuat banyak service test:

```go
store *entity.SQLStore
```

diubah menjadi:

```go
store entity.Store
```

Target utama:

- `internal/module/project/service`
- `internal/module/monitoring/service`
- `internal/module/tracker/service`
- `internal/module/finance/service`
- `internal/module/support/dashboard/service`
- `internal/module/support/uploader/service`

Catatan:

- `entity.Store` sudah punya `Querier` dan `ExecTx`, sehingga cocok untuk unit test dengan fake store.
- Untuk test transactional behavior, fake store bisa merekam apakah `ExecTx` dipanggil dan query apa saja yang terjadi.
- Jika refactor terlalu besar, alternatifnya adalah integration test dengan test database, tetapi itu bukan unit test murni.

## 4. Prioritas Implementasi Test

Urutan paling disarankan:

1. Policy layer.
2. Service dengan business rule high-risk: finance, monitoring, tracker.
3. Auth/token/session security.
4. Upload/file security.
5. Audit, SLA, cache, outbox.
6. HTTP handlers untuk contract request/response penting.
7. Utilities dan middleware.

## 5. Core Domain dan Cross-Cutting Tests

### 5.1 Policy Layer

File target:

```text
internal/policy/policy_test.go
```

Test yang perlu dibuat:

- `CanCreateProject`
  - allow GOV_SUPERVISOR admin;
  - reject GOV_SUPERVISOR member jika adminOnly;
  - reject role lain.
- `CanViewProject`
  - allow participant;
  - allow GOV_SUPERVISOR fallback;
  - reject non-participant.
- `CanConfigureProject`
  - allow EXECUTING_AGENCY admin;
  - reject non-admin;
  - reject role lain.
- `CanManageProjectContract`
  - allow EXECUTING_AGENCY admin;
  - reject participant non-admin.
- `CanManageTask`
  - task not found;
  - contract not found;
  - allow EA admin.
- `CanSubmitProgress`
  - allow CONTRACTOR;
  - reject non-contractor;
  - task not found.
- `CanValidateProgress`
  - allow INDEPENDENT_AUDITOR;
  - reject role lain;
  - progress log not found.
- `CanCreateTicket`
  - allow contractor participant;
  - reject non-participant;
  - reject non-contractor.
- `CanViewTicket`
  - allow project participant;
  - reject non-participant.
- `CanAssignTicket`
  - allow EXECUTING_AGENCY admin;
  - reject non-admin;
  - reject role lain.
- `CanAddTicketLog`
  - allow executing agency;
  - allow assigned gov partner;
  - reject unassigned gov partner;
  - reject contractor.
- `CanViewPaymentRequest`
  - allow project participant;
  - reject non-participant.
- `CanApprovePaymentAsAuditor`
  - allow auditor;
  - reject non-auditor.
- `CanApprovePaymentAsAgency`
  - allow executing agency;
  - reject role lain.
- `CanDisbursePayment`
  - allow loan provider;
  - reject role lain.
- `CanViewDashboardProject`
  - allow participant;
  - reject non-participant.

Standar mock:

- fake `entity.Store` yang mengembalikan `GetUserProjectAccess`, `GetUserMemberships`, `GetTaskByID`, `GetContractByID`, `GetTicketWithContract`, `GetPaymentRequestWithContract`, dan `GetProgressLogWithDetails`.

### 5.2 Workflow FSM

Sudah ada test dasar:

```text
internal/workflow/finance_test.go
internal/workflow/monitoring_test.go
internal/workflow/tracker_test.go
```

Tambahan test yang perlu dibuat:

- invalid state string;
- transition ke state yang sama;
- transition skipped stage;
- role/action mismatch;
- rejected state tidak bisa lanjut ke accepted/paid;
- closed ticket tidak bisa reopen jika rule tidak mendukung.

### 5.3 Audit

File target:

```text
internal/audit/service_test.go
internal/audit/middleware_test.go
```

Test yang perlu dibuat:

- `Record`
  - marshal `before_data` dan `after_data`;
  - handle nil before/after sebagai `{}`;
  - pass actor/user/entity fields ke query;
  - return internal error jika store gagal.
- `RecordTx`
  - memakai `q.CreateAuditLog`;
  - tidak memakai store global.
- Middleware
  - inject request metadata ke context;
  - tidak memblok request;
  - request ID kosong tetap aman.

## 6. IAM dan Auth Tests

### 6.1 Token Service

Sudah ada:

```text
internal/module/support/token/service/refresh_token_test.go
```

Tambahan:

- access token generate/validate success;
- access token invalid signature;
- access token expired;
- refresh token missing `jti`;
- refresh token missing `iat`;
- refresh token expired;
- decode without verify untuk access dan refresh.

### 6.2 Auth Service

File target:

```text
internal/module/iam/auth/service/service_test.go
```

Test yang perlu dibuat:

- `CheckEmail`
  - not found -> `NOT_FOUND`;
  - user tanpa password -> `SETUP_REQUIRED`;
  - user dengan password -> `LOGIN_READY`;
  - DB error -> internal error.
- `Login`
  - success returns access token, refresh token, user, memberships;
  - wrong email -> `INVALID_CREDENTIALS`;
  - wrong password -> `INVALID_CREDENTIALS`;
  - password not set -> `PASSWORD_NOT_SET`;
  - membership query error.
- `SetPassword`
  - success;
  - user not found;
  - password already set;
  - hash/update error.
- `RefreshToken`
  - success rotates refresh token;
  - revoked token -> `REFRESH_TOKEN_REVOKED`;
  - old token blacklisted after refresh;
  - user not found;
  - Redis nil fallback.
- `Logout`
  - success blacklists token;
  - invalid token;
  - Redis nil should return internal error because revocation requires Redis.
- `RevokeAll`
  - success writes revoke-all cutoff;
  - blacklists current refresh token;
  - invalid token.

### 6.3 Auth Controller

File target:

```text
internal/module/iam/auth/controller/handler_test.go
```

Test yang perlu dibuat:

- invalid request body returns validation error;
- `POST /check-email`;
- `POST /login`;
- `POST /refresh-token`;
- `POST /logout`;
- `POST /revoke-all`;
- service error maps to response error.

## 7. Project Module Tests

### 7.1 Project Service

File target:

```text
internal/module/project/service/service_test.go
```

Test yang perlu dibuat:

- `CreateProject`
  - success creates project, settings, contract baseline as expected;
  - empty image URLs normalized;
  - audit recorded;
  - cache invalidation not needed;
  - DB error propagation.
- `GetProjectByID`
  - success;
  - not found;
  - maps contract/settings/participants correctly.
- `UpdateProject`
  - success;
  - cache invalidated;
  - audit recorded.
- `ConfigureSettings`
  - success upsert settings;
  - invalid date/budget if service has rule;
  - audit and cache invalidation.
- `AssignExecutingAgency`
  - success;
  - organization role validation;
  - audit/outbox/cache.
- `AssignVendor`
  - success;
  - contract value set;
  - audit/cache.
- `AssignAuditor`
  - success;
  - role validation;
  - audit/cache.
- `AssignGovPartners`
  - replaces old participants;
  - validates all org roles;
  - audit/cache.

### 7.2 Project Controller

File target:

```text
internal/module/project/controller/handler_test.go
```

Test yang perlu dibuat:

- auth context missing -> unauthorized;
- invalid UUID param -> validation failed;
- policy rejection -> forbidden;
- request validation errors;
- success response code/message per route.

## 8. Monitoring Module Tests

### 8.1 WBS Validation

Sudah ada:

```text
internal/module/monitoring/service/wbs_validation_test.go
```

Tambahan:

- orphan task;
- root total less than 100;
- leaf total less than 100;
- sibling group over 100;
- self-parent;
- parent different contract;
- update parent deadline makes existing child invalid.

### 8.2 Monitoring Service

File target:

```text
internal/module/monitoring/service/service_test.go
```

Test yang perlu dibuat:

- `CreateTask`
  - success;
  - parent not found;
  - parent different contract;
  - child deadline exceeds parent;
  - weight exceeds root/leaf/sibling total;
  - audit/cache behavior if applicable.
- `UpdateTask`
  - success;
  - cycle detected;
  - self-parent;
  - child deadline invalid;
  - WBS overweight rejected.
- `DeleteTask`
  - reject if has children;
  - reject if pending progress log;
  - success invalidates dashboard cache.
- `SubmitProgress`
  - only leaf task;
  - task not found;
  - creates progress log snapshot;
  - audit recorded;
  - outbox `ProgressSubmitted`;
  - rejects non-leaf.
- `ValidateProgress`
  - accept path;
  - reject path requires/handles comment;
  - cannot validate twice;
  - FSM transition checked;
  - audit/outbox/cache.
- `GetSCurve`
  - only accepted validations count;
  - empty data;
  - rejected logs excluded.
- `ValidateWBS`
  - contract not found;
  - maps validation response exactly.

### 8.3 Monitoring Controller

File target:

```text
internal/module/monitoring/controller/handler_test.go
```

Test yang perlu dibuat:

- route-level policy rejection for create/update/delete task;
- `GET /project/{projectId}/wbs/validation`;
- invalid UUID;
- submit progress validation;
- validate progress validation.

## 9. Tracker Module Tests

### 9.1 Tracker Service

File target:

```text
internal/module/tracker/service/service_test.go
```

Test yang perlu dibuat:

- `CreateTicket`
  - success creates ticket, initial log, SLA, audit, outbox;
  - task item not found;
  - task item belongs to different contract;
  - cache invalidated.
- `GetTickets`
  - all/open/closed filters;
  - maps current status from latest log;
  - attaches SLA;
  - handles missing SLA.
- `AssignTicket`
  - FSM transition OPEN -> ASSIGNED;
  - assignee org validation;
  - SLA moved to assigned org;
  - audit/outbox/cache.
- `AddTicketLog`
  - comment only;
  - status change to IN_PROGRESS/RESOLVED;
  - invalid FSM transition;
  - gov partner assignment rule;
  - outbox only on resolved.
- `CloseTicket`
  - only RESOLVED can close;
  - SLA resolved;
  - audit/outbox/cache.
- `GetTicketByID`
  - not found;
  - logs sorted;
  - attachment URL mapping;
  - SLA mapping.

### 9.2 Tracker Controller

File target:

```text
internal/module/tracker/controller/handler_test.go
```

Test yang perlu dibuat:

- create ticket request validation;
- status filter query;
- assign ticket body validation;
- add log body validation;
- close ticket auth/policy error.

## 10. Finance Module Tests

### 10.1 Finance Service

File target:

```text
internal/module/finance/service/service_test.go
```

Test yang perlu dibuat:

- `CreatePaymentRequest`
  - success locks contract row using `GetContractByIDForUpdate`;
  - reject if vendor participant mismatch;
  - reject contract value not set;
  - reject pending payment exists;
  - reject amount exceeds contract value;
  - reject amount exceeds available amount;
  - creates audit log;
  - transaction rollback on failure.
- `GetPaymentRequests`
  - contract not found;
  - maps rows including signatures.
- `GetPaymentSummary`
  - computes paid, pending, available;
  - canSubmit false if pending exists;
  - handles unset contract value.
- `ApproveByAuditor`
  - approve path;
  - reject path;
  - invalid FSM transition;
  - audit/outbox/cache.
- `ApproveByAgency`
  - approve/reject path;
  - invalid stage rejected.
- `DisburseByLoanProvider`
  - only allowed after agency approved;
  - payment proof required;
  - outbox `PaymentPaid`;
  - cache invalidated.

### 10.2 Finance Controller

File target:

```text
internal/module/finance/controller/handler_test.go
```

Test yang perlu dibuat:

- create payment request validation;
- get list project UUID validation;
- get detail UUID validation;
- approve/reject validation;
- disburse validation;
- policy rejection.

## 11. Support Module Tests

### 11.1 SLA Service

File target:

```text
internal/module/support/sla/service/service_test.go
```

Test yang perlu dibuat:

- `StartTicket` sets due date based on configured duration;
- `MoveTicketState` updates state and owner;
- `ResolveTicket` sets resolvedAt;
- `GetTicket` returns nil if no record;
- `effectiveStatus`
  - on track;
  - warning;
  - violated;
  - resolved before due date;
  - resolved after due date remains violated.

Catatan:

- Perlu helper/test constructor untuk inject `now`, `ticketDuration`, dan `warningThreshold`.

### 11.2 Dashboard Cache

File target:

```text
internal/module/support/dashboard/cache/cache_test.go
```

Test yang perlu dibuat:

- `Enabled` false if Redis nil;
- key format for S-Curve and stats;
- `Get` cache miss;
- `Get` invalid JSON;
- `Set` marshaling error;
- `InvalidateProject` deletes both keys.

### 11.3 Dashboard Service

File target:

```text
internal/module/support/dashboard/service/service_test.go
```

Test yang perlu dibuat:

- `GetSCurve`
  - cache hit returns without DB query;
  - cache miss computes and sets cache;
  - Redis error fallback to DB;
  - empty project data.
- `GetStats`
  - cache hit/miss;
  - ticket stats mapping;
  - finance amount mapping.

### 11.4 Uploader Service

File target:

```text
internal/module/support/uploader/service/service_test.go
```

Test yang perlu dibuat:

- `prepareUpload`
  - JPEG content accepted;
  - PNG content accepted;
  - PDF content accepted;
  - header spoofing rejected based on content;
  - empty file rejected;
  - over 5MB rejected;
  - checksum is stable SHA-256;
  - file name sanitized;
  - entityType/entityId must come together;
  - organization membership checked.
- `UploadSingle`
  - Cloudinary upload failure;
  - empty Cloudinary URL;
  - metadata insert failure.

Catatan:

- Perlu interface wrapper untuk Cloudinary uploader agar bisa dimock tanpa network.

### 11.5 Mail Service

File target:

```text
internal/module/support/mail/service/service_test.go
```

Test yang perlu dibuat:

- template invite email generated correctly;
- SMTP config validation;
- send failure maps to error.

Catatan:

- Perlu interface SMTP sender agar tidak memakai network saat unit test.

## 12. Realtime WebSocket Tests

Sebagian sudah ada:

```text
internal/module/realtime/websocket/controller/handler_test.go
internal/module/realtime/websocket/service/hub_test.go
internal/module/realtime/websocket/service/message_test.go
internal/module/realtime/websocket/service/publisher_test.go
internal/module/realtime/websocket/service/topic_test.go
```

Tambahan:

- connect without token -> unauthorized;
- invalid token -> unauthorized;
- WebSocket disabled -> bad request;
- invalid JSON -> `realtime.error`;
- unknown message type -> `realtime.error`;
- unsubscribe invalid topic -> rejected;
- slow client send buffer full -> unregister;
- shutdown closes client connection;
- Redis Pub/Sub subscriber invalid payload logs and continues;
- Redis publish error falls back local hub.

## 13. Outbox Tests

Sebagian sudah ada:

```text
internal/outbox/realtime_test.go
```

Tambahan:

- `Service.Publish` stores correct event type, aggregate, payload;
- `PublishTx` uses transaction query;
- `Processor`
  - no pending events;
  - marks processing;
  - skips already claimed event;
  - unknown event type marked processed;
  - mapper error marks failed;
  - publisher error marks failed;
  - success marks processed after publish.

## 14. Middleware Tests

### 14.1 Auth Middleware

File target:

```text
internal/http/internal_middleware/auth_test.go
```

Test yang perlu dibuat:

- missing token;
- invalid token;
- token from `Authorization`;
- token from `X-Postmatic-AccessToken`;
- token from legacy query if still supported;
- context contains user claims.

### 14.2 Rate Limiter

Sudah ada:

```text
internal/http/internal_middleware/rate_limit_test.go
```

Tambahan:

- Redis path success;
- Redis error fallback memory;
- invalid JSON body still rate limited by IP;
- `Retry-After` header set.

### 14.3 Org Role Middleware

File target:

```text
internal/http/internal_middleware/org_role_test.go
```

Test yang perlu dibuat:

- missing auth context;
- invalid org ID;
- user not member;
- role mismatch;
- adminOnly reject member;
- success injects membership context.

### 14.4 Request Filter dan Logger

File target:

```text
internal/http/internal_middleware/req_filter_test.go
internal/http/internal_middleware/logger_test.go
```

Test yang perlu dibuat:

- filter parses pagination/search query;
- logger sets `X-Request-ID`;
- logger does not log query string/accessToken.

## 15. Utility dan Package Tests

Target:

```text
pkg/errs
pkg/filter
pkg/hash
pkg/pagination
pkg/response
pkg/utils
```

Test yang perlu dibuat:

- `errs`
  - each factory returns correct HTTP code/message;
  - internal error does not expose original error to client.
- `hash`
  - hash password success;
  - compare password true/false.
- `utils`
  - validator tags;
  - invalid JSON body;
  - URL/email/UUID validation.
- `pagination`
  - default page/limit;
  - max limit;
  - negative input normalized.
- `response`
  - OK envelope shape;
  - error envelope shape;
  - validation failed shape.

## 16. Handler Contract Tests

Handler tests sebaiknya dibuat untuk endpoint yang contract-nya penting untuk frontend.

Prioritas:

- Auth controller.
- Upload controller.
- Monitoring WBS validation.
- Tracker ticket routes.
- Finance payment routes.
- WebSocket controller.

Standar:

- gunakan `httptest`;
- fake service atau service dengan fake store;
- cek HTTP status;
- cek `message`;
- cek shape `data`;
- cek validation error map;
- cek auth/policy error.

## 17. Unit vs Integration Test

Unit test:

- policy;
- workflow;
- mapper;
- service rule dengan fake store;
- middleware;
- utils.

Integration test:

- sqlc query + migration dengan test PostgreSQL;
- Redis cache/pubsub dengan test Redis;
- full HTTP route + DB transaction;
- Cloudinary upload tidak perlu integration test default karena external.

Integration test disarankan diberi build tag:

```go
//go:build integration
```

Command:

```sh
go test ./... -tags=integration
```

## 18. Coverage Target

Target realistis:

- domain/workflow/policy: 90%+;
- service business rule: 80%+;
- middleware/security: 80%+;
- controllers: route penting saja, bukan semua variasi internal;
- generated sqlc tidak perlu unit coverage;
- cmd bootstrap tidak wajib unit coverage.

## 19. Roadmap Eksekusi

### Sprint 1: Foundation Testability

- [x] Refactor service `store *entity.SQLStore` ke `store entity.Store` untuk module utama.
- [x] Buat fake store helper untuk policy/service tests.
- [ ] Buat helper assertion error code.
- [x] Buat helper JWT/test config.

### Sprint 2: Policy dan Security

- [x] Policy layer tests.
- [x] Auth service tests.
- [x] Auth controller tests.
- [x] Auth middleware tests.
- [x] Rate limiter tambahan.

### Sprint 3: Monitoring dan Tracker

- [x] Monitoring service tests.
- [x] WBS edge case tambahan.
- [x] Tracker service tests.
- [x] SLA service tests.
- [x] Tracker controller tests.

### Sprint 4: Finance dan Project

- [x] Finance service tests.
- [x] Finance controller tests.
- [x] Project service tests.
- [x] Project controller tests.

### Sprint 5: Support, Outbox, Realtime

- [x] Uploader service tests.
- [x] Dashboard cache tests.
- [x] Dashboard service tests.
- [x] Outbox payload dan realtime mapper tests.
- [x] Outbox processor tests.
- [x] Realtime hardening tests.
- [x] Utility/package tests.

## 20. Definition of Done

Unit test project dianggap layak jika:

- [x] `go test ./...` pass tanpa external DB/Redis/network;
- [x] service high-risk punya test success dan failure path;
- [x] policy layer seluruh rule utama tertutup;
- [x] workflow transition seluruh invalid path tertutup;
- [x] security-sensitive auth/token/rate-limit/upload tertutup;
- [x] WebSocket topic authorization dan outbox mapper tertutup;
- [x] frontend-facing response contract penting punya handler test;
- [x] tidak ada test flaky;
- [x] tidak ada secret/token tercetak di log test;
- [x] test plan ini diperbarui saat fitur baru dibuat.
