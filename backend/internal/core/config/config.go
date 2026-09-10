package config

import (
	"fmt"
	"os"
	"strconv"
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
	// MinIO* config for item-photo object storage (see internal/infrastructure/storage).
	// MinIOEndpoint is the S3-client address inside the Docker network (e.g. "minio:9000").
	// MinIOPublicEndpoint is the address that lands in image_url and must be reachable
	// from the host browser (e.g. "localhost:9000", since MinIO's port is published).
	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseSSL         bool
	// MinIOPublicUseSSL is the scheme of public image_urls (via Caddy HTTPS).
	// Defaults to MinIOUseSSL when the env var is unset.
	MinIOPublicUseSSL bool

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

		MinIOEndpoint:       envOrDefault("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: envOrDefault("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:      envOrDefault("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:      envOrDefault("MINIO_SECRET_KEY", ""),
		MinIOBucket:         envOrDefault("MINIO_BUCKET", "items"),
		MinIOUseSSL:         envOrDefault("MINIO_USE_SSL", "false") == "true",
		MinIOPublicUseSSL:   envOrDefault("MINIO_PUBLIC_USE_SSL", envOrDefault("MINIO_USE_SSL", "false")) == "true",
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
