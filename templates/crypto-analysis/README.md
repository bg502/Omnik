# DeFi Portfolio Management System

An automated platform for evaluating, monitoring, and managing DeFi investment strategies with comprehensive risk assessment and transparent decision-making.

## 🎯 Overview

This system automates the complex process of DeFi strategy evaluation by implementing a multi-stage assessment framework that analyzes protocols based on security, liquidity, yield transparency, and risk factors. It provides institutional-grade portfolio management capabilities while maintaining complete transparency in the decision-making process.

## ✨ Key Features

- **Automated Strategy Evaluation**: Multi-stage scoring model with 8+ evaluation criteria
- **Real-time Monitoring**: Continuous tracking of TVL, APY, and risk metrics
- **Portfolio Rebalancing**: Automated rebalancing based on configurable triggers
- **Extensible Data Sources**: Plugin architecture for easy integration of new data providers
- **Decision Transparency**: Complete audit trail with algorithm explanation
- **Risk Management**: Comprehensive risk scoring and alerts

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Domain with nginx-proxy already configured
- API keys for data sources (DefiLlama, Dune Analytics, etc.)

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/defi-portfolio-manager
cd defi-portfolio-manager

# Configure environment
cp .env.example .env
# Edit .env with your configuration

# Start the services
docker-compose up -d

# Initialize database
docker-compose exec backend ./migrate up

# Access the dashboard
# https://your-domain.com
```

## 📊 Strategy Evaluation Criteria

### Basic Requirements (Must Pass)
- ✅ **TVL Minimum**: $50 million
- ✅ **Pool Allocation**: Maximum 5% per pool
- ✅ **Yield Transparency**: Complete visibility of yield sources
- ✅ **Security Audit**: From recognized firms
- ✅ **Team Transparency**: Public team with portfolio track record
- ✅ **Open Source**: Active GitHub repository
- ✅ **Protocol Listings**: Minimum 2 major protocols (Curve, Pendle, Spectra, Morpho, AAVE)
- ✅ **Liquidity Depth**: 20x minimum allocation

### Extended Scoring Model
- **TVL Score** (20%): Logarithmic scale evaluation
- **Liquidity Score** (25%): Depth and availability assessment
- **Security Score** (20%): Audit, team, and code transparency
- **Protocol Score** (15%): Diversity across platforms
- **Transparency Score** (10%): Information availability
- **Stability Score** (10%): Historical APY consistency

## 🏗 Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│                 │     │                  │     │                 │
│  React Frontend │────▶│   Go Backend     │────▶│  Data Sources   │
│   (Dashboard)   │     │  (Evaluation)    │     │  (APIs)         │
│                 │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
         │                       │                         │
         │                       ▼                         │
         │              ┌──────────────────┐              │
         │              │                  │              │
         └─────────────▶│   PostgreSQL     │◀─────────────┘
                        │   + TimescaleDB  │
                        │                  │
                        └──────────────────┘
                                 ▲
                                 │
                        ┌──────────────────┐
                        │                  │
                        │      Redis       │
                        │    (Cache)       │
                        │                  │
                        └──────────────────┘
```

## 🔄 Portfolio Rebalancing Rules

The system automatically triggers rebalancing when:

1. **APY Degradation**: Strategy yield < base rate for 7+ days
2. **TVL Alert**: Sharp decline indicating depeg/liquidation risk
3. **Better Opportunity**: Higher yield at comparable risk available
4. **Market Shift**: Structural changes requiring stable/ETH adjustment

## 🔌 Data Source Integration

### Built-in Providers
- DefiLlama (yields, TVL, protocols)
- Dune Analytics (on-chain metrics)
- Etherscan (contract verification)
- Direct protocol APIs

### Adding Custom Providers

```python
from plugins.base import DataSourcePlugin

class CustomProvider(DataSourcePlugin):
    def fetch_strategies(self):
        # Implementation
        pass
    
    def fetch_tvl(self, protocol):
        # Implementation
        pass
```

## 📈 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/strategies` | List evaluated strategies |
| GET | `/api/v1/strategies/{id}` | Strategy details |
| POST | `/api/v1/strategies/evaluate` | Trigger evaluation |
| GET | `/api/v1/portfolio` | Current portfolio |
| POST | `/api/v1/portfolio/rebalance` | Trigger rebalancing |
| GET | `/api/v1/decisions` | Decision audit trail |
| WS | `/ws` | Real-time updates |

## 🛡 Security Features

- Environment-based configuration
- API key encryption
- Rate limiting
- Input validation
- Audit logging
- Read-only protocol integration

## 📊 Monitoring

The system includes Grafana dashboards for:
- Strategy performance metrics
- Portfolio composition over time
- Risk indicators
- Alert history
- System health

Access monitoring at: `https://monitor.your-domain.com`

## 🧪 Testing Strategy

Before allocating capital, each strategy undergoes:

1. **Entry/Exit Analysis**: Commission and slippage assessment
2. **Reward Token Evaluation**: Stability and IL risk analysis
3. **Smart Contract Testing**: Non-mainnet verification
4. **Stability Period**: 1-week minimum test with small amounts

## 🚦 Risk Levels

| Level | Score | Action |
|-------|-------|--------|
| LOW | 85-100 | Strong Buy/Buy |
| MEDIUM-LOW | 70-84 | Buy/Watch |
| MEDIUM | 55-69 | Watch |
| MEDIUM-HIGH | 40-54 | Caution |
| HIGH | <40 | Avoid |

## 🔧 Configuration

### Environment Variables

```bash
# Required
DOMAIN=defi.example.com
DB_PASSWORD=secure_password
JWT_SECRET=secure_secret

# Data Sources
DEFILLAMA_API_URL=https://api.llama.fi
DUNE_API_KEY=your_key
ETHERSCAN_API_KEY=your_key

# Strategy Parameters
MIN_TVL_THRESHOLD=50000000
MAX_POOL_ALLOCATION=0.05
MIN_LIQUIDITY_MULTIPLIER=20

# Rebalancing
REBALANCE_CHECK_INTERVAL=3600
APY_DECLINE_THRESHOLD=7
```

## 📝 Development

```bash
# Run in development mode
docker-compose -f docker-compose.dev.yml up

# Run tests
docker-compose exec backend go test ./...

# View logs
docker-compose logs -f backend

# Database migrations
docker-compose exec backend ./migrate create add_new_field
docker-compose exec backend ./migrate up
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📄 License

MIT License - see LICENSE file for details

## 🆘 Support

- Documentation: `/docs`
- Issues: GitHub Issues
- Discord: [Join our community](https://discord.gg/example)

## 🚀 Roadmap

- [ ] Machine learning risk prediction
- [ ] Automated trading execution
- [ ] Mobile application
- [ ] Multi-chain support
- [ ] Social features
- [ ] Backtesting framework
- [ ] Integration with hardware wallets

## ⚠️ Disclaimer

This software is for informational purposes only. Always conduct your own research before making investment decisions. DeFi investments carry significant risks including total loss of capital.

---

Built with ❤️ for the DeFi community
