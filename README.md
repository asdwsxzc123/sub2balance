# Sub2Balance

Internal tool for converting Claude monthly subscriptions to balance with approval workflow.

## Overview

Sub2Balance is a secure web application that enables staff to convert Claude monthly subscriptions into account balance through an approval-based workflow. The system integrates with sub2api for subscription management and provides comprehensive audit logging.

## 默认登录账号 / Default Login Credentials

首次启动时,如果数据库中没有任何用户,系统会根据 `config.yaml` 中 `admin` 段自动创建一个管理员账号:

| 字段 | 默认值 |
| --- | --- |
| Email | `admin@sub2balance.local` |
| Password | `admin123` |

> ⚠️ **强烈建议首次登录后立即修改密码**(顶部导航 → 修改密码)。
>
> - 只有当 `users` 表为空时,`config.yaml` 里的 `admin.email` / `admin.password` 才会被用来创建初始账号;之后修改 config.yaml 不会影响已有账号。
> - 想用其它邮箱/密码作为初始账号:删掉 `data/sub2balance.db` 后,改好 `config.yaml` 再启动即可。

登录后,管理员需要到 **系统设置** 页面 (`/admin/settings`) 配置 Sub2API 的 `Base URL` 和 `API Key` — 这两项已不再放在 `config.yaml`,而是保存在数据库里,可随时在网页上修改。

## Features

- **Subscription Query**: Staff can search and view subscription details before conversion
- **Conversion Requests**: Submit conversion requests with automatic validation
- **Approval Workflow**: Admin review and approval/rejection with reason tracking
- **Automated Processing**: Automatic balance addition and subscription cancellation via sub2api
- **User Management**: Admin can create and manage staff accounts
- **Audit Logging**: Complete audit trail for all operations
- **Rate Limiting**: Built-in protection against abuse
- **JWT Authentication**: Secure token-based authentication

## Quick Start

### Option 1: Download Pre-built Binary (Recommended)

**One-line install:**
```bash
curl -fsSL https://raw.githubusercontent.com/asdwsxzc123/sub2balance/main/deploy.sh | bash -s v1.0.0
```

**Manual download:**
```bash
# Linux AMD64
wget https://github.com/asdwsxzc123/sub2balance/releases/latest/download/sub2balance-linux-amd64.tar.gz
tar xzf sub2balance-linux-amd64.tar.gz
chmod +x sub2balance-linux-amd64

# Configure
cp config.yaml.example config.yaml
cp .env.example .env
nano .env  # Set JWT_SECRET, ADMIN_EMAIL, ADMIN_PASSWORD

# Run
set -a; source .env; set +a
./sub2balance-linux-amd64
```

See [Releases](https://github.com/asdwsxzc123/sub2balance/releases) for other platforms (ARM64, macOS, Windows).

### Option 2: Using Docker

```bash
# 1. Prepare config files
cp config.yaml.example config.yaml
cp .env.example .env
# Edit .env — set JWT_SECRET, ADMIN_EMAIL, ADMIN_PASSWORD

# 2. Create data directory with correct ownership (container runs as UID 1000)
mkdir -p data && sudo chown 1000:1000 data
# Or, to reuse your own UID, set SUB2BALANCE_UID / SUB2BALANCE_GID in .env

# 3. Start services
docker compose up -d

# 4. Access application
open http://localhost:8080
```

### Option 3: Build from Source

```bash
# 1. Prerequisites: Go 1.25+, Node 20+, pnpm 9+, a C compiler (for sqlite CGO)

# 2. Clone and configure
git clone https://github.com/asdwsxzc123/sub2balance.git
cd sub2balance
cp config.yaml.example config.yaml
cp .env.example .env
nano .env  # Configure required variables

# 3. Build frontend (embedded into the Go binary)
cd frontend && pnpm install --frozen-lockfile && pnpm build && cd ..

# 4. Build and run
CGO_ENABLED=1 go build -o sub2balance .
set -a; source .env; set +a
./sub2balance
```

**Bootstrap admin credentials:** the first run reads `ADMIN_EMAIL` / `ADMIN_PASSWORD` from `.env` and creates that user in the `users` table. Subsequent runs ignore these values — manage users from the admin UI.

**Important:** Change the admin password after first login. Then go to `/admin/settings` to configure the Sub2API upstream (`Base URL` + `API Key`).

### Required Environment Variables

| Variable | Purpose |
| --- | --- |
| `JWT_SECRET` | JWT signing key (generate: `openssl rand -hex 32`). Rotating invalidates all sessions. |
| `ADMIN_EMAIL` | Bootstrap admin email. Only used when the users table is empty. |
| `ADMIN_PASSWORD` | Bootstrap admin password. Only used when the users table is empty. |

### Configuration File

Edit `config.yaml` to customize. The upstream Sub2API credentials are **not** in this file — they live in the database and are managed via `/admin/settings`.

```yaml
server:
  port: 8080
  mode: release  # debug or release

database:
  path: ./data/sub2balance.db

jwt:
  secret: ${JWT_SECRET}
  expire_hours: 24

sub2api:
  timeout_seconds: 30
  max_retries: 3

security:
  bcrypt_cost: 12
  rate_limit:
    enabled: true
    requests_per_minute: 60

admin:
  email: ${ADMIN_EMAIL}
  password: ${ADMIN_PASSWORD}
```

## Usage

### For Staff Users

1. **Login**: Access the web interface and login with your credentials
2. **Query Subscription**: Enter email to search for active subscriptions
3. **Submit Request**: Review subscription details and submit conversion request
4. **Track Status**: View your requests and their approval status in "My Requests"

### For Administrators

1. **Review Requests**: View all pending conversion requests
2. **Approve/Reject**: Review details and approve or reject with reason
3. **User Management**: Create staff accounts and manage permissions
4. **Audit Logs**: View complete audit trail of all operations

### API Endpoints

#### Authentication

- `POST /api/auth/login` - Login with email/password
- `POST /api/auth/logout` - Logout current session
- `GET /api/auth/me` - Get current user info

#### Conversions (Staff)

- `POST /api/conversions/query` - Query subscription by email
- `POST /api/conversions` - Create conversion request
- `GET /api/conversions` - List my requests
- `GET /api/conversions/:id` - Get request details

#### Admin

- `GET /api/admin/conversions` - List all requests
- `PUT /api/admin/conversions/:id/approve` - Approve request
- `PUT /api/admin/conversions/:id/reject` - Reject request
- `GET /api/admin/users` - List users
- `POST /api/admin/users` - Create user
- `PUT /api/admin/users/:id` - Update user
- `DELETE /api/admin/users/:id` - Delete user
- `GET /api/admin/audit-logs` - View audit logs

## Development

### Build from Source

```bash
# Install Go dependencies
go mod download

# Build frontend (once, or whenever frontend/ changes)
cd frontend && pnpm install --frozen-lockfile && pnpm build && cd ..

# Build binary
CGO_ENABLED=1 go build -o sub2balance .

# Run tests
go test ./...
```

For frontend-only iteration, run `pnpm dev` inside `frontend/` and point it at a running backend.

### Project Structure

```
sub2balance/
├── main.go                  # Application entry point (embeds frontend/dist)
├── internal/
│   ├── config/              # Configuration management
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # HTTP middleware
│   ├── model/               # Database models
│   ├── repository/          # Data access layer
│   └── service/             # Business logic
├── frontend/                # React + Vite + Tailwind source
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── config.yaml              # Configuration file (copied from .example)
├── Dockerfile               # Multi-stage container build
└── docker-compose.yml       # Docker compose setup
```

## Deployment

### Docker Deployment

For most cases, pull the GHCR image via `docker compose` (see Quick Start Option 2). To build locally:

```bash
# Build image (multi-stage: frontend + Go binary + alpine runtime)
docker build -t sub2balance:latest .

# Run — config.yaml and .env must exist on the host
mkdir -p data && sudo chown 1000:1000 data
docker run -d \
  -p 8080:8080 \
  --user 1000:1000 \
  --env-file .env \
  -v "$(pwd)/data:/app/data" \
  -v "$(pwd)/config.yaml:/app/config.yaml:ro" \
  sub2balance:latest
```

### Production Considerations

1. **Security**:
   - Use strong JWT_SECRET (minimum 32 random characters)
   - Enable HTTPS with reverse proxy (nginx/caddy)
   - Restrict admin API key permissions in sub2api
   - Regular security updates

2. **Database**:
   - Regular backups of SQLite database
   - Consider PostgreSQL for high-traffic deployments
   - Monitor database size and performance

3. **Monitoring**:
   - Health check endpoint: `GET /health`
   - Monitor audit logs for suspicious activity
   - Set up alerts for failed conversions

4. **Backup**:
   - Backup `data/sub2balance.db` regularly
   - Store backups securely off-site
   - Test restore procedures

## Troubleshooting

### Common Issues

**Cannot connect to sub2api**
- Log in as admin and verify Base URL / API Key at `/admin/settings`
- Use the `Test` button on that page to probe connectivity
- Check API key has admin privileges and review network/firewall settings

**Login fails**
- Verify JWT_SECRET is set and consistent
- Check database contains admin user
- Review browser console for errors

**Conversion fails**
- Check sub2api API key permissions
- Verify subscription exists and is active
- Review audit logs for error details

**Rate limit errors**
- Adjust `security.rate_limit.requests_per_minute` in config
- Consider disabling rate limiting for trusted networks

### Logs

Application logs include:
- Server startup and configuration
- Authentication attempts
- Conversion request lifecycle
- Sub2api API calls and responses
- Error details and stack traces

## Security

- JWT-based authentication with configurable expiration
- Bcrypt password hashing with configurable cost
- Role-based access control (staff/admin)
- Rate limiting to prevent abuse
- Audit logging for accountability
- Input validation and sanitization

## Tech Stack

- **Backend**: Go 1.25+ with Gin web framework
- **Database**: SQLite with GORM ORM (CGO-linked via mattn/go-sqlite3)
- **Frontend**: React 18 + Vite + TailwindCSS, embedded into the Go binary via `//go:embed`
- **Authentication**: JWT with golang-jwt/jwt
- **Deployment**: Docker + Docker Compose, or single-binary releases

## License

Internal use only - Proprietary

## Support

For issues or questions, contact the development team.
