package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/api/handlers"
	"github.com/yoockh/yoospeak/internal/api/middleware"
)

type Deps struct {
	Session      *handlers.SessionHandler
	Profile      *handlers.ProfileHandler
	Conversation *handlers.ConversationHandler
	WS           *handlers.WSHandler
}

func RegisterRoutes(r *gin.Engine, d Deps) {
	// Health-ish
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Protected routes (JWT)
	auth := r.Group("/")
	auth.Use(middleware.JWTAuth())

	auth.POST("/session/start", d.Session.Start)
	auth.GET("/session/:session_id", d.Session.Get)
	auth.POST("/session/:session_id/end", d.Session.End)

	auth.GET("/profile/me", d.Profile.Me)
	auth.PUT("/profile/update", d.Profile.Update)

	auth.GET("/conversation/:session_id", d.Conversation.ListBySession)

	// WebSocket
	auth.GET("/ws/session/:session_id", d.WS.SessionWS)
}
