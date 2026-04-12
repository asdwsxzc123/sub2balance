# Sub2Balance

Internal tool for converting Claude monthly subscriptions to balance with approval workflow.

## Overview

Sub2Balance is a secure web application that enables staff to convert Claude monthly subscriptions into account balance through an approval-based workflow. The system integrates with sub2api for subscription management and provides comprehensive audit logging.

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

### Using Docker (Recommended)

```bash
# 1. Clone and configure
cp .env.example .env
# Edit .env with your credentials

# 2. Start services
docker-compose up -d

# 3. Access application
open http://localhost:8080
```

Default admin credentials: `admin@sub2balance.local` / `admin123`

### Manual Setup

```bash
# 1. Prerequisites
# - Go 1.25+
# - SQLite3

# 2. Configure
cp config.yaml.example config.yaml
# Edit config.yaml with your settings

# 3. Set environment variables
export JWT_SECRET="your-secret-key-min-32-chars"
export SUB2API_URL="https://your-sub2api.com"
export SUB2API_API_KEY="admin-xxxxx"

# 4. Build and run
go build -o sub2balance cmd/server/main.go
./sub2balance

# 5. Access
open http://localhost:8080
```

## Configuration

### Environment Variables

Required environment variables:

- `JWT_SECRET`: Secret key for JWT token signing (minimum 32 characters)
- `SUB2API_URL`: Base URL of your sub2api instance
- `SUB2API_API_KEY`: Admin API key for sub2api (must have admin privileges)

Optional:

- `PORT`: Server port (default: 8080)
- `GIN_MODE`: Gin mode - `debug` or `release` (default: release)

### Configuration File

Edit `config.yaml` to customize:

```yaml
server:
  port: 8080
  mode: release

database:
  path: ./data/sub2balance.db

jwt:
  secret: ${JWT_SECRET}
  expire_hours: 24

sub2api:
  base_url: ${SUB2API_URL}
  api_key: ${SUB2API_API_KEY}
  timeout_seconds: 30
  max_retries: 3

security:
  bcrypt_cost: 12
  rate_limit:
    enabled: true
    requests_per_minute: 60
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
# Install dependencies
go mod download

# Build binary
go build -o sub2balance cmd/server/main.go

# Run tests
go test ./...

# Run with hot reload (requires air)
air
```

### Project Structure

```
sub2balance/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # HTTP middleware
│   ├── model/               # Database models
│   ├── repository/          # Data access layer
│   └── service/             # Business logic
├── web/                     # Frontend assets (embedded)
│   ├── assets/
│   ├── index.html
│   ├── login.html
│   └── ...
├── config.yaml              # Configuration file
├── Dockerfile               # Container image
└── docker-compose.yml       # Docker compose setup
```

## Deployment

### Docker Deployment

```bash
# Build image
docker build -t sub2balance:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e JWT_SECRET="your-secret" \
  -e SUB2API_URL="https://api.example.com" \
  -e SUB2API_API_KEY="admin-key" \
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
- Verify SUB2API_URL is correct and accessible
- Check API key has admin privileges
- Review network/firewall settings

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
- **Database**: SQLite with GORM ORM
- **Frontend**: Alpine.js + TailwindCSS
- **Authentication**: JWT with golang-jwt/jwt
- **Deployment**: Docker + Docker Compose

## License

Internal use only - Proprietary

## Support

For issues or questions, contact the development team.
