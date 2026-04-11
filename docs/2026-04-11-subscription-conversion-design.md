---
name: Subscription to Balance Conversion System Design
description: Internal tool for converting Claude monthly subscriptions to balance with approval workflow
type: design
date: 2026-04-11
---

# Subscription to Balance Conversion System Design

## Overview

An internal administrative tool that allows staff to submit requests to convert users' Claude monthly subscriptions into account balance, with a two-tier approval workflow (staff submission + admin review).

## Business Requirements

### Core Functionality
- Convert Claude monthly subscription to pay-as-you-go balance
- Conversion formula: `Converted Amount = Paid Amount - Consumed USD (1:1 deduction)`
- Two-tier workflow: Staff submits → Admin reviews → System executes
- Integration with sub2api via x-api-key authentication

### User Roles
1. **Staff (工作人员)**
   - Query subscription info by email
   - Submit conversion requests
   - View own submission history

2. **Admin (管理员)**
   - All staff permissions
   - Review pending requests (approve/reject)
   - Modify conversion amount during review
   - Add review notes
   - Manage user accounts
   - View audit logs

### Workflow
1. Staff enters user email → System queries sub2api API
2. System displays subscription info and calculates conversion amount
3. Staff confirms and submits request
4. Admin reviews: can modify amount, add notes, approve/reject
5. On approval: System calls sub2api API to add balance + cancel subscription
6. On rejection: Staff can resubmit

## Technical Architecture

### Tech Stack
- **Backend**: Go 1.21+ + Gin + GORM
- **Frontend**: Embedded HTML + Alpine.js + TailwindCSS
- **Database**: SQLite (single-file deployment)
- **Authentication**: JWT (24-hour expiry)
- **Deployment**: Single binary with embedded frontend

### Why This Stack?
- **SQLite**: Sufficient for internal tool, simplifies deployment
- **Embedded frontend**: Single binary deployment, no separate web server needed
- **Alpine.js**: Lightweight reactivity without build complexity
- **Go**: Fast, reliable, easy to deploy

## Data Model

### 1. users
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'staff')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2. conversion_requests
```sql
CREATE TABLE conversion_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_email TEXT NOT NULL,              -- sub2api user email
    sub2api_user_id INTEGER NOT NULL,      -- sub2api user ID
    subscription_id INTEGER NOT NULL,       -- sub2api subscription ID
    group_name TEXT NOT NULL,              -- subscription group name
    
    original_amount REAL NOT NULL,         -- paid amount
    consumed_amount REAL NOT NULL,         -- consumed USD
    conversion_amount REAL NOT NULL,       -- calculated conversion amount
    final_amount REAL,                     -- admin-modified amount (if changed)
    
    status TEXT NOT NULL CHECK(status IN ('pending', 'approved', 'rejected')),
    
    submitted_by INTEGER NOT NULL,         -- staff user ID
    reviewed_by INTEGER,                   -- admin user ID
    review_note TEXT,                      -- admin review note
    reviewed_at DATETIME,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (submitted_by) REFERENCES users(id),
    FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
```

### 3. audit_logs
```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id INTEGER,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL,                  -- 'create', 'approve', 'reject', 'query'
    details TEXT,                          -- JSON details
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (request_id) REFERENCES conversion_requests(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## API Design

### Authentication
- `POST /api/auth/login` - Login (email + password)
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user info

### Conversion Requests (Staff)
- `POST /api/conversions/query` - Query sub2api subscription by email
  - Request: `{ "email": "user@example.com" }`
  - Response: Subscription info + calculated conversion amount
- `POST /api/conversions` - Create conversion request
  - Request: `{ "user_email", "subscription_id", "original_amount", "consumed_amount", "conversion_amount", "sub2api_user_id", "group_name" }`
- `GET /api/conversions` - List own submissions (with status filter)
- `GET /api/conversions/:id` - Get request details

### Admin Operations
- `GET /api/admin/conversions` - List all requests (with status filter)
- `GET /api/admin/conversions/:id` - Get request details
- `PUT /api/admin/conversions/:id/approve` - Approve request
  - Request: `{ "final_amount": 100.0, "review_note": "approved" }`
  - Action: Call sub2api API to add balance + cancel subscription
- `PUT /api/admin/conversions/:id/reject` - Reject request
  - Request: `{ "review_note": "reason for rejection" }`
- `GET /api/admin/audit-logs` - List audit logs (with pagination)

### User Management (Admin)
- `GET /api/admin/users` - List users
- `POST /api/admin/users` - Create user
- `PUT /api/admin/users/:id` - Update user
- `DELETE /api/admin/users/:id` - Delete user

## Frontend Pages

### Public
- `/login` - Login page

### Staff Pages
- `/` - Home: Query subscription and create request
- `/my-requests` - My submissions list

### Admin Pages
- `/admin/pending` - Pending requests (default view)
- `/admin/requests` - All requests (with status filter)
- `/admin/users` - User management
- `/admin/logs` - Audit logs

### UI Features
- Responsive design (mobile-friendly)
- Alpine.js for dynamic interactions
- TailwindCSS styling
- Light/dark theme toggle
- Real-time form validation

## Sub2API Integration

### Required APIs

**1. Query Active Subscription**
```
GET /api/v1/admin/users/:id
Headers: x-api-key: admin-<key>
```
Need to find subscription by email first, then get user details.

**2. Get Subscription Details**
```
GET /api/v1/admin/subscriptions/:id
Headers: x-api-key: admin-<key>
```
Extract: `original_amount`, `consumed_amount` (daily/weekly/monthly used)

**3. Add Balance**
```
POST /api/v1/admin/users/:id/balance
Headers: 
  x-api-key: admin-<key>
  Idempotency-Key: <unique-key>
Body: {
  "balance": 100.0,
  "operation": "add",
  "notes": "Converted from subscription #123"
}
```

**4. Cancel Subscription**
```
DELETE /api/v1/admin/subscriptions/:id
Headers: x-api-key: admin-<key>
```

### Error Handling Strategy

**Scenario 1: Query fails**
- Retry up to 3 times with exponential backoff
- Show user-friendly error message
- Log error details

**Scenario 2: Balance added, but subscription cancellation fails**
- Mark request as "partially_completed"
- Store error details in audit log
- Provide admin interface to retry cancellation
- Alert admin via log

**Scenario 3: Subscription already cancelled**
- Check subscription status before processing
- If already cancelled, reject request with clear message

**Scenario 4: Insufficient balance calculation**
- If consumed_amount >= original_amount, conversion_amount = 0
- Show warning to staff before submission
- Admin can still approve with modified amount

## Configuration Management

### config.yaml
```yaml
server:
  port: 8080
  mode: release  # debug/release

database:
  path: ./data/sub2balance.db

jwt:
  secret: ${JWT_SECRET}  # from env var
  expire_hours: 24

sub2api:
  base_url: ${SUB2API_URL}  # from env var
  api_key: ${SUB2API_API_KEY}  # from env var
  timeout_seconds: 30
  max_retries: 3

security:
  bcrypt_cost: 12
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

### Environment Variables (Required)
- `JWT_SECRET` - JWT signing secret
- `SUB2API_URL` - sub2api base URL
- `SUB2API_API_KEY` - sub2api admin API key (admin-<64hex>)

### Configuration Priority
1. Environment variables (highest)
2. config.yaml
3. Default values (lowest)

## Security Measures

### Authentication & Authorization
- JWT token with 24-hour expiry
- Password hashing with bcrypt (cost 12)
- Role-based access control (RBAC)
- Admin operations require admin role

### API Security
- Rate limiting (60 requests/minute per IP)
- CORS configuration (whitelist only)
- Request size limits
- SQL injection prevention (GORM parameterized queries)
- XSS prevention (HTML escaping)

### Data Security
- sub2api API key encrypted storage
- Sensitive config from environment variables
- Audit logs for all critical operations
- No plaintext passwords in database

### Operational Security
- Admin operations require confirmation dialog
- Idempotency keys for balance operations
- Transaction rollback on partial failures
- Detailed error logging (without exposing internals to users)

## Deployment

### Build Process
```bash
# Build frontend (if using build step)
cd frontend && npm run build

# Build Go binary with embedded frontend
cd backend
go build -tags embed -o sub2balance ./cmd/server

# Output: single binary file
```

### Deployment Options

**Option 1: Binary Deployment (Recommended)**
```bash
# 1. Copy binary and config
./sub2balance
./config.yaml
./data/  # SQLite database directory (auto-created)

# 2. Set environment variables
export JWT_SECRET="your-secret"
export SUB2API_URL="https://your-sub2api.com"
export SUB2API_API_KEY="admin-xxxxx"

# 3. Run
./sub2balance
```

**Option 2: Docker Deployment**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -tags embed -o sub2balance ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sub2balance .
COPY config.yaml .
EXPOSE 8080
CMD ["./sub2balance"]
```

**Option 3: systemd Service**
```ini
[Unit]
Description=Sub2Balance Service
After=network.target

[Service]
Type=simple
User=sub2balance
WorkingDirectory=/opt/sub2balance
Environment="JWT_SECRET=xxx"
Environment="SUB2API_URL=xxx"
Environment="SUB2API_API_KEY=xxx"
ExecStart=/opt/sub2balance/sub2balance
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Initialization
- First run: Auto-create database tables
- Default admin account: `admin@sub2balance.local` / `admin123`
- Force password change on first login

### Backup Strategy
- SQLite database: Simple file copy
- Recommended: Daily automated backup of `./data/sub2balance.db`
- Config backup: Version control `config.yaml` (without secrets)

## Testing Strategy

### Unit Tests
- Conversion amount calculation logic
- JWT middleware authentication
- Password hashing/verification
- sub2api API client wrapper

### Integration Tests
- Complete workflow: submit → review → execute
- Error scenarios: API failures, network timeouts
- Idempotency: Duplicate balance operations
- Authorization: Role-based access control

### Manual Test Checklist
- [ ] Staff can query subscription by email
- [ ] Staff can submit conversion request
- [ ] Staff cannot access admin pages
- [ ] Admin can approve request with modified amount
- [ ] Admin can reject request with note
- [ ] Rejected request can be resubmitted
- [ ] Balance added successfully to sub2api
- [ ] Subscription cancelled successfully
- [ ] Audit logs recorded correctly
- [ ] Error handling displays user-friendly messages
- [ ] Rate limiting works
- [ ] JWT expiry enforced

## Project Structure

```
sub2balance/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── model/
│   │   ├── user.go
│   │   ├── conversion_request.go
│   │   └── audit_log.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── conversion_repo.go
│   │   └── audit_repo.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── conversion_service.go
│   │   ├── sub2api_client.go   # sub2api API wrapper
│   │   └── audit_service.go
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── conversion_handler.go
│   │   ├── admin_handler.go
│   │   └── user_handler.go
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication
│   │   ├── role.go              # Role-based authorization
│   │   └── rate_limit.go
│   └── web/
│       └── dist/                # Embedded frontend files
├── web/
│   ├── index.html
│   ├── login.html
│   ├── staff/
│   │   ├── home.html
│   │   └── my-requests.html
│   └── admin/
│       ├── pending.html
│       ├── requests.html
│       ├── users.html
│       └── logs.html
├── config.yaml.example
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Implementation Phases

### Phase 1: Core Backend (Week 1)
- Database models and migrations
- Authentication (JWT)
- User management
- Basic CRUD for conversion requests

### Phase 2: Sub2API Integration (Week 1)
- API client wrapper
- Query subscription logic
- Balance addition + subscription cancellation
- Error handling and retry logic

### Phase 3: Frontend (Week 2)
- Login page
- Staff pages (query + submit)
- Admin pages (review + approve/reject)
- User management UI

### Phase 4: Testing & Deployment (Week 2)
- Unit tests
- Integration tests
- Manual testing
- Documentation
- Deployment scripts

## Success Criteria

- Staff can successfully query subscriptions and submit requests
- Admin can review and approve/reject with amount modification
- System correctly calls sub2api APIs to add balance and cancel subscription
- All operations are logged in audit trail
- Error scenarios are handled gracefully
- Single binary deployment works on Linux/macOS
- Documentation is complete and clear

## Future Enhancements (Out of Scope)

- Email notifications for request status changes
- Batch processing multiple conversions
- Export audit logs to CSV
- Dashboard with statistics
- Multi-language support
- OAuth integration for authentication
