package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/config"
	database "github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
	"github.com/Avito-Team-Not-Found/menyayka/internal/core/router"
	chainHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/chain"
	exchangeOfferHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/exchange_offer"
	itemHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/item"
	userHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/user"
	"github.com/Avito-Team-Not-Found/menyayka/internal/infrastructure/chainnotification"
	"github.com/Avito-Team-Not-Found/menyayka/internal/infrastructure/embedding"
	"github.com/Avito-Team-Not-Found/menyayka/internal/infrastructure/mailer"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
	chainRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/chain"
	clusterRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/cluster"
	exchangeOfferRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/exchange_offer"
	itemRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/item"
	searchRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/search"
	userRepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/user"
	chainservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/chain"
	clusterservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/cluster"
	exchangeOfferService "github.com/Avito-Team-Not-Found/menyayka/internal/service/exchange_offer"
	itemService "github.com/Avito-Team-Not-Found/menyayka/internal/service/item"
	"github.com/Avito-Team-Not-Found/menyayka/internal/service/matching"
	userService "github.com/Avito-Team-Not-Found/menyayka/internal/service/user"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/codestore"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/storage"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/token"
)

// buildServer wires dependencies and builds the HTTP server. cleanup closes the DB pool.
func buildServer(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (*http.Server, func(), error) {
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}
	logger.Info("connected to PostgreSQL, pgvector OK")

	// Join workers before closing the pool.
	var workers sync.WaitGroup
	cleanup := func() {
		waitTimeout(&workers, workerShutdownTimeout)
		pool.Close()
	}

	pingHandler := router.NewPingHandler()

	tokenService := token.NewService(cfg.JWTSecret, cfg.JWTTokenTTL)
	userRepository := userRepo.NewRepository(pool)
	codeStore := codestore.New()
	mailerSvc, err := mailer.NewService(mailer.Config{
		Provider: mailer.Provider(cfg.MailerProvider),
		SMTP: mailer.SMTPConfig{
			Host:       cfg.SMTPHost,
			Port:       cfg.SMTPPort,
			Username:   cfg.SMTPUsername,
			Password:   cfg.SMTPPassword,
			From:       cfg.SMTPFrom,
			Encryption: cfg.SMTPEncryption,
		},
		Resend: mailer.ResendConfig{
			APIKey: cfg.ResendAPIKey,
			From:   cfg.MailFrom,
		},
	})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("mailer init: %w", err)
	}
	userSvc := userService.NewService(userRepository, tokenService, codeStore, mailerSvc, cfg.RecoveryCodeTTL)
	userH := userHandler.NewHandler(userSvc)

	exchangeOfferRepository := exchangeOfferRepo.NewRepository(pool)
	clusterRepository := clusterRepo.NewRepository(pool)
	candidateSearch := searchRepo.New(pool)
	clusterSvc := clusterservice.NewService(
		clusterRepository,
		candidateSearch,
		cfg.ClusterTopK,
		cfg.ClusterThreshold,
		cfg.ClusterDirectionMargin,
	)
	cycleFinder := matching.NewCycleFinder(
		candidateSearch,
		cfg.CycleOutgoingK,
		cfg.CycleMaxDrafts,
		cfg.MatchingThreshold,
	).WithQualityRules(cfg.CycleMinAverageScore, cfg.CycleMaxScoreGap)
	chainRepository := chainRepo.NewRepository(pool, cfg.MatchingThreshold)

	hub := realtime.NewHub(ctx, realtime.HubConfig{
		Heartbeat:       cfg.WSHeartbeat,
		WriteTimeout:    cfg.WSWriteTimeout,
		MaxChainSubs:    cfg.WSMaxChainSubs,
		SubTTL:          cfg.WSSubTTL,
		MaxConnections:  cfg.WSMaxConnections,
		MaxConnsPerUser: cfg.WSMaxConnsPerUser,
		MsgRate:         cfg.WSMsgRate,
		MsgRateWindow:   cfg.WSMsgRateWindow,
	}, logger)
	// TxEmitter is both TransactionManager and Emitter: events buffered inside a
	// transaction flush to the Hub only after commit.
	transactionManager := realtime.NewTxEmitter(database.NewTransactionManager(pool), hub).
		WithLogger(logger).
		WithParticipantResolver(chainRepository)

	scoreRanker, err := matching.NewRuntimeRanker(cfg.RankerMode, cfg.RankerModelPath)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("ranker init: %w", err)
	}
	chainSvc := chainservice.NewService(chainRepository, transactionManager)
	chainSvc = chainSvc.WithScorer(scoreRanker)
	chainSvc = chainSvc.WithNotifier(chainnotification.New(pool, mailerSvc))
	chainSvc = chainSvc.WithRealtimeEmitter(transactionManager)

	matchingFacade := matching.NewFacade(clusterSvc, cycleFinder, chainSvc).
		WithRanker(scoreRanker).
		WithRankerContextLoader(chainSvc)
	freezer := chainservice.NewFreezeService(chainRepository, matchingFacade)
	chainSvc = chainSvc.WithFreezer(freezer)

	var embedClient embedding.Client
	switch cfg.EmbeddingProvider {
	case "tei":
		embedClient = embedding.NewTEIClient(cfg.TEIURL, cfg.EmbeddingTimeout, cfg.MaxInputLength)
	case "stub", "":
		embedClient = embedding.NewStubClient()
	default:
		cleanup()
		return nil, nil, fmt.Errorf("unknown embedding provider %q", cfg.EmbeddingProvider)
	}

	exchangeOfferSvc := exchangeOfferService.NewService(
		exchangeOfferRepository,
		embedClient,
		matchingFacade,
		transactionManager,
	).WithRealtimeEmitter(transactionManager)
	exchangeOfferH := exchangeOfferHandler.NewHandler(exchangeOfferSvc)

	publicUseSSL := cfg.S3PublicUseSSL
	imageStorage, err := storage.New(ctx, storage.Config{
		Endpoint:       cfg.S3Endpoint,
		PublicEndpoint: cfg.S3PublicEndpoint,
		AccessKey:      cfg.S3AccessKey,
		SecretKey:      cfg.S3SecretKey,
		Bucket:         cfg.S3Bucket,
		UseSSL:         cfg.S3UseSSL,
		PublicUseSSL:   &publicUseSSL,
	})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("s3 storage init: %w", err)
	}

	itemRepository := itemRepo.NewRepository(pool)
	itemSvc := itemService.NewService(itemRepository, embedClient, imageStorage).
		WithRealtimeEmitter(transactionManager)
	itemH := itemHandler.NewHandler(itemSvc)
	chainH := chainHandler.NewHandler(chainSvc)

	realtimeH := realtime.NewHandler(hub, tokenService, codeStore, chainRepository, cfg, logger)
	if len(cfg.WSAllowedOrigins) == 0 {
		logger.Warn("WS_ALLOWED_ORIGINS is empty: browser WebSocket handshakes are rejected (fail-closed) and the frontend degrades to polling")
	}

	startWorker(&workers, func() { runExpiryWorker(ctx, chainSvc, logger) })
	startWorker(&workers, func() { runCodeStoreSweeper(ctx, codeStore, logger) })

	engine := router.New(tokenService, pingHandler, userH, exchangeOfferH, itemH, chainH, realtimeH)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv, cleanup, nil
}

const (
	expiryWorkerInterval   = 45 * time.Second
	codeStoreSweepInterval = 5 * time.Minute
	// Per-tick deadlines: a stalled query aborts instead of delaying later ticks.
	expiryWorkerTickTimeout = 30 * time.Second
	codeStoreSweepTimeout   = 10 * time.Second
	workerShutdownTimeout   = 5 * time.Second
)

type chainExpirer interface {
	ExpireDue(ctx context.Context) error
}

// startWorker starts a loop joined on shutdown.
func startWorker(workers *sync.WaitGroup, run func()) {
	workers.Add(1)
	go func() {
		defer workers.Done()
		run()
	}()
}

// waitTimeout waits for wg up to timeout.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// runTick runs one tick with its own deadline and a panic guard.
func runTick(ctx context.Context, logger *logrus.Logger, name string, fn func(context.Context) error, timeout time.Duration) {
	tickCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error(name + " tick panicked")
		}
	}()
	start := time.Now()
	if err := fn(tickCtx); err != nil {
		logger.WithError(err).Error(name + " tick failed")
		return
	}
	logger.WithField("duration", time.Since(start)).Debug(name + " tick done")
}

// runExpiryWorker applies elapsed chain deadlines, single-flight (the next tick
// is scheduled only after the previous one finished).
func runExpiryWorker(ctx context.Context, expirer chainExpirer, logger *logrus.Logger) {
	timer := time.NewTimer(expiryWorkerInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runTick(ctx, logger, "expiry worker", expirer.ExpireDue, expiryWorkerTickTimeout)
			timer.Reset(expiryWorkerInterval)
		}
	}
}

// runCodeStoreSweeper removes unused one-time codes (recovery codes, ws tickets).
func runCodeStoreSweeper(ctx context.Context, store *codestore.Store, logger *logrus.Logger) {
	ticker := time.NewTicker(codeStoreSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runTick(ctx, logger, "codestore sweeper", func(context.Context) error {
				if removed := store.Sweep(time.Now()); removed > 0 {
					logger.WithField("removed", removed).Debug("codestore sweep removed expired entries")
				}
				return nil
			}, codeStoreSweepTimeout)
		}
	}
}
