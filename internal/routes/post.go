package routes

import (
	"publimd/internal/features/post"

	"github.com/gin-gonic/gin"
)

func PostRoutes(rg *gin.RouterGroup) {
	h := post.NewPostHandler(ucPost)

	// rg.GET("/post/train-data", h.GetTrainData)
	rg.GET("/post/public", h.GetAllPublic)
	rg.GET("/post/slug/:slug", h.GetBySlugPublic)

	protected := rg.Group("/post")
	protected.Use(authMiddleware())
	{
		protected.GET("/user", h.GetAll)
		protected.GET("/user/recent", h.GetAllRecent)
		protected.GET("/slug-private/:slug", h.GetBySlugPrivate)
		protected.GET("/states", h.GetAllStates)
		protected.POST("", h.Create)
		protected.PUT("/:id", h.Update)
		protected.PATCH("/:id/state", h.UpdateStatus)
		protected.PATCH("/vectorize/:slug", h.UpdateEmbedding)
	}
}
