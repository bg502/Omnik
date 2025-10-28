import React, { useState, useEffect } from 'react';
import { 
  Card, 
  CardContent, 
  CardHeader, 
  CardTitle,
  CardDescription 
} from '@/components/ui/card';
import { 
  Tabs, 
  TabsContent, 
  TabsList, 
  TabsTrigger 
} from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { 
  ChevronRight, 
  CheckCircle2, 
  XCircle, 
  AlertCircle,
  Info,
  TrendingUp,
  Shield,
  Activity,
  Database,
  GitBranch,
  Users,
  DollarSign,
  Droplets
} from 'lucide-react';

interface DecisionData {
  strategyId: string;
  strategyName: string;
  protocol: string;
  timestamp: string;
  recommendation: string;
  totalScore: number;
  riskLevel: string;
  evaluation: {
    basic: BasicRequirements;
    extended: ExtendedScoring;
    dataSources: DataSource[];
  };
  algorithm: AlgorithmStep[];
}

interface BasicRequirements {
  passed: boolean;
  checks: RequirementCheck[];
}

interface RequirementCheck {
  name: string;
  passed: boolean;
  value: string | number;
  threshold: string | number;
  description: string;
}

interface ExtendedScoring {
  scores: ScoreComponent[];
  totalScore: number;
}

interface ScoreComponent {
  name: string;
  score: number;
  weight: number;
  weightedScore: number;
  details: string;
}

interface DataSource {
  name: string;
  endpoint: string;
  timestamp: string;
  dataPoints: number;
}

interface AlgorithmStep {
  step: number;
  name: string;
  description: string;
  formula?: string;
  result: any;
}

const DecisionTransparencyViewer: React.FC<{ decisionId: string }> = ({ decisionId }) => {
  const [decision, setDecision] = useState<DecisionData | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');

  useEffect(() => {
    fetchDecisionData(decisionId);
  }, [decisionId]);

  const fetchDecisionData = async (id: string) => {
    try {
      const response = await fetch(`/api/v1/decisions/${id}/explanation`);
      const data = await response.json();
      setDecision(data);
    } catch (error) {
      console.error('Error fetching decision data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!decision) {
    return (
      <Alert className="m-4">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>Could not load decision data</AlertDescription>
      </Alert>
    );
  }

  const getRiskLevelColor = (level: string) => {
    switch (level.toUpperCase()) {
      case 'LOW': return 'bg-green-500';
      case 'MEDIUM-LOW': return 'bg-lime-500';
      case 'MEDIUM': return 'bg-yellow-500';
      case 'MEDIUM-HIGH': return 'bg-orange-500';
      case 'HIGH': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  const getRecommendationColor = (rec: string) => {
    switch (rec.toUpperCase()) {
      case 'STRONG_BUY': return 'bg-green-600 text-white';
      case 'BUY': return 'bg-green-500 text-white';
      case 'WATCH': return 'bg-yellow-500 text-white';
      case 'CAUTION': return 'bg-orange-500 text-white';
      case 'AVOID': return 'bg-red-500 text-white';
      case 'REJECT': return 'bg-red-600 text-white';
      default: return 'bg-gray-500 text-white';
    }
  };

  const getIconForCheck = (name: string) => {
    if (name.includes('TVL')) return <DollarSign className="h-4 w-4" />;
    if (name.includes('Liquidity')) return <Droplets className="h-4 w-4" />;
    if (name.includes('Audit')) return <Shield className="h-4 w-4" />;
    if (name.includes('GitHub')) return <GitBranch className="h-4 w-4" />;
    if (name.includes('Team')) return <Users className="h-4 w-4" />;
    return <Activity className="h-4 w-4" />;
  };

  return (
    <div className="max-w-6xl mx-auto p-4 space-y-4">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-start">
            <div>
              <CardTitle className="text-2xl">
                Decision Analysis: {decision.strategyName}
              </CardTitle>
              <CardDescription>
                Protocol: {decision.protocol} | Evaluated: {new Date(decision.timestamp).toLocaleString()}
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <Badge className={getRecommendationColor(decision.recommendation)}>
                {decision.recommendation.replace('_', ' ')}
              </Badge>
              <Badge className={getRiskLevelColor(decision.riskLevel)}>
                Risk: {decision.riskLevel}
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <div className="flex justify-between mb-2">
                <span className="text-sm text-muted-foreground">Total Score</span>
                <span className="font-bold">{decision.totalScore.toFixed(2)}/100</span>
              </div>
              <Progress value={decision.totalScore} className="h-3" />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Detailed Analysis Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="requirements">Requirements</TabsTrigger>
          <TabsTrigger value="scoring">Scoring Details</TabsTrigger>
          <TabsTrigger value="algorithm">Algorithm</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Evaluation Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 bg-muted rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    {decision.evaluation.basic.passed ? (
                      <CheckCircle2 className="h-5 w-5 text-green-500" />
                    ) : (
                      <XCircle className="h-5 w-5 text-red-500" />
                    )}
                    <span className="font-semibold">Basic Requirements</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {decision.evaluation.basic.passed ? 
                      'All minimum requirements met' : 
                      `Failed ${decision.evaluation.basic.checks.filter(c => !c.passed).length} requirement(s)`
                    }
                  </p>
                </div>
                <div className="p-4 bg-muted rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <TrendingUp className="h-5 w-5 text-blue-500" />
                    <span className="font-semibold">Extended Score</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Score: {decision.evaluation.extended.totalScore.toFixed(2)}/100
                  </p>
                </div>
              </div>

              {/* Data Sources Used */}
              <div>
                <h4 className="font-semibold mb-2 flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  Data Sources Used
                </h4>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                  {decision.evaluation.dataSources.map((source, idx) => (
                    <div key={idx} className="p-2 bg-secondary rounded text-sm">
                      <div className="font-medium">{source.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {source.dataPoints} data points
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Requirements Tab */}
        <TabsContent value="requirements" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Basic Requirements Check</CardTitle>
              <CardDescription>
                Minimum criteria that must be met for strategy consideration
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {decision.evaluation.basic.checks.map((check, idx) => (
                  <div key={idx} className="flex items-start gap-3 p-3 rounded-lg bg-muted/50">
                    <div className="mt-1">
                      {check.passed ? (
                        <CheckCircle2 className="h-5 w-5 text-green-500" />
                      ) : (
                        <XCircle className="h-5 w-5 text-red-500" />
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        {getIconForCheck(check.name)}
                        <span className="font-medium">{check.name}</span>
                      </div>
                      <p className="text-sm text-muted-foreground mb-2">
                        {check.description}
                      </p>
                      <div className="flex gap-4 text-sm">
                        <span>
                          <strong>Value:</strong> {check.value}
                        </span>
                        <span>
                          <strong>Required:</strong> {check.threshold}
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Scoring Details Tab */}
        <TabsContent value="scoring" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Extended Scoring Breakdown</CardTitle>
              <CardDescription>
                Detailed scoring across multiple evaluation dimensions
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {decision.evaluation.extended.scores.map((component, idx) => (
                  <div key={idx} className="border rounded-lg p-4">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <h4 className="font-semibold">{component.name}</h4>
                        <p className="text-sm text-muted-foreground">
                          {component.details}
                        </p>
                      </div>
                      <Badge variant="outline">
                        Weight: {(component.weight * 100).toFixed(0)}%
                      </Badge>
                    </div>
                    <div className="grid grid-cols-3 gap-4 mt-3">
                      <div>
                        <span className="text-xs text-muted-foreground">Raw Score</span>
                        <div className="font-bold">{component.score.toFixed(2)}/100</div>
                      </div>
                      <div>
                        <span className="text-xs text-muted-foreground">Weight</span>
                        <div className="font-bold">{(component.weight * 100).toFixed(0)}%</div>
                      </div>
                      <div>
                        <span className="text-xs text-muted-foreground">Contribution</span>
                        <div className="font-bold">{component.weightedScore.toFixed(2)}</div>
                      </div>
                    </div>
                    <Progress value={component.score} className="h-2 mt-2" />
                  </div>
                ))}

                <div className="mt-4 p-4 bg-primary/10 rounded-lg">
                  <div className="flex justify-between items-center">
                    <span className="font-semibold">Total Score</span>
                    <span className="text-2xl font-bold">
                      {decision.evaluation.extended.totalScore.toFixed(2)}/100
                    </span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Algorithm Tab */}
        <TabsContent value="algorithm" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Algorithm Steps</CardTitle>
              <CardDescription>
                Step-by-step breakdown of the evaluation process
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {decision.algorithm.map((step, idx) => (
                  <div key={idx} className="relative">
                    <div className="flex gap-4">
                      <div className="flex flex-col items-center">
                        <div className="w-10 h-10 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-bold">
                          {step.step}
                        </div>
                        {idx < decision.algorithm.length - 1 && (
                          <div className="w-0.5 h-16 bg-border mt-2" />
                        )}
                      </div>
                      <div className="flex-1 pb-8">
                        <h4 className="font-semibold mb-1">{step.name}</h4>
                        <p className="text-sm text-muted-foreground mb-2">
                          {step.description}
                        </p>
                        {step.formula && (
                          <div className="p-3 bg-muted rounded-lg font-mono text-sm mb-2">
                            {step.formula}
                          </div>
                        )}
                        <div className="p-3 bg-secondary rounded-lg">
                          <span className="text-sm font-medium">Result: </span>
                          <span className="text-sm">
                            {typeof step.result === 'object' 
                              ? JSON.stringify(step.result, null, 2)
                              : step.result
                            }
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Alert>
            <Info className="h-4 w-4" />
            <AlertTitle>Algorithm Transparency</AlertTitle>
            <AlertDescription>
              This evaluation used the Extended Scoring Model v2.1. All calculations 
              are deterministic and reproducible. The source code for this algorithm 
              is available in our GitHub repository.
            </AlertDescription>
          </Alert>
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default DecisionTransparencyViewer;
