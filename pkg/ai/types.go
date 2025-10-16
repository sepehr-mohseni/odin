package ai

import (
	"time"
)

// TrafficPattern represents aggregated traffic metrics for pattern recognition
type TrafficPattern struct {
	Timestamp       time.Time         `json:"timestamp" bson:"timestamp"`
	ServiceName     string            `json:"service_name" bson:"service_name"`
	Endpoint        string            `json:"endpoint" bson:"endpoint"`
	Method          string            `json:"method" bson:"method"`
	RequestCount    int64             `json:"request_count" bson:"request_count"`
	ErrorCount      int64             `json:"error_count" bson:"error_count"`
	ErrorRate       float64           `json:"error_rate" bson:"error_rate"`
	AvgLatency      float64           `json:"avg_latency" bson:"avg_latency"`
	P95Latency      float64           `json:"p95_latency" bson:"p95_latency"`
	P99Latency      float64           `json:"p99_latency" bson:"p99_latency"`
	StatusCodes     map[string]int64  `json:"status_codes" bson:"status_codes"`
	UserAgents      map[string]int64  `json:"user_agents" bson:"user_agents"`
	SourceIPs       map[string]int64  `json:"source_ips" bson:"source_ips"`
	RequestSizes    []int64           `json:"request_sizes" bson:"request_sizes"`
	ResponseSizes   []int64           `json:"response_sizes" bson:"response_sizes"`
	UniqueUsers     int64             `json:"unique_users" bson:"unique_users"`
	AvgRequestSize  float64           `json:"avg_request_size" bson:"avg_request_size"`
	AvgResponseSize float64           `json:"avg_response_size" bson:"avg_response_size"`
	Tags            map[string]string `json:"tags" bson:"tags"`
}

// Anomaly represents a detected traffic anomaly
type Anomaly struct {
	ID               string                 `json:"id" bson:"_id,omitempty"`
	Timestamp        time.Time              `json:"timestamp" bson:"timestamp"`
	ServiceName      string                 `json:"service_name" bson:"service_name"`
	Endpoint         string                 `json:"endpoint" bson:"endpoint"`
	AnomalyType      string                 `json:"anomaly_type" bson:"anomaly_type"`
	Severity         string                 `json:"severity" bson:"severity"` // low, medium, high, critical
	Score            float64                `json:"score" bson:"score"`       // 0-100
	Description      string                 `json:"description" bson:"description"`
	Details          map[string]interface{} `json:"details" bson:"details"`
	BaselineSnapshot map[string]interface{} `json:"baseline_snapshot,omitempty" bson:"baseline_snapshot,omitempty"`
	Current          *TrafficPattern        `json:"current,omitempty" bson:"current,omitempty"`
	Resolved         bool                   `json:"resolved" bson:"resolved"`
	ResolvedAt       *time.Time             `json:"resolved_at,omitempty" bson:"resolved_at,omitempty"`
	FalsePositive    bool                   `json:"false_positive" bson:"false_positive"`
	NotifiedAt       *time.Time             `json:"notified_at,omitempty" bson:"notified_at,omitempty"`
	Tags             map[string]string      `json:"tags" bson:"tags"`
}

// AnomalyType constants
const (
	AnomalyTypeErrorSpike       = "error_spike"
	AnomalyTypeLatencySpike     = "latency_spike"
	AnomalyTypeTrafficSpike     = "traffic_spike"
	AnomalyTypeTrafficDrop      = "traffic_drop"
	AnomalyTypeUnusualPattern   = "unusual_pattern"
	AnomalyTypeSuspiciousIP     = "suspicious_ip"
	AnomalyTypeRateLimitAbuse   = "rate_limit_abuse"
	AnomalyTypeDDoS             = "ddos_attack"
	AnomalyTypeBotActivity      = "bot_activity"
	AnomalyTypeDataExfiltration = "data_exfiltration"
)

// Severity levels
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Baseline represents statistical baseline for normal traffic
type Baseline struct {
	ServiceName       string            `json:"service_name" bson:"service_name"`
	Endpoint          string            `json:"endpoint" bson:"endpoint"`
	TimeWindow        string            `json:"time_window" bson:"time_window"` // hourly, daily, weekly
	StartTime         time.Time         `json:"start_time" bson:"start_time"`
	EndTime           time.Time         `json:"end_time" bson:"end_time"`
	SampleSize        int64             `json:"sample_size" bson:"sample_size"`
	AvgRequestRate    float64           `json:"avg_request_rate" bson:"avg_request_rate"`
	StdDevRequestRate float64           `json:"stddev_request_rate" bson:"stddev_request_rate"`
	AvgErrorRate      float64           `json:"avg_error_rate" bson:"avg_error_rate"`
	StdDevErrorRate   float64           `json:"stddev_error_rate" bson:"stddev_error_rate"`
	AvgLatency        float64           `json:"avg_latency" bson:"avg_latency"`
	StdDevLatency     float64           `json:"stddev_latency" bson:"stddev_latency"`
	P95Latency        float64           `json:"p95_latency" bson:"p95_latency"`
	P99Latency        float64           `json:"p99_latency" bson:"p99_latency"`
	CommonUserAgents  []string          `json:"common_user_agents" bson:"common_user_agents"`
	CommonIPs         []string          `json:"common_ips" bson:"common_ips"`
	LastUpdated       time.Time         `json:"last_updated" bson:"last_updated"`
	Tags              map[string]string `json:"tags" bson:"tags"`
}

// AIConfig represents configuration for AI-powered analysis
type AIConfig struct {
	Enabled               bool              `yaml:"enabled" json:"enabled"`
	AnalysisInterval      time.Duration     `yaml:"analysis_interval" json:"analysis_interval"`
	BaselineWindow        time.Duration     `yaml:"baseline_window" json:"baseline_window"`
	AnomalyThreshold      float64           `yaml:"anomaly_threshold" json:"anomaly_threshold"` // Z-score threshold
	MinSamplesForBaseline int               `yaml:"min_samples_for_baseline" json:"min_samples_for_baseline"`
	UseGrokModel          bool              `yaml:"use_grok_model" json:"use_grok_model"`
	GrokServiceURL        string            `yaml:"grok_service_url" json:"grok_service_url"`
	GrokTimeout           time.Duration     `yaml:"grok_timeout" json:"grok_timeout"`
	EnableAlerts          bool              `yaml:"enable_alerts" json:"enable_alerts"`
	AlertWebhookURL       string            `yaml:"alert_webhook_url" json:"alert_webhook_url"`
	RetentionDays         int               `yaml:"retention_days" json:"retention_days"`
	Tags                  map[string]string `yaml:"tags" json:"tags"`
}

// GrokRequest represents a request to Grok inference service
type GrokRequest struct {
	Prompt      string                 `json:"prompt"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// GrokResponse represents a response from Grok inference service
type GrokResponse struct {
	Response    string                 `json:"response"`
	Confidence  float64                `json:"confidence,omitempty"`
	Anomalies   []string               `json:"anomalies,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AnalysisResult represents the result of traffic analysis
type AnalysisResult struct {
	Timestamp      time.Time         `json:"timestamp"`
	ServiceName    string            `json:"service_name"`
	AnomaliesFound int               `json:"anomalies_found"`
	Anomalies      []*Anomaly        `json:"anomalies"`
	BaselineUsed   *Baseline         `json:"baseline_used,omitempty"`
	GrokAnalysis   *GrokResponse     `json:"grok_analysis,omitempty"`
	ProcessingTime time.Duration     `json:"processing_time"`
	Tags           map[string]string `json:"tags"`
}

// Alert represents an anomaly alert
type Alert struct {
	ID          string            `json:"id" bson:"_id,omitempty"`
	Timestamp   time.Time         `json:"timestamp" bson:"timestamp"`
	AnomalyID   string            `json:"anomaly_id" bson:"anomaly_id"`
	Severity    string            `json:"severity" bson:"severity"`
	Title       string            `json:"title" bson:"title"`
	Message     string            `json:"message" bson:"message"`
	ServiceName string            `json:"service_name" bson:"service_name"`
	Endpoint    string            `json:"endpoint" bson:"endpoint"`
	Sent        bool              `json:"sent" bson:"sent"`
	SentAt      *time.Time        `json:"sent_at,omitempty" bson:"sent_at,omitempty"`
	Channels    []string          `json:"channels" bson:"channels"` // webhook, email, slack, etc.
	Tags        map[string]string `json:"tags" bson:"tags"`
}
