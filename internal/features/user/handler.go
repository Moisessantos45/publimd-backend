package user

import (
	"log"
	"publimd/internal/shared/models"
	"publimd/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	uc UserService
}

func NewUserHandler(uc UserService) *UserHandler {
	return &UserHandler{uc: uc}
}

func (h *UserHandler) GetDashboardMetrics(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	metrics, err := h.uc.GetDashboardMetrics(c.Request.Context(), authID)
	if err != nil {
		c.JSON(404, gin.H{"message": "Metrics not found"})
		return
	}

	c.JSON(200, gin.H{"data": metrics, "message": "Metrics retrieved successfully"})
}

func (h *UserHandler) GetByUserID(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	user, err := h.uc.GetByAuthID(c.Request.Context(), authID)
	if err != nil {
		c.JSON(404, gin.H{"message": "User not found"})
		return
	}

	c.JSON(200, gin.H{"data": user, "message": "User retrieved successfully"})
}

func (h *UserHandler) Create(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		log.Printf("Error extracting authID: %v", err)
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	log.Printf("Creating user with authID: %d", authID)

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	user.AuthID = authID

	if err := h.uc.Create(c.Request.Context(), &user); err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(500, gin.H{"message": "Failed to create user"})
		return
	}

	c.JSON(201, gin.H{"message": "User created successfully", "data": user})
}

func (h *UserHandler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	if err := h.uc.Update(c.Request.Context(), authID, &user); err != nil {
		c.JSON(500, gin.H{"message": "Failed to update user"})
		return
	}

	c.JSON(200, gin.H{"message": "User updated successfully"})
}
