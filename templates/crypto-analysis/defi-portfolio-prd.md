# Product Requirements Document
## DeFi Portfolio Management and Analysis System

### 1. Executive Summary

The DeFi Portfolio Management System is an automated platform designed to evaluate, monitor, and manage DeFi investment strategies across multiple protocols. The system implements a comprehensive scoring model to assess investment opportunities based on security, liquidity, yield transparency, and risk factors.

### 2. Product Overview

#### 2.1 Purpose
Automate the evaluation and management of DeFi strategies by implementing a rule-based system that:
- Screens DeFi protocols and strategies against predefined criteria
- Calculates risk-adjusted returns using an extended scoring model
- Triggers portfolio rebalancing based on market conditions
- Provides transparent decision-making process visibility

#### 2.2 Key Features
- **Strategy Evaluation Engine**: Automated assessment of DeFi protocols
- **Portfolio Management**: Allocation and rebalancing automation
- **Risk Monitoring**: Real-time alerts and risk assessment
- **Decision Transparency**: Full audit trail of investment decisions
- **Data Integration**: Support for multiple DeFi data sources

### 3. Functional Requirements

#### 3.1 Core Evaluation Criteria

##### 3.1.1 Basic Asset Requirements
- **Pool Allocation Limit**: Maximum 5% allocation per pool
- **Yield Transparency**: Complete transparency of yield generation mechanism
- **Audit Verification**: 
  - Minimum one audit from recognized firm
  - Public team with portfolio track record
- **GitHub Transparency**: Open-source repositories with active development
- **Pricing Transparency**: Full visibility into asset pricing and pegging mechanisms
- **TVL Minimum**: $50 million minimum Total Value Locked
- **Protocol Integration**: Listing on at least 2 major DeFi protocols:
  - Curve
  - Pendle
  - Spectra
  - Morpho
  - AAVE
  - Others as configured
- **Liquidity Requirement**: Minimum 20x application allocation in combined liquidity

##### 3.1.2 Extended Scoring Model
The system evaluates strategies through multiple stages as indicated in the scoring model link.

#### 3.2 Strategy Testing Requirements

Before allocating funds, each strategy must undergo:

##### 3.2.1 Entry/Exit Analysis
- Commission impact assessment
- Slippage estimation for non-obvious costs

##### 3.2.2 Rewarder Evaluation
- Assessment of reward token value stability
- Analysis of potential impermanent loss issues

##### 3.2.3 Non-EVM Testing
- Smart contract connection testing without mainnet deployment
- Asset withdrawal verification

##### 3.2.4 Stability Testing
- Minimum one-week test period on small amounts
- APY stability verification for new instruments

#### 3.3 Portfolio Management Rules

##### 3.3.1 Rebalancing Triggers
Initiate rebalancing when:
- Strategy yield drops below base market rate for 7+ days
- TVL sharp decline alert (depeg risk, liquidation risk, health factor deterioration)
- New higher-yield opportunity with comparable risk emerges
- Market structure shift requiring stable/ETH exposure adjustment

##### 3.3.2 Secondary Scoring Conditions
- **Transition Criteria**: Define clear conditions for strategy migration
- **Protocol Presence**: Verify instrument availability on non-scam protocols
- **Emitter Audit**: Assess new protocol reliability

### 4. Technical Architecture

#### 4.1 System Components

##### 4.1.1 Backend Service
- **Language**: Go (Gin framework)
- **Purpose**: 
  - Strategy evaluation engine
  - Portfolio management logic
  - Data aggregation and processing
  - API endpoints for frontend
- **Key Functions**:
  - Real-time strategy scoring
  - Portfolio rebalancing algorithms
  - Risk assessment calculations
  - Alert generation

##### 4.1.2 Frontend Dashboard
- **Framework**: React with Vite
- **Features**:
  - Strategy overview dashboard
  - Portfolio composition visualization
  - Decision audit trail viewer
  - Risk metrics display
  - Manual override controls
  - Configuration management

##### 4.1.3 Data Layer
- **PostgreSQL**: 
  - Historical strategy performance
  - Portfolio transactions
  - Audit logs
  - Configuration storage
- **Redis**: 
  - Real-time market data caching
  - Session management
  - Rate limiting

##### 4.1.4 Data Integration Service
- **Purpose**: Aggregate data from multiple sources
- **Initial Sources**:
  - DefiLlama API
  - Direct protocol APIs (Curve, Pendle, etc.)
  - On-chain data via RPC nodes
- **Extensible Architecture**: Plugin system for adding new data sources

#### 4.2 Docker Deployment

```yaml
version: '3.8'

services:
  # DeFi Analysis Backend
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://defi:password@postgres:5432/defi_db?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - VIRTUAL_HOST=${DOMAIN}
      - VIRTUAL_PORT=8080
      - LETSENCRYPT_HOST=${DOMAIN}
      - LETSENCRYPT_EMAIL=${EMAIL}
      # DeFi Data Sources
      - DEFILLAMA_API=https://api.llama.fi
      - DUNE_API_KEY=${DUNE_API_KEY}
      # Strategy Parameters
      - MIN_TVL=50000000
      - MAX_POOL_ALLOCATION=0.05
      - MIN_LIQUIDITY_RATIO=20
    depends_on:
      - postgres
      - redis
    networks:
      - internal
      - proxy

  # Frontend Dashboard
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    environment:
      - VITE_API_URL=https://${DOMAIN}/api
    volumes:
      - ./frontend/dist:/usr/share/nginx/html
    networks:
      - internal

  # PostgreSQL Database
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=defi
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=defi_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - internal

  # Redis Cache
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    networks:
      - internal

volumes:
  postgres_data:
  redis_data:

networks:
  internal:
    driver: bridge
  proxy:
    external: true
```

### 5. Data Model

#### 5.1 Core Entities

##### Strategy
```json
{
  "id": "uuid",
  "protocol": "string",
  "name": "string",
  "tvl": "number",
  "apy": "number",
  "risk_score": "number",
  "audit_status": "boolean",
  "github_url": "string",
  "liquidity": "number",
  "last_updated": "timestamp"
}
```

##### Portfolio Position
```json
{
  "id": "uuid",
  "strategy_id": "uuid",
  "allocation_percentage": "number",
  "invested_amount": "number",
  "current_value": "number",
  "entry_date": "timestamp",
  "last_rebalance": "timestamp"
}
```

##### Decision Log
```json
{
  "id": "uuid",
  "action": "string",
  "strategy_id": "uuid",
  "reason": "string",
  "metrics": "json",
  "timestamp": "timestamp"
}
```

### 6. User Interface Requirements

#### 6.1 Dashboard View
- **Portfolio Overview**: Current allocation pie chart
- **Strategy Table**: Sortable list with all evaluated strategies
- **Risk Metrics**: Overall portfolio risk indicators
- **Performance Graph**: Historical returns visualization

#### 6.2 Strategy Details View
- **Scoring Breakdown**: Detailed scoring model results
- **Audit Trail**: Historical decisions for this strategy
- **Risk Factors**: Identified risks and mitigations
- **Liquidity Analysis**: Real-time liquidity metrics

#### 6.3 Configuration Panel
- **Data Sources**: Add/remove data provider integrations
- **Risk Parameters**: Adjust thresholds and limits
- **Rebalancing Rules**: Configure trigger conditions
- **Alert Settings**: Notification preferences

#### 6.4 Decision Transparency View
- **Algorithm Explanation**: Step-by-step decision process
- **Data Sources Used**: Which APIs provided input data
- **Calculation Details**: Mathematical formulas applied
- **Override History**: Manual interventions logged

### 7. Integration Requirements

#### 7.1 Data Source Plugin System
- **Plugin Interface**: Standardized API for data providers
- **Authentication**: Secure storage of API keys
- **Rate Limiting**: Respect provider limits
- **Fallback Logic**: Handle provider failures gracefully

#### 7.2 Initial Integrations
- DefiLlama API (yields endpoint)
- Dune Analytics (on-chain metrics)
- Direct protocol APIs where available
- Etherscan/Similar block explorers

### 8. Security Requirements

#### 8.1 Application Security
- Environment variable management for sensitive data
- Encrypted storage of API keys
- Rate limiting on all endpoints
- Input validation and sanitization

#### 8.2 Investment Security
- Read-only integration with protocols
- No direct fund management initially
- Audit log immutability
- Multi-signature support for future trading

### 9. Performance Requirements

- **Data Refresh**: Every 5 minutes for active strategies
- **Decision Latency**: < 1 second for scoring calculations
- **Dashboard Load**: < 2 seconds initial load
- **Concurrent Users**: Support 100+ simultaneous users

### 10. Monitoring and Alerts

#### 10.1 System Monitoring
- Application health checks
- API integration status
- Database performance metrics
- Error rate tracking

#### 10.2 Investment Alerts
- TVL sudden changes (>10% in 1 hour)
- APY degradation below thresholds
- New strategy opportunities
- Rebalancing recommendations

### 11. MVP Scope

#### Phase 1: Core Evaluation Engine
- Basic scoring model implementation
- DefiLlama integration
- PostgreSQL data storage
- Simple REST API

#### Phase 2: Dashboard Development
- React frontend with basic views
- Portfolio visualization
- Strategy comparison table
- Docker deployment setup

#### Phase 3: Advanced Features
- Multiple data source support
- Automated rebalancing logic
- Alert system
- Decision transparency view

### 12. Success Metrics

- **Accuracy**: 90%+ correlation with manual evaluation
- **Coverage**: Track 100+ DeFi strategies
- **Performance**: Identify opportunities within 15 minutes
- **Reliability**: 99.9% uptime
- **User Adoption**: 50+ active users within 3 months

### 13. Future Enhancements

- Automated trading execution
- Machine learning for risk prediction
- Social features for strategy sharing
- Mobile application
- Multi-chain support expansion
- Backtesting framework
