// Package router builds the application HTTP router.
//
// Feature handlers are passed explicitly to New and routes are registered in its
// body; this is the only place that needs changing to expose new endpoints.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Avito-Team-Not-Found/menyayka/docs"
	chainHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/chain"
	exchangeOfferHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/exchange_offer"
	itemHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/item"
	userHandler "github.com/Avito-Team-Not-Found/menyayka/internal/handler/user"
	"github.com/Avito-Team-Not-Found/menyayka/internal/middleware"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
)

func New(
	tokenParser middleware.TokenParser,
	pingH *PingHandler,
	userH *userHandler.Handler,
	exchangeOfferH *exchangeOfferHandler.Handler,
	itemH *itemHandler.Handler,
	chainH *chainHandler.Handler,
	realtimeH *realtime.Handler,
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// Swagger UI: http://localhost:8080/swagger/index.html
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// info
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	{
		// ping is a scaffold smoke-test endpoint; remove once a real feature exists.
		api.GET("/ping", pingH.Ping)

		// auth: register/login are public, the rest require a Bearer token.
		auth := api.Group("/auth")
		auth.POST("/register", userH.Register)
		auth.POST("/login", userH.Login)

		authProtected := auth.Group("")
		authProtected.Use(middleware.Auth(tokenParser))
		{
			authProtected.POST("/logout", userH.Logout)
			authProtected.GET("/me", userH.Me)
			authProtected.POST("/change-password", userH.ChangePassword)
		}

		exchangeOffers := api.Group("/exchange-offers")
		exchangeOffers.Use(middleware.Auth(tokenParser))
		{
			exchangeOffers.POST("", exchangeOfferH.Create)
			exchangeOffers.GET("", exchangeOfferH.List)
			exchangeOffers.GET("/:id/exchange-options", chainH.ExchangeOptions)
			exchangeOffers.GET("/:id", exchangeOfferH.Get)
			exchangeOffers.PUT("/:id", exchangeOfferH.Update)
			exchangeOffers.DELETE("/:id", exchangeOfferH.Delete)
		}

		items := api.Group("/items")
		items.Use(middleware.Auth(tokenParser))
		{
			items.POST("", itemH.Create)
			items.GET("", itemH.List)
			items.GET("/:id", itemH.Get)
			items.PATCH("/:id", itemH.Update)
			items.DELETE("/:id", itemH.Archive)
			items.POST("/:id/image", itemH.UploadImage)
		}

		chains := api.Group("/chains")
		chains.Use(middleware.Auth(tokenParser))
		{
			chains.GET("", chainH.List)
			chains.GET("/:id", chainH.Get)
			chains.PUT("/:id/votes", chainH.Vote)
			chains.DELETE("/:id/votes", chainH.WithdrawVote)
			chains.POST("/:id/confirm", chainH.Confirm)
			chains.POST("/:id/unconfirm", chainH.Unconfirm)
			chains.POST("/:id/think", chainH.Think)
			chains.POST("/:id/decline", chainH.Decline)
			chains.GET("/:id/replacements", chainH.ListReplacements)
			chains.PUT("/:id/replacement", chainH.SelectReplacement)
			chains.POST("/:id/receipt", chainH.ConfirmReceipt)
		}

		// Handoff callback (MVP without external auth; protect before production).
		integrations := api.Group("/integrations")
		integrations.POST("/avito/handoffs", chainH.Handoff)

		// /ws authenticates by ticket; /ws/ticket requires a session.
		api.GET("/ws", realtimeH.Connect)
		api.GET("/ws/ticket", middleware.Auth(tokenParser), realtimeH.Ticket)

		// recovery without JWT: the user is not logged in yet
		recovery := api.Group("/account/password-recovery")
		recovery.POST("/send-code/", userH.SendRecoveryCode)
		recovery.POST("/verify-code/", userH.VerifyRecoveryCode)
		recovery.POST("/reset-password/", userH.ResetPassword)
	}

	return engine
}
