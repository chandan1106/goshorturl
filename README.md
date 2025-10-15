# Short URL Generator Service

A high-performance URL shortening service built with Go and PostgreSQL, featuring pre-generated shortcodes using the Factory Method pattern for optimal performance and scalability.

## 🚀 Features

- **Pre-generated Shortcodes**: Uses Factory Method pattern with pre-stored shortcodes in PostgreSQL for instant URL generation
- **Multiple URL Types**: Support for Generic, Unique, and Custom shortcode generation
- **Click Analytics**: Comprehensive tracking with geolocation data
- **Campaign Management**: Organize URLs by campaign names
- **Expiry Management**: Set custom expiration times for URLs
- **User Integration Ready**: Modular design allows easy integration with user authentication systems
- **High Performance**: Fiber web framework for fast HTTP handling
- **Containerized**: Docker and Docker Compose ready deployment
- **Nginx Reverse Proxy**: Production-ready setup with load balancing

## 🏗️ Architecture

The application follows a clean architecture pattern with:
- **Factory Method Pattern**: Pre-generated shortcodes stored in database for instant retrieval
- **Modular Handlers**: Separate handlers for generation, redirection, and reporting
- **Connection Pooling**: PostgreSQL connection pooling for optimal database performance
- **Microservice Ready**: Stateless design suitable for horizontal scaling

## 📊 Database Schema

### Tables

#### 1. `shortcodes` - Factory Storage Table
```sql
CREATE TABLE shortcodes (
    id SERIAL PRIMARY KEY,
    shortcode VARCHAR(10) UNIQUE NOT NULL,
    status INTEGER DEFAULT 0,  -- 0: available, 1: taken
    taken_timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### 2. `mainurl` - URL Mapping Table
```sql
CREATE TABLE mainurl (
    id SERIAL PRIMARY KEY,
    longurl TEXT NOT NULL,
    shortcode VARCHAR(10) NOT NULL,
    expirytime TIMESTAMP NOT NULL,
    domain VARCHAR(255),
    senderid VARCHAR(100),
    createdby VARCHAR(100),
    campaignname VARCHAR(255),
    status INTEGER DEFAULT 0,  -- 0: active, 1: inactive
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### 3. `creport` - Click Analytics Table
```sql
CREATE TABLE creport (
    id SERIAL PRIMARY KEY,
    shortcode VARCHAR(10) NOT NULL,
    clicks INTEGER DEFAULT 1,
    ip INET,
    location VARCHAR(255),
    device TEXT,
    country_code VARCHAR(5),
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    organization VARCHAR(255),
    timezone VARCHAR(50),
    time TIMESTAMP DEFAULT NOW()
);
```

#### 4. `users` - User Management Table (Optional Integration)
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    status INTEGER DEFAULT 1  -- 1: active, 0: inactive
);
```

### Database Indexes (Recommended)
```sql
-- Performance indexes
CREATE INDEX idx_shortcodes_status ON shortcodes(status);
CREATE INDEX idx_mainurl_shortcode ON mainurl(shortcode);
CREATE INDEX idx_mainurl_expiry ON mainurl(expirytime);
CREATE INDEX idx_creport_shortcode ON creport(shortcode);
CREATE INDEX idx_creport_time ON creport(time);
```

## 🔧 Installation & Setup

### Prerequisites
- Go 1.23+
- PostgreSQL 16+
- Docker & Docker Compose (optional)

### Local Development Setup

1. **Clone the repository**
```bash
git clone <repository-url>
cd shorturl
```

2. **Install dependencies**
```bash
go mod download
```

3. **Setup PostgreSQL Database**
```bash
# Create database
createdb shortcodes

# Run the schema creation scripts
psql -d shortcodes -f schema.sql
```

4. **Environment Configuration**
Create a `.env` file:
```env
DATABASE_URL=postgres://username:password@localhost:5432/shortcodes?sslmode=disable
```

5. **Populate Shortcodes Factory**
```sql
-- Generate sample shortcodes (customize as needed)
INSERT INTO shortcodes (shortcode) 
SELECT 
    UPPER(
        SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 6)
    )
FROM generate_series(1, 100000);
```

6. **Run the application**
```bash
go run main.go
```

### Docker Deployment

1. **Using Docker Compose**
```bash
docker-compose up -d
```

This will start:
- PostgreSQL database on port 5434
- Go application on port 8080
- Nginx reverse proxy on port 200

2. **Manual Docker Build**
```bash
docker build -t shorturl .
docker run -p 8080:8080 shorturl
```

## 📡 API Endpoints

### 1. Generate Short URL
**POST** `/shorturl/generate`

**Request Body:**
```json
{
    "campignName": "Summer Campaign 2024",
    "mainUrl": "https://example.com/very-long-url",
    "apikey": "abcjdakfdsnkndskvn",
    "type": "unique",
    "count": "5",
    "expiry": "30",
    "senderId": "marketing",
    "domain": "https://short.ly/"
}
```

**URL Types:**
- `generic`: Single shortcode (count must be 1)
- `unique`: Multiple unique shortcodes
- `custom`: User-defined shortcode (requires `shortcode` field)

**Response:**
```json
{
    "1": "https://short.ly/marketing/ABC123",
    "2": "https://short.ly/marketing/DEF456"
}
```

### 2. Redirect Short URL
**GET** `/shorturl/{shortcode}` or `/shorturl/{senderId}/{shortcode}`

Redirects to the original URL and logs analytics data.

### 3. Get Click Reports
**POST** `/shorturl/report`

**Request Body:**
```json
{
    "campaignName": "Summer Campaign",
    "shortcode": "ABC123",
    "reportType": "detailed"
}
```

**Report Types:**
- `summary`: Aggregated click counts
- `detailed`: Individual click records with geolocation

**Response:**
```json
{
    "status": "success",
    "reports": [
        {
            "shortcode": "ABC123",
            "campaignName": "Summer Campaign 2024",
            "clicks": 150,
            "ip": "192.168.1.1",
            "country": "United States",
            "city": "New York",
            "device": "Mozilla/5.0...",
            "time": "2024-01-15 14:30:25"
        }
    ]
}
```

## 🔐 Authentication & Security

### API Key Management
- API keys are validated against the `validAPIKeys` array
- For production, integrate with the `users` table:

```go
func isValidAPIKey(key string) bool {
    var count int
    err := db.Pool.QueryRow(context.Background(), 
        "SELECT COUNT(*) FROM users WHERE api_key = $1 AND status = 1", key).Scan(&count)
    return err == nil && count > 0
}
```

### Security Features
- Input validation and sanitization
- SQL injection prevention with parameterized queries
- Rate limiting (implement with middleware)
- HTTPS enforcement (configure in Nginx)

## 👥 User Module Integration

### Adding User Authentication

1. **Update API Key Validation**
```go
func getUserByAPIKey(apiKey string) (*User, error) {
    var user User
    err := db.Pool.QueryRow(context.Background(),
        "SELECT id, username, email FROM users WHERE api_key = $1 AND status = 1",
        apiKey).Scan(&user.ID, &user.Username, &user.Email)
    return &user, err
}
```

2. **Add User Context to URL Generation**
```go
func insertMainURL(ctx context.Context, longURL, shortcode string, expiryTime time.Time, 
                  domain, senderID, userID string, campaignName string) (int, error) {
    // Include userID in the insert statement
}
```

3. **User-specific Reports**
```go
// Add WHERE clause: AND mu.created_by = $userID
```

## 🚀 Performance Optimization

### Factory Method Benefits
- **Pre-generated Pool**: Eliminates real-time shortcode generation overhead
- **Atomic Operations**: Database-level status updates prevent race conditions
- **Scalable**: Can pre-generate millions of shortcodes
- **Collision-free**: Guaranteed unique shortcodes

### Recommended Optimizations
1. **Connection Pooling**: Already implemented with pgxpool
2. **Caching**: Add Redis for frequently accessed URLs
3. **CDN**: Use CloudFront for global distribution
4. **Database Partitioning**: Partition `creport` table by date
5. **Async Analytics**: Queue click tracking for better response times

## 📈 Monitoring & Analytics

### Metrics to Track
- URL generation rate
- Click-through rates
- Geographic distribution
- Device/browser analytics
- Campaign performance
- System performance (response times, error rates)

### Geolocation Integration
The service integrates with Fortnic GeoIP API for location tracking:
- Country, region, city identification
- ISP and organization detection
- Timezone information
- Latitude/longitude coordinates

## 🔧 Configuration

### Environment Variables
```env
DATABASE_URL=postgres://user:pass@host:port/dbname?sslmode=disable
PORT=8080
GEO_API_URL=https://geoip.fortnic.com
DOMAIN=https://your-domain.com/shorturl/
```

### Nginx Configuration
The included `nginx.conf` provides:
- Reverse proxy to Go application
- Load balancing (when scaled)
- Static file serving
- Gzip compression
- Security headers

## 🚀 Deployment

### Production Checklist
- [ ] Set up SSL certificates
- [ ] Configure environment variables
- [ ] Set up database backups
- [ ] Configure monitoring (Prometheus/Grafana)
- [ ] Set up log aggregation
- [ ] Configure auto-scaling
- [ ] Set up CI/CD pipeline

### Scaling Considerations
- **Horizontal Scaling**: Stateless design allows multiple instances
- **Database Scaling**: Read replicas for analytics queries
- **Caching Layer**: Redis for hot URLs
- **CDN Integration**: Global edge locations
- **Microservices**: Split into generation, redirection, and analytics services

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the API examples

---

**Built with ❤️ using Go, PostgreSQL, and modern web technologies**