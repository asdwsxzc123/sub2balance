# Sub2Balance

Internal tool for converting Claude monthly subscriptions to balance with approval workflow.

## Features

- Staff can query subscriptions and submit conversion requests
- Admin can review and approve/reject requests
- Automatic balance addition and subscription cancellation via sub2api API
- Audit logging for all operations

## Quick Start

1. Copy config file: `cp config.yaml.example config.yaml`
2. Edit config.yaml with your sub2api credentials
3. Set environment variables:
   ```bash
   export JWT_SECRET="your-secret-here"
   export SUB2API_URL="https://your-sub2api.com"
   export SUB2API_API_KEY="admin-xxxxx"
   ```
4. Run: `go run cmd/server/main.go`
5. Access: `http://localhost:8080`
6. Default admin: `admin@sub2balance.local` / `admin123`

## Build

```bash
go build -o sub2balance cmd/server/main.go
```

## Tech Stack

- Go 1.21+ + Gin + GORM
- SQLite
- Alpine.js + TailwindCSS
- JWT authentication
