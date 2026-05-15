package collaborator

import (
	"publimd/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type CollaboratorHandler struct {
	uc CollaboratorService
}

func NewCollaboratorHandler(uc CollaboratorService) *CollaboratorHandler {
	return &CollaboratorHandler{uc: uc}
}

func (h *CollaboratorHandler) GetAllPermissions(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	permissions, err := h.uc.GetAllPermissions(c.Request.Context(), userID, slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error fetching permissions: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": permissions, "message": "Permissions fetched successfully"})
}

func (h *CollaboratorHandler) GetCollaborators(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	collaborators, err := h.uc.GetAll(c.Request.Context(), userID, slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error fetching collaborators: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": collaborators, "message": "Collaborators fetched successfully"})
}

func (h *CollaboratorHandler) GetCollaboratorsBasicInfo(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	collaborators, err := h.uc.GetAllUserInfoBasic(c.Request.Context(), userID, slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error fetching collaborators basic info: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": collaborators, "message": "Collaborators basic info fetched successfully"})
}

func (h *CollaboratorHandler) GetUserByEmail(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	email := c.DefaultQuery("email", "")
	if email == "" {
		c.JSON(400, gin.H{"message": "email is required"})
		return
	}

	userInfo, err := h.uc.GetUserByEmail(c.Request.Context(), email, userID, slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error fetching user by email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": userInfo, "message": "User found successfully"})
}

func (h *CollaboratorHandler) AddCollaborator(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	var req struct {
		TargetUserID uint64 `json:"target_user_id"`
		PermissionID uint64 `json:"permission_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	err = h.uc.Create(c.Request.Context(), userID, slug, req.TargetUserID, req.PermissionID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error adding collaborator: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Collaborator added successfully"})
}

func (h *CollaboratorHandler) UpdateCollaborator(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	var req struct {
		TargetUserID uint64 `json:"target_user_id"`
		PermissionID uint64 `json:"permission_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	err = h.uc.UpdatePermission(c.Request.Context(), userID, slug, req.TargetUserID, req.PermissionID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error updating collaborator: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Collaborator updated successfully"})
}

func (h *CollaboratorHandler) RemoveCollaborator(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	targetUserIDParam, err := utils.ValidateParamsId(c, "targetUserID")
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid target user ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	err = h.uc.Delete(c.Request.Context(), userID, slug, targetUserIDParam)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error removing collaborator: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Collaborator removed successfully"})
}

func (h *CollaboratorHandler) ConfirmInvitation(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(400, gin.H{"message": "token is required"})
		return
	}

	if err := h.uc.ConfirmInvitation(c.Request.Context(), token); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Invitation confirmed successfully"})
}

func (h *CollaboratorHandler) ResendInvitation(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "slug is required"})
		return
	}

	var req struct {
		TargetUserID uint64 `json:"target_user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	if err := h.uc.ResendInvitation(c.Request.Context(), userID, slug, req.TargetUserID); err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Invitation resent successfully"})
}
