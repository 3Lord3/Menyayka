package config_test

import (
	"testing"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/config"
)

func TestParseRankerMode(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: config.RankerModeML},
		{in: "ml", want: config.RankerModeML},
		{in: "formula", want: config.RankerModeFormula},
		{in: "onnx", wantErr: true},
		{in: "ML", wantErr: true},
	}
	for _, tt := range tests {
		got, err := config.ParseRankerMode(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseRankerMode(%q) error = nil", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseRankerMode(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseRankerMode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLoadRejectsUnknownRankerMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "onnx")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load() expected error for unknown RANKER_MODE")
	}
}

func TestLoadDefaultRankerModeML(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RankerMode != config.RankerModeML {
		t.Fatalf("RankerMode = %q, want ml", cfg.RankerMode)
	}
}

func TestLoadRejectsInvalidNumericEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "")
	t.Setenv("MATCHING_TOPK", "not-a-number")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load() expected error for invalid MATCHING_TOPK")
	}
}

func TestLoadRejectsInvalidDurationEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "")
	t.Setenv("MATCHING_TOPK", "")
	t.Setenv("EMBEDDING_TIMEOUT", "fast")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load() expected error for invalid EMBEDDING_TIMEOUT")
	}
}

// A non-positive heartbeat would panic time.NewTicker and a zero rate window would
// silently disable the limiter, so both must fail at startup.
func TestLoadRejectsNonPositiveRealtimeSettings(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
	}{
		{"zero heartbeat", "WS_HEARTBEAT", "0s"},
		{"negative heartbeat", "WS_HEARTBEAT", "-1s"},
		{"zero write timeout", "WS_WRITE_TIMEOUT", "0"},
		{"zero ticket ttl", "WS_TICKET_TTL", "0s"},
		{"zero sub ttl", "WS_SUB_TTL", "0s"},
		{"zero rate window", "WS_MSG_RATE_WINDOW", "0s"},
		{"negative rate", "WS_MSG_RATE", "-1"},
		{"negative per-user limit", "WS_MAX_CONNS_PER_USER", "-5"},
		{"negative chain subs", "WS_MAX_CHAIN_SUBS", "-1"},
		{"negative max connections", "WS_MAX_CONNECTIONS", "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			t.Setenv("JWT_SECRET", "secret")
			t.Setenv("RANKER_MODE", "")
			t.Setenv(tc.key, tc.val)
			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() expected error for %s=%s", tc.key, tc.val)
			}
		})
	}
}

func TestLoadRealtimeDefaultsAreUsable(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "")
	t.Setenv("WS_ALLOWED_ORIGINS", "")
	t.Setenv("WS_ALLOW_QUERY_TICKET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WSHeartbeat <= 0 || cfg.WSWriteTimeout <= 0 || cfg.WSTicketTTL <= 0 || cfg.WSSubTTL <= 0 || cfg.WSMsgRateWindow <= 0 {
		t.Fatalf("durations must be positive: %+v", cfg)
	}
	if cfg.WSAllowedOrigins != nil {
		t.Fatalf("an empty WS_ALLOWED_ORIGINS must stay nil (fail-closed), got %v", cfg.WSAllowedOrigins)
	}
	if cfg.WSAllowQueryTicket {
		t.Fatal("the ?ticket= fallback must be off by default")
	}
}

// A comma-only whitelist must not allow anything.
func TestLoadAllowedOriginsParsing(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("RANKER_MODE", "")
	t.Setenv("WS_ALLOWED_ORIGINS", " , ")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WSAllowedOrigins != nil {
		t.Fatalf("WSAllowedOrigins = %v, want nil", cfg.WSAllowedOrigins)
	}

	t.Setenv("WS_ALLOWED_ORIGINS", " http://localhost:5173 , https://menyayka.tech ")
	cfg, err = config.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:5173", "https://menyayka.tech"}
	if len(cfg.WSAllowedOrigins) != len(want) {
		t.Fatalf("WSAllowedOrigins = %v, want %v", cfg.WSAllowedOrigins, want)
	}
	for i := range want {
		if cfg.WSAllowedOrigins[i] != want[i] {
			t.Fatalf("WSAllowedOrigins = %v, want %v", cfg.WSAllowedOrigins, want)
		}
	}
}
