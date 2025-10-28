# Technical Implementation Guide
## DeFi Portfolio Management System

### Quick Start

```bash
# Clone repository
git clone https://github.com/your-org/defi-portfolio-manager
cd defi-portfolio-manager

# Configure environment
cp .env.example .env
# Edit .env with your domain and API keys

# Start services
docker-compose up -d

# Initialize database
docker-compose exec backend ./migrate up

# Access dashboard at https://your-domain.com
```

### Docker Compose Configuration

```yaml
version: '3.8'

services:
  # Go Backend - Strategy Evaluation Engine
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: defi-backend
    environment:
      # Server Configuration
      - GIN_MODE=release
      - PORT=8080
      - LOG_LEVEL=info
      
      # Database
      - DATABASE_URL=postgres://defi:${DB_PASSWORD:-defi_password}@postgres:5432/defi_portfolio?sslmode=disable
      - REDIS_URL=redis://redis:6379
      
      # Nginx Proxy Configuration
      - VIRTUAL_HOST=api.${DOMAIN}
      - VIRTUAL_PORT=8080
      - LETSENCRYPT_HOST=api.${DOMAIN}
      - LETSENCRYPT_EMAIL=${ADMIN_EMAIL}
      
      # DeFi Data Sources
      - DEFILLAMA_API_URL=https://api.llama.fi
      - DUNE_API_KEY=${DUNE_API_KEY}
      - ETHERSCAN_API_KEY=${ETHERSCAN_API_KEY}
      - ALCHEMY_API_KEY=${ALCHEMY_API_KEY}
      
      # Strategy Evaluation Parameters
      - MIN_TVL_THRESHOLD=50000000
      - MAX_POOL_ALLOCATION=0.05
      - MIN_LIQUIDITY_MULTIPLIER=20
      - REQUIRED_PROTOCOLS=Curve,Pendle,Spectra,Morpho,AAVE
      - MIN_PROTOCOL_COUNT=2
      
      # Rebalancing Parameters
      - REBALANCE_CHECK_INTERVAL=3600
      - APY_DECLINE_THRESHOLD=7
      - TVL_ALERT_THRESHOLD=0.1
      
      # Security
      - JWT_SECRET=${JWT_SECRET:-your_jwt_secret_here}
      - API_RATE_LIMIT=100
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./backend/config:/app/config:ro
      - strategy_data:/app/data
    networks:
      - internal
      - proxy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped

  # React Frontend - Dashboard
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - VITE_API_URL=https://api.${DOMAIN}
    container_name: defi-frontend
    environment:
      - VIRTUAL_HOST=${DOMAIN}
      - VIRTUAL_PORT=80
      - LETSENCRYPT_HOST=${DOMAIN}
      - LETSENCRYPT_EMAIL=${ADMIN_EMAIL}
    volumes:
      - ./frontend/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    networks:
      - proxy
    restart: unless-stopped

  # Data Aggregator - Python Service for Multiple Sources
  data-aggregator:
    build:
      context: ./data-aggregator
      dockerfile: Dockerfile
    container_name: defi-aggregator
    environment:
      - PYTHONUNBUFFERED=1
      - DATABASE_URL=postgres://defi:${DB_PASSWORD:-defi_password}@postgres:5432/defi_portfolio?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - AGGREGATION_INTERVAL=300
      - MAX_RETRIES=3
      - REQUEST_TIMEOUT=30
    depends_on:
      - postgres
      - redis
    volumes:
      - ./data-aggregator/plugins:/app/plugins:ro
      - aggregator_logs:/app/logs
    networks:
      - internal
    restart: unless-stopped

  # PostgreSQL with TimescaleDB for time-series data
  postgres:
    image: timescale/timescaledb:latest-pg15
    container_name: defi-postgres
    environment:
      - POSTGRES_USER=defi
      - POSTGRES_PASSWORD=${DB_PASSWORD:-defi_password}
      - POSTGRES_DB=defi_portfolio
      - POSTGRES_INITDB_ARGS=--encoding=UTF8
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./database/init.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
      - ./database/timescale.sql:/docker-entrypoint-initdb.d/02-timescale.sql:ro
    ports:
      - "127.0.0.1:5433:5432"
    networks:
      - internal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U defi -d defi_portfolio"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # Redis for caching and queues
  redis:
    image: redis:7-alpine
    container_name: defi-redis
    command: redis-server --appendonly yes --maxmemory 512mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    networks:
      - internal
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3
    restart: unless-stopped

  # Grafana for monitoring (optional)
  grafana:
    image: grafana/grafana:latest
    container_name: defi-grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
      - GF_INSTALL_PLUGINS=redis-datasource
      - VIRTUAL_HOST=monitor.${DOMAIN}
      - VIRTUAL_PORT=3000
      - LETSENCRYPT_HOST=monitor.${DOMAIN}
      - LETSENCRYPT_EMAIL=${ADMIN_EMAIL}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./grafana/datasources:/etc/grafana/provisioning/datasources:ro
    networks:
      - internal
      - proxy
    restart: unless-stopped

volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local
  strategy_data:
    driver: local
  aggregator_logs:
    driver: local
  grafana_data:
    driver: local

networks:
  internal:
    driver: bridge
  proxy:
    external: true
```

### Environment Configuration (.env)

```bash
# Domain Configuration
DOMAIN=defi.yourdomain.com
ADMIN_EMAIL=admin@yourdomain.com

# Database
DB_PASSWORD=secure_database_password

# Security
JWT_SECRET=your_secure_jwt_secret_here

# External APIs
DUNE_API_KEY=your_dune_api_key
ETHERSCAN_API_KEY=your_etherscan_key
ALCHEMY_API_KEY=your_alchemy_key

# Optional: Additional Data Sources
COINGECKO_API_KEY=
MESSARI_API_KEY=
THE_GRAPH_API_KEY=

# Monitoring
GRAFANA_PASSWORD=secure_grafana_password
```

### Backend Service Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── core/
│   │   ├── evaluator/      # Strategy evaluation engine
│   │   ├── rebalancer/     # Portfolio rebalancing logic
│   │   └── scorer/         # Scoring model implementation
│   ├── data/
│   │   ├── aggregator/     # Data aggregation logic
│   │   ├── providers/      # External API clients
│   │   └── cache/          # Redis caching layer
│   └── models/
│       ├── strategy.go
│       ├── portfolio.go
│       └── decision.go
├── config/
│   └── config.yaml
├── migrations/
├── Dockerfile
└── go.mod
```

### Frontend Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── Dashboard/
│   │   ├── StrategyTable/
│   │   ├── PortfolioChart/
│   │   └── DecisionViewer/
│   ├── pages/
│   │   ├── Home.tsx
│   │   ├── Strategies.tsx
│   │   ├── Portfolio.tsx
│   │   └── Settings.tsx
│   ├── services/
│   │   ├── api.ts
│   │   └── websocket.ts
│   ├── store/
│   │   └── index.ts
│   └── App.tsx
├── Dockerfile
├── nginx.conf
└── package.json
```

### Data Aggregator Plugin System

```python
# data-aggregator/plugins/base.py
from abc import ABC, abstractmethod

class DataSourcePlugin(ABC):
    """Base class for data source plugins"""
    
    @abstractmethod
    def fetch_strategies(self):
        """Fetch strategy data from the source"""
        pass
    
    @abstractmethod
    def fetch_tvl(self, protocol):
        """Fetch TVL for a specific protocol"""
        pass
    
    @abstractmethod
    def fetch_apy(self, strategy_id):
        """Fetch APY for a specific strategy"""
        pass
```

### Sample Plugin Implementation

```python
# data-aggregator/plugins/defillama.py
import requests
from .base import DataSourcePlugin

class DefiLlamaPlugin(DataSourcePlugin):
    def __init__(self, config):
        self.base_url = "https://api.llama.fi"
        self.timeout = config.get('timeout', 30)
    
    def fetch_strategies(self):
        response = requests.get(
            f"{self.base_url}/yields",
            timeout=self.timeout
        )
        data = response.json()
        
        # Transform to standard format
        strategies = []
        for item in data:
            if self._meets_criteria(item):
                strategies.append({
                    'protocol': item['project'],
                    'name': item['symbol'],
                    'tvl': item['tvlUsd'],
                    'apy': item['apy'],
                    'chain': item['chain'],
                    'il_risk': item.get('ilRisk', 'unknown')
                })
        
        return strategies
    
    def _meets_criteria(self, item):
        # Check against configured thresholds
        return (
            item.get('tvlUsd', 0) >= 50_000_000 and
            item.get('apy') is not None and
            item.get('audit', False)
        )
```

### Database Schema

```sql
-- init.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Strategies table
CREATE TABLE strategies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    protocol VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    chain VARCHAR(50) NOT NULL,
    tvl DECIMAL(20, 2) NOT NULL,
    apy DECIMAL(10, 4),
    risk_score DECIMAL(5, 2),
    audit_status BOOLEAN DEFAULT FALSE,
    github_url TEXT,
    liquidity DECIMAL(20, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Portfolio positions
CREATE TABLE portfolio_positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID REFERENCES strategies(id),
    allocation_percentage DECIMAL(5, 2),
    invested_amount DECIMAL(20, 2),
    current_value DECIMAL(20, 2),
    entry_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_rebalance TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active'
);

-- Decision logs
CREATE TABLE decision_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    action VARCHAR(50) NOT NULL,
    strategy_id UUID REFERENCES strategies(id),
    reason TEXT,
    metrics JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_strategies_tvl ON strategies(tvl DESC);
CREATE INDEX idx_strategies_apy ON strategies(apy DESC);
CREATE INDEX idx_portfolio_status ON portfolio_positions(status);
CREATE INDEX idx_decisions_created ON decision_logs(created_at DESC);
```

### API Endpoints

```yaml
# API Documentation
endpoints:
  # Strategies
  - method: GET
    path: /api/v1/strategies
    description: List all evaluated strategies
    query_params:
      - min_tvl: number
      - min_apy: number
      - chain: string
      - sort: string (tvl|apy|risk)
  
  - method: GET
    path: /api/v1/strategies/{id}
    description: Get detailed strategy information
  
  - method: POST
    path: /api/v1/strategies/evaluate
    description: Trigger manual strategy evaluation
  
  # Portfolio
  - method: GET
    path: /api/v1/portfolio
    description: Get current portfolio composition
  
  - method: POST
    path: /api/v1/portfolio/rebalance
    description: Trigger portfolio rebalancing
  
  - method: GET
    path: /api/v1/portfolio/history
    description: Get portfolio historical performance
  
  # Decisions
  - method: GET
    path: /api/v1/decisions
    description: Get decision audit trail
  
  - method: GET
    path: /api/v1/decisions/{id}/explanation
    description: Get detailed decision explanation
  
  # Data Sources
  - method: GET
    path: /api/v1/sources
    description: List configured data sources
  
  - method: POST
    path: /api/v1/sources
    description: Add new data source
  
  # WebSocket
  - method: WS
    path: /ws
    description: Real-time updates stream
```

### Deployment Commands

```bash
# Production deployment
docker-compose -f docker-compose.yml up -d

# Development with hot reload
docker-compose -f docker-compose.dev.yml up

# View logs
docker-compose logs -f backend

# Execute database migration
docker-compose exec backend ./migrate up

# Backup database
docker-compose exec postgres pg_dump -U defi defi_portfolio > backup.sql

# Update services
docker-compose pull
docker-compose up -d --build

# Scale data aggregator
docker-compose up -d --scale data-aggregator=3
```

This implementation provides a complete, production-ready DeFi portfolio management system with extensible data source integration and comprehensive monitoring capabilities.
