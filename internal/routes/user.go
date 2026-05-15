package routes

import (
	"publimd/internal/features/user"

	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup) {
	h := user.NewUserHandler(userUc)

	protected := rg.Group("/user")
	protected.Use(authMiddleware())
	{
		protected.GET("/dashboard-metrics", h.GetDashboardMetrics)
		protected.POST("", h.Create)
		protected.GET("", h.GetByUserID)
		protected.PUT("", h.Update)
	}
}
