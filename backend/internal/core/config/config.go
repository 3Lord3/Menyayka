package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// jwtTokenTTL is the session JWT lifetime (PROJECT.md §3.2: token lives 24h).
const jwtTokenTTL = 24 * time.Hour

// recoveryCodeTTL is the recovery-code lifetime (PROJECT.md §4.1: 10 minutes).
const recoveryCodeTTL = 10 * time.Minute

// Config holds the application configuration.
type Config struct {
	DatabaseURL string
	ServerPort  string
	LogLevel    string
	JWTSecret   string
	JWTTokenTTL time.Duration

	// SMTP* config for sending password-recovery emails. Deliberately not required:
	// the rest of the backend starts without them and only email sending fails
	// (see mailer.ErrNotConfigured).
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	// SMTPEncryption is "plain" | "starttls" | "tls", see internal/infrastructure/mailer.
	SMTPEncryption  string
	RecoveryCodeTTL time.Duration

	// Embedding* config for generating want vectors (TEI).
	// Provider: "tei" | "stub" | "" (empty = stub for local dev without TEI).
	EmbeddingProvider string
	TEIURL            string
	VectorDim         int
	EmbeddingTimeout  time.Duration
	MaxInputLength    int
	// S3* config for item-photo object storage (see pkg/storage).
	// S3Endpoint is the S3-client address inside the Docker network (e.g. "seaweedfs:9000").
	// S3PublicEndpoint is the address that lands in image_url and must be reachable
	// from the host browser (e.g. "localhost:9000", since the S3 port is published).
	S3Endpoint       string
	S3PublicEndpoint string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	S3UseSSL         bool
	// S3PublicUseSSL is the scheme of public image_urls (via Caddy HTTPS).
	// Defaults to S3UseSSL when the env var is unset.
	S3PublicUseSSL bool

	// Matching* config for vector candidate search (SCRUM-24). pgvector only yields
	// top/threshold semantic candidates; cycle search and clusters are built on top
	// in Go. Parameters come from the environment.
	MatchingTopK           int     // LIMIT for top-K graph edges (default 20)
	MatchingThreshold      float64 // cosine threshold for want→item edges (default 0.5)
	ClusterTopK            int     // candidate LIMIT during clustering (default 50)
	ClusterThreshold       float64 // same-direction offer+want threshold (default 0.9)
	ClusterDirectionMargin float64 // margin of forward over reverse similarity without category
	VectorMetric           string  // "cosine" (fixed, matches the index)
	CycleOutgoingK         int     // max outgoing edges per vertex
	CycleMaxDrafts         int     // max chain drafts to return
	CycleMinAverageScore   float64 // min average quality of a cycle's edges
	CycleMaxScoreGap       float64 // allowed gap between best and worst edge

	// Ranker* chain scoring: LightGBM (default) or formula (explicit baseline).
	RankerMode      string
	RankerModelPath string

	// Realtime WebSocket.
	WSHeartbeat       time.Duration // ping interval
	WSWriteTimeout    time.Duration // write deadline
	WSTicketTTL       time.Duration // ticket lifetime
	WSAllowedOrigins  []string      // fail-closed: empty rejects every handshake
	WSMaxConnections  int           // 0 = unlimited
	WSMaxConnsPerUser int
	WSMaxChainSubs    int           // chain subs per connection
	WSSubTTL          time.Duration // idle subscription TTL
	WSMsgRate         int           // messages per WSMsgRateWindow
	WSMsgRateWindow   time.Duration
	// WSAllowQueryTicket enables the ?ticket= query fallback (off: the ticket
	// travels as a subprotocol and never reaches access logs).
	WSAllowQueryTicket bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	rankerMode, err := ParseRankerMode(os.Getenv("RANKER_MODE"))
	if err != nil {
		return nil, err
	}

	vectorDim, err := envIntOrDefault("VECTOR_DIM", 384)
	if err != nil {
		return nil, err
	}
	embeddingTimeout, err := envDurationOrDefault("EMBEDDING_TIMEOUT", 2*time.Second)
	if err != nil {
		return nil, err
	}
	maxInputLength, err := envIntOrDefault("MAX_INPUT_LENGTH", 1500)
	if err != nil {
		return nil, err
	}
	matchingTopK, err := envIntOrDefault("MATCHING_TOPK", 20)
	if err != nil {
		return nil, err
	}
	matchingThreshold, err := envFloatOrDefault("MATCHING_THRESHOLD", 0.5)
	if err != nil {
		return nil, err
	}
	clusterTopK, err := envIntOrDefault("CLUSTER_TOPK", 50)
	if err != nil {
		return nil, err
	}
	clusterThreshold, err := envFloatOrDefault("CLUSTER_SIMILARITY_THRESHOLD", 0.9)
	if err != nil {
		return nil, err
	}
	clusterDirectionMargin, err := envFloatOrDefault("CLUSTER_DIRECTION_MARGIN", 0.05)
	if err != nil {
		return nil, err
	}
	cycleOutgoingK, err := envIntOrDefault("CYCLE_OUTGOING_K", 20)
	if err != nil {
		return nil, err
	}
	cycleMaxDrafts, err := envIntOrDefault("CYCLE_MAX_DRAFTS", 10)
	if err != nil {
		return nil, err
	}
	cycleMinAverageScore, err := envFloatOrDefault("CYCLE_MIN_AVERAGE_SCORE", 0.5)
	if err != nil {
		return nil, err
	}
	cycleMaxScoreGap, err := envFloatOrDefault("CYCLE_MAX_SCORE_GAP", 1)
	if err != nil {
		return nil, err
	}
	wsHeartbeat, err := envDurationOrDefault("WS_HEARTBEAT", 30*time.Second)
	if err != nil {
		return nil, err
	}
	wsWriteTimeout, err := envDurationOrDefault("WS_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	wsTicketTTL, err := envDurationOrDefault("WS_TICKET_TTL", 30*time.Second)
	if err != nil {
		return nil, err
	}
	wsMaxConnections, err := envIntOrDefault("WS_MAX_CONNECTIONS", 0)
	if err != nil {
		return nil, err
	}
	wsMaxConnsPerUser, err := envIntOrDefault("WS_MAX_CONNS_PER_USER", 5)
	if err != nil {
		return nil, err
	}
	wsMaxChainSubs, err := envIntOrDefault("WS_MAX_CHAIN_SUBS", 50)
	if err != nil {
		return nil, err
	}
	wsSubTTL, err := envDurationOrDefault("WS_SUB_TTL", 30*time.Minute)
	if err != nil {
		return nil, err
	}
	// Survives the subscription replay burst of a reconnecting client.
	wsMsgRate, err := envIntOrDefault("WS_MSG_RATE", 100)
	if err != nil {
		return nil, err
	}
	wsMsgRateWindow, err := envDurationOrDefault("WS_MSG_RATE_WINDOW", 10*time.Second)
	if err != nil {
		return nil, err
	}
	if err := validateRealtimeSettings(realtimeEnvSettings{
		heartbeat:       wsHeartbeat,
		writeTimeout:    wsWriteTimeout,
		ticketTTL:       wsTicketTTL,
		subTTL:          wsSubTTL,
		msgRate:         wsMsgRate,
		msgRateWindow:   wsMsgRateWindow,
		maxConnections:  wsMaxConnections,
		maxConnsPerUser: wsMaxConnsPerUser,
		maxChainSubs:    wsMaxChainSubs,
	}); err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL: dbURL,
		ServerPort:  envOrDefault("SERVER_PORT", "8080"),
		LogLevel:    envOrDefault("LOG_LEVEL", "info"),
		JWTSecret:   jwtSecret,
		JWTTokenTTL: jwtTokenTTL,

		SMTPHost:        envOrDefault("SMTP_HOST", ""),
		SMTPPort:        envOrDefault("SMTP_PORT", "587"),
		SMTPUsername:    envOrDefault("SMTP_USERNAME", ""),
		SMTPPassword:    envOrDefault("SMTP_PASSWORD", ""),
		SMTPFrom:        envOrDefault("SMTP_FROM", ""),
		SMTPEncryption:  envOrDefault("SMTP_ENCRYPTION", "starttls"),
		RecoveryCodeTTL: recoveryCodeTTL,

		EmbeddingProvider: envOrDefault("EMBEDDING_PROVIDER", "stub"),
		TEIURL:            envOrDefault("TEI_URL", "http://tei:80"),
		VectorDim:         vectorDim,
		EmbeddingTimeout:  embeddingTimeout,
		MaxInputLength:    maxInputLength,

		MatchingTopK:           matchingTopK,
		MatchingThreshold:      matchingThreshold,
		ClusterTopK:            clusterTopK,
		ClusterThreshold:       clusterThreshold,
		ClusterDirectionMargin: clusterDirectionMargin,
		VectorMetric:           envOrDefault("VECTOR_METRIC", "cosine"),
		CycleOutgoingK:         cycleOutgoingK,
		CycleMaxDrafts:         cycleMaxDrafts,
		CycleMinAverageScore:   cycleMinAverageScore,
		CycleMaxScoreGap:       cycleMaxScoreGap,

		RankerMode:      rankerMode,
		RankerModelPath: envOrDefault("RANKER_MODEL_PATH", "pkg/utils/ranker/models/ranker_v1.txt"),

		WSHeartbeat:        wsHeartbeat,
		WSWriteTimeout:     wsWriteTimeout,
		WSTicketTTL:        wsTicketTTL,
		WSAllowedOrigins:   parseCSV(os.Getenv("WS_ALLOWED_ORIGINS")),
		WSMaxConnections:   wsMaxConnections,
		WSMaxConnsPerUser:  wsMaxConnsPerUser,
		WSMaxChainSubs:     wsMaxChainSubs,
		WSSubTTL:           wsSubTTL,
		WSMsgRate:          wsMsgRate,
		WSMsgRateWindow:    wsMsgRateWindow,
		WSAllowQueryTicket: envOrDefault("WS_ALLOW_QUERY_TICKET", "false") == "true",

		S3Endpoint:       envOrDefault("S3_ENDPOINT", "localhost:9000"),
		S3PublicEndpoint: envOrDefault("S3_PUBLIC_ENDPOINT", "localhost:9000"),
		S3AccessKey:      envOrDefault("S3_ACCESS_KEY", ""),
		S3SecretKey:      envOrDefault("S3_SECRET_KEY", ""),
		S3Bucket:         envOrDefault("S3_BUCKET", "items"),
		S3UseSSL:         envOrDefault("S3_USE_SSL", "false") == "true",
		S3PublicUseSSL:   envOrDefault("S3_PUBLIC_USE_SSL", envOrDefault("S3_USE_SSL", "false")) == "true",
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseCSV splits a comma-separated value, trimming whitespace and dropping
// empties; an empty input yields nil.
func parseCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// realtimeEnvSettings holds parsed WS_* values for validation.
type realtimeEnvSettings struct {
	heartbeat       time.Duration
	writeTimeout    time.Duration
	ticketTTL       time.Duration
	subTTL          time.Duration
	msgRate         int
	msgRateWindow   time.Duration
	maxConnections  int
	maxConnsPerUser int
	maxChainSubs    int
}

// validateRealtimeSettings fails fast on a misconfigured WS_* value: a non-positive
// duration would panic a ticker or silently disable a limiter.
func validateRealtimeSettings(s realtimeEnvSettings) error {
	for _, d := range []struct {
		name  string
		value time.Duration
	}{
		{"WS_HEARTBEAT", s.heartbeat},
		{"WS_WRITE_TIMEOUT", s.writeTimeout},
		{"WS_TICKET_TTL", s.ticketTTL},
		{"WS_SUB_TTL", s.subTTL},
		{"WS_MSG_RATE_WINDOW", s.msgRateWindow},
	} {
		if d.value <= 0 {
			return fmt.Errorf("%s must be a positive duration, got %s", d.name, d.value)
		}
	}
	for _, l := range []struct {
		name  string
		value int
	}{
		{"WS_MSG_RATE", s.msgRate},
		{"WS_MAX_CONNECTIONS", s.maxConnections},
		{"WS_MAX_CONNS_PER_USER", s.maxConnsPerUser},
		{"WS_MAX_CHAIN_SUBS", s.maxChainSubs},
	} {
		if l.value < 0 {
			return fmt.Errorf("%s must not be negative, got %d", l.name, l.value)
		}
	}
	return nil
}

// envIntOrDefault falls back only if the variable is unset; an invalid value is a
// startup error.
func envIntOrDefault(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid int: %w", key, v, err)
	}
	return n, nil
}

// envDurationOrDefault is like envIntOrDefault for time.Duration.
func envDurationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid duration: %w", key, v, err)
	}
	return d, nil
}

// envFloatOrDefault is like envIntOrDefault for float64.
func envFloatOrDefault(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid float: %w", key, v, err)
	}
	return f, nil
}

const (
	RankerModeFormula = "formula"
	RankerModeML      = "ml"
)

// ParseRankerMode accepts formula|ml; empty defaults to ml. Unknown values are a
// fail-fast startup error.
func ParseRankerMode(v string) (string, error) {
	switch v {
	case "", RankerModeML:
		return RankerModeML, nil
	case RankerModeFormula:
		return RankerModeFormula, nil
	default:
		return "", fmt.Errorf("RANKER_MODE must be formula or ml, got %q", v)
	}
}
