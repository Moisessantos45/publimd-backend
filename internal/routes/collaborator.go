package routes

import (
	"publimd/internal/features/collaborator"

	"github.com/gin-gonic/gin"
)

func CollaboratorRoutes(rg *gin.RouterGroup) {
	h := collaborator.NewCollaboratorHandler(ucCols)

	rg.POST("/collaborator/confirm/:token", h.ConfirmInvitation)

	protected := rg.Group("/collaborator")
	protected.Use(authMiddleware())
	{
		protected.GET("/permissions/:slug", h.GetAllPermissions)
		protected.GET("/users/:slug", h.GetCollaborators)
		protected.GET("/user-info/:slug", h.GetCollaboratorsBasicInfo)
		protected.GET("/invite/:slug", h.GetUserByEmail)
		protected.POST("/:slug", h.AddCollaborator)
		protected.POST("/resend", h.ResendInvitation)
		protected.PUT("/:slug", h.UpdateCollaborator)
		protected.DELETE("/:slug/:targetUserID", h.RemoveCollaborator)
	}
}
