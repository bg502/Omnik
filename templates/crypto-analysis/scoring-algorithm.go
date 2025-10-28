package evaluator

import (
	"math"
	"time"
)

// Strategy represents a DeFi investment strategy
type Strategy struct {
	ID                string
	Protocol          string
	Name              string
	Chain             string
	TVL               float64
	APY               float64
	Liquidity         float64
	AuditStatus       bool
	TeamPublic        bool
	GithubURL         string
	ListedProtocols   []string
	YieldTransparency bool
	PricingMechanism  string
	IL_Risk           string
	HistoricalAPY     []APYData
	UpdatedAt         time.Time
}

// APYData represents historical APY information
type APYData struct {
	Date  time.Time
	Value float64
}

// ScoringResult contains the evaluation results
type ScoringResult struct {
	StrategyID        string
	TotalScore        float64
	PassedBasicCheck  bool
	PassedExtended    bool
	RiskLevel         string
	RecommendedAction string
	Details           ScoringDetails
	Timestamp         time.Time
}

// ScoringDetails provides breakdown of scoring
type ScoringDetails struct {
	TVLScore          float64
	LiquidityScore    float64
	AuditScore        float64
	ProtocolScore     float64
	TransparencyScore float64
	APYStabilityScore float64
	RiskAdjustedAPY   float64
	FailureReasons    []string
}

// StrategyEvaluator handles strategy evaluation logic
type StrategyEvaluator struct {
	config EvaluatorConfig
}

// EvaluatorConfig contains evaluation parameters
type EvaluatorConfig struct {
	MinTVL               float64   // $50M minimum
	MaxPoolAllocation    float64   // 5% max
	MinLiquidityRatio    float64   // 20x minimum
	RequiredProtocols    []string  // Curve, Pendle, Spectra, Morpho, etc.
	MinProtocolCount     int       // At least 2
	APYStabilityDays     int       // 7 days for stability check
	TVLAlertThreshold    float64   // 10% drop threshold
	BaseMarketRate       float64   // Current risk-free rate
}

// NewEvaluator creates a new strategy evaluator
func NewEvaluator(config EvaluatorConfig) *StrategyEvaluator {
	return &StrategyEvaluator{
		config: config,
	}
}

// EvaluateStrategy performs comprehensive strategy evaluation
func (e *StrategyEvaluator) EvaluateStrategy(strategy Strategy, portfolioValue float64) ScoringResult {
	result := ScoringResult{
		StrategyID: strategy.ID,
		Timestamp:  time.Now(),
		Details:    ScoringDetails{},
	}

	// Phase 1: Basic Requirements Check
	passedBasic := e.checkBasicRequirements(strategy, portfolioValue, &result)
	result.PassedBasicCheck = passedBasic

	if !passedBasic {
		result.RiskLevel = "REJECTED"
		result.RecommendedAction = "DO_NOT_INVEST"
		return result
	}

	// Phase 2: Extended Scoring Model
	e.calculateExtendedScore(strategy, &result)

	// Phase 3: Risk Assessment
	e.assessRisk(strategy, &result)

	// Phase 4: Generate Recommendation
	e.generateRecommendation(&result)

	return result
}

// checkBasicRequirements validates against minimum criteria
func (e *StrategyEvaluator) checkBasicRequirements(
	strategy Strategy,
	portfolioValue float64,
	result *ScoringResult,
) bool {
	failures := []string{}

	// 1. Check TVL minimum ($50M)
	if strategy.TVL < e.config.MinTVL {
		failures = append(failures, 
			fmt.Sprintf("TVL below minimum: $%.2fM < $%.2fM", 
			strategy.TVL/1e6, e.config.MinTVL/1e6))
	}

	// 2. Check pool allocation limit (5%)
	maxAllocation := portfolioValue * e.config.MaxPoolAllocation
	if maxAllocation > strategy.TVL*0.05 {
		failures = append(failures, "Position would exceed 5% of pool")
	}

	// 3. Check yield transparency
	if !strategy.YieldTransparency {
		failures = append(failures, "Yield source not transparent")
	}

	// 4. Check audit status and team
	if !strategy.AuditStatus {
		failures = append(failures, "No audit from recognized firm")
	}
	if !strategy.TeamPublic {
		failures = append(failures, "Team not public or lacks portfolio")
	}

	// 5. Check GitHub presence
	if strategy.GithubURL == "" {
		failures = append(failures, "No open GitHub repository")
	}

	// 6. Check pricing transparency
	if strategy.PricingMechanism == "" || strategy.PricingMechanism == "unknown" {
		failures = append(failures, "Pricing mechanism not transparent")
	}

	// 7. Check protocol listings (minimum 2 from approved list)
	approvedCount := 0
	for _, protocol := range strategy.ListedProtocols {
		for _, required := range e.config.RequiredProtocols {
			if protocol == required {
				approvedCount++
				break
			}
		}
	}
	if approvedCount < e.config.MinProtocolCount {
		failures = append(failures, 
			fmt.Sprintf("Listed on %d approved protocols, need %d", 
			approvedCount, e.config.MinProtocolCount))
	}

	// 8. Check liquidity ratio (20x minimum)
	requiredLiquidity := maxAllocation * e.config.MinLiquidityRatio
	if strategy.Liquidity < requiredLiquidity {
		failures = append(failures, 
			fmt.Sprintf("Insufficient liquidity: %.2fx < %.2fx required", 
			strategy.Liquidity/maxAllocation, e.config.MinLiquidityRatio))
	}

	result.Details.FailureReasons = failures
	return len(failures) == 0
}

// calculateExtendedScore computes detailed scoring metrics
func (e *StrategyEvaluator) calculateExtendedScore(
	strategy Strategy,
	result *ScoringResult,
) {
	var totalScore float64
	var weightSum float64

	// TVL Score (weight: 20%)
	tvlScore := e.calculateTVLScore(strategy.TVL)
	result.Details.TVLScore = tvlScore
	totalScore += tvlScore * 0.20
	weightSum += 0.20

	// Liquidity Score (weight: 25%)
	liquidityScore := e.calculateLiquidityScore(strategy.Liquidity, strategy.TVL)
	result.Details.LiquidityScore = liquidityScore
	totalScore += liquidityScore * 0.25
	weightSum += 0.25

	// Audit & Security Score (weight: 20%)
	auditScore := e.calculateAuditScore(strategy)
	result.Details.AuditScore = auditScore
	totalScore += auditScore * 0.20
	weightSum += 0.20

	// Protocol Diversity Score (weight: 15%)
	protocolScore := e.calculateProtocolScore(strategy.ListedProtocols)
	result.Details.ProtocolScore = protocolScore
	totalScore += protocolScore * 0.15
	weightSum += 0.15

	// Transparency Score (weight: 10%)
	transparencyScore := e.calculateTransparencyScore(strategy)
	result.Details.TransparencyScore = transparencyScore
	totalScore += transparencyScore * 0.10
	weightSum += 0.10

	// APY Stability Score (weight: 10%)
	apyStability := e.calculateAPYStability(strategy.HistoricalAPY)
	result.Details.APYStabilityScore = apyStability
	totalScore += apyStability * 0.10
	weightSum += 0.10

	// Calculate final score
	result.TotalScore = totalScore / weightSum * 100

	// Calculate risk-adjusted APY
	riskMultiplier := result.TotalScore / 100
	result.Details.RiskAdjustedAPY = strategy.APY * riskMultiplier
}

// calculateTVLScore scores based on TVL size
func (e *StrategyEvaluator) calculateTVLScore(tvl float64) float64 {
	// Logarithmic scale scoring
	// $50M = 50 points, $500M = 80 points, $5B = 100 points
	if tvl < e.config.MinTVL {
		return 0
	}

	logTVL := math.Log10(tvl / 1e6) // Convert to millions and log
	logMin := math.Log10(50)        // log(50M)
	logMax := math.Log10(5000)      // log(5B)

	score := (logTVL - logMin) / (logMax - logMin) * 50 + 50
	return math.Min(100, math.Max(50, score))
}

// calculateLiquidityScore evaluates liquidity depth
func (e *StrategyEvaluator) calculateLiquidityScore(liquidity, tvl float64) float64 {
	// Ratio of liquidity to TVL
	ratio := liquidity / tvl

	switch {
	case ratio >= 0.8:
		return 100
	case ratio >= 0.5:
		return 85
	case ratio >= 0.3:
		return 70
	case ratio >= 0.1:
		return 50
	default:
		return 30
	}
}

// calculateAuditScore evaluates security measures
func (e *StrategyEvaluator) calculateAuditScore(strategy Strategy) float64 {
	score := 0.0

	if strategy.AuditStatus {
		score += 40
	}
	if strategy.TeamPublic {
		score += 30
	}
	if strategy.GithubURL != "" {
		score += 30
	}

	return score
}

// calculateProtocolScore evaluates protocol diversity
func (e *StrategyEvaluator) calculateProtocolScore(protocols []string) float64 {
	// Count recognized protocols
	recognized := 0
	for _, protocol := range protocols {
		for _, approved := range e.config.RequiredProtocols {
			if protocol == approved {
				recognized++
				break
			}
		}
	}

	// Score based on count
	switch {
	case recognized >= 5:
		return 100
	case recognized >= 4:
		return 90
	case recognized >= 3:
		return 75
	case recognized >= 2:
		return 60
	default:
		return 40
	}
}

// calculateTransparencyScore evaluates information transparency
func (e *StrategyEvaluator) calculateTransparencyScore(strategy Strategy) float64 {
	score := 0.0

	if strategy.YieldTransparency {
		score += 40
	}
	if strategy.PricingMechanism != "" && strategy.PricingMechanism != "unknown" {
		score += 30
	}
	if strategy.IL_Risk != "unknown" {
		score += 30
	}

	return score
}

// calculateAPYStability evaluates historical APY stability
func (e *StrategyEvaluator) calculateAPYStability(history []APYData) float64 {
	if len(history) < e.config.APYStabilityDays {
		return 50 // Not enough data, neutral score
	}

	// Calculate standard deviation
	var sum, sumSquares float64
	for _, data := range history {
		sum += data.Value
		sumSquares += data.Value * data.Value
	}

	mean := sum / float64(len(history))
	variance := (sumSquares / float64(len(history))) - (mean * mean)
	stdDev := math.Sqrt(variance)

	// Calculate coefficient of variation
	cv := stdDev / mean

	// Score based on stability
	switch {
	case cv <= 0.1: // Very stable
		return 100
	case cv <= 0.2:
		return 85
	case cv <= 0.3:
		return 70
	case cv <= 0.5:
		return 50
	default:
		return 30
	}
}

// assessRisk determines overall risk level
func (e *StrategyEvaluator) assessRisk(strategy Strategy, result *ScoringResult) {
	score := result.TotalScore

	switch {
	case score >= 85:
		result.RiskLevel = "LOW"
	case score >= 70:
		result.RiskLevel = "MEDIUM-LOW"
	case score >= 55:
		result.RiskLevel = "MEDIUM"
	case score >= 40:
		result.RiskLevel = "MEDIUM-HIGH"
	default:
		result.RiskLevel = "HIGH"
	}

	// Additional risk factors
	if strategy.IL_Risk == "high" {
		result.RiskLevel = e.increaseRiskLevel(result.RiskLevel)
	}

	// Check for recent TVL drops
	if e.hasRecentTVLDrop(strategy) {
		result.RiskLevel = e.increaseRiskLevel(result.RiskLevel)
		result.Details.FailureReasons = append(result.Details.FailureReasons,
			"Recent TVL drop detected")
	}
}

// generateRecommendation creates actionable recommendation
func (e *StrategyEvaluator) generateRecommendation(result *ScoringResult) {
	if !result.PassedBasicCheck {
		result.RecommendedAction = "REJECT"
		return
	}

	switch result.RiskLevel {
	case "LOW":
		if result.Details.RiskAdjustedAPY > e.config.BaseMarketRate*1.5 {
			result.RecommendedAction = "STRONG_BUY"
		} else {
			result.RecommendedAction = "BUY"
		}
	case "MEDIUM-LOW":
		if result.Details.RiskAdjustedAPY > e.config.BaseMarketRate*2 {
			result.RecommendedAction = "BUY"
		} else {
			result.RecommendedAction = "WATCH"
		}
	case "MEDIUM":
		result.RecommendedAction = "WATCH"
	case "MEDIUM-HIGH":
		result.RecommendedAction = "CAUTION"
	case "HIGH":
		result.RecommendedAction = "AVOID"
	default:
		result.RecommendedAction = "REJECT"
	}

	// Check for extended scoring pass
	result.PassedExtended = result.TotalScore >= 60
}

// hasRecentTVLDrop checks for significant TVL decrease
func (e *StrategyEvaluator) hasRecentTVLDrop(strategy Strategy) bool {
	// This would check historical TVL data
	// For now, returning false as placeholder
	return false
}

// increaseRiskLevel moves risk up one level
func (e *StrategyEvaluator) increaseRiskLevel(current string) string {
	switch current {
	case "LOW":
		return "MEDIUM-LOW"
	case "MEDIUM-LOW":
		return "MEDIUM"
	case "MEDIUM":
		return "MEDIUM-HIGH"
	default:
		return "HIGH"
	}
}
