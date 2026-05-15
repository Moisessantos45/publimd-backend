package routes

import (
	"publimd/internal/features/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(rg *gin.RouterGroup) {
	s := authUc
	h := auth.NewAuthHandler(s)

	rg.POST("/login", h.Login)
	rg.POST("/forward-email-verification", h.ForwardEmailVerification)
	rg.POST("/forgot-password", h.SendPasswordReset)
	rg.POST("/register", h.Register)
	rg.POST("/logout", h.Logout)
	rg.POST("/refresh-token", h.RefreshToken)

	preAuth := rg.Group("/")
	preAuth.Use(preAuthMiddleware())
	{
		preAuth.POST("/2fa/verify", h.Verify2FALogin)
	}

	protected := rg.Group("/")
	protected.Use(authMiddleware())
	{
		protected.GET("/confirm-account", h.ConfirmAccount)
		protected.GET("/session", h.GetSession)
		protected.POST("/verify-email", h.VerifyEmail)
		protected.PATCH("/reset-password", h.ResetPassword)
		protected.PATCH("/change-password", h.UpdatePassword)

		protected.GET("/generate-two-factor", h.TestingCrateTWOFA)
	}
}
