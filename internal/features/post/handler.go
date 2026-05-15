package post

import (
	"net/http"
	"publimd/internal/shared/models"
	"publimd/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	uc PostService
}

func NewPostHandler(uc PostService) *PostHandler {
	return &PostHandler{uc: uc}
}

// func (h *PostHandler) GetTrainData(c *gin.Context) {
// 	data, err := h.uc.GetTrainData(c.Request.Context())
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"message": "Failed to retrieve training data",
// 		})
// 		return
// 	}

// 	payload := gin.H{
// 		"data":    data,
// 		"message": "Training data retrieved successfully",
// 	}

// 	jsonBytes, err := json.MarshalIndent(payload, "", "  ")
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"message": "Failed to generate JSON file",
// 		})
// 		return
// 	}

// 	c.Header("Content-Disposition", `attachment; filename="train-data.json"`)
// 	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonBytes)
// }

func (h *PostHandler) GetAllStates(c *gin.Context) {
	states, err := h.uc.GetAllStates(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve post states"})
		return
	}

	c.JSON(200, gin.H{"data": states, "message": "Post states retrieved successfully"})
}

func (h *PostHandler) GetAll(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	page, pageSize, _, err := utils.ValidateQueryPagination(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid pagination parameters: " + err.Error()})
		return
	}

	posts, err := h.uc.GetAll(c.Request.Context(), authID, page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve posts"})
		return
	}

	c.JSON(200, gin.H{"data": posts, "message": "Posts retrieved successfully"})
}

func (h *PostHandler) GetAllPublic(c *gin.Context) {
	page, pageSize, query, err := utils.ValidateQueryPagination(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid pagination parameters: " + err.Error()})
		return
	}

	posts, err := h.uc.GetAllPublic(c.Request.Context(), page, pageSize, query)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve public posts"})
		return
	}

	c.JSON(200, gin.H{"data": posts, "message": "Public posts retrieved successfully"})
}

func (h *PostHandler) GetAllRecent(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	posts, err := h.uc.GetAllRecent(c.Request.Context(), authID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve recent posts"})
		return
	}

	c.JSON(200, gin.H{"data": posts, "message": "Recent posts retrieved successfully"})
}

func (h *PostHandler) Create(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	var post models.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	if err := h.uc.Create(c.Request.Context(), authID, &post); err != nil {
		c.JSON(500, gin.H{"message": "Failed to create post"})
		return
	}

	c.JSON(201, gin.H{"message": "Post created successfully", "data": post})
}

func (h *PostHandler) GetBySlugPrivate(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	post, err := h.uc.GetBySlugPrivate(c.Request.Context(), slug, authID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve post"})
		return
	}

	c.JSON(200, gin.H{"data": post, "message": "Post retrieved successfully"})
}

func (h *PostHandler) GetBySlugPublic(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	post, err := h.uc.GetBySlugPublic(c.Request.Context(), slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to retrieve post"})
		return
	}

	c.JSON(200, gin.H{"data": post, "message": "Post retrieved successfully"})
}

func (h *PostHandler) Update(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	id, err := utils.ValidateParamsId(c, "")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	var post models.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	if err := h.uc.Update(c.Request.Context(), id, userID, &post); err != nil {
		c.JSON(500, gin.H{"message": "Failed to update post"})
		return
	}

	c.JSON(200, gin.H{"message": "Post updated successfully"})
}

func (h *PostHandler) UpdateStatus(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	id, err := utils.ValidateParamsId(c, "")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	var req struct {
		Status uint64 `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	if err := h.uc.UpdateState(c.Request.Context(), id, userID, req.Status); err != nil {
		c.JSON(500, gin.H{"message": "Failed to update post status"})
		return
	}

	c.JSON(200, gin.H{"message": "Post status updated successfully"})
}

func (h *PostHandler) UpdateEmbedding(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	if err := h.uc.UpdateEmbedding(c.Request.Context(), userID, slug); err != nil {
		c.JSON(500, gin.H{"message": "Failed to update post embedding"})
		return
	}

	c.JSON(200, gin.H{"message": "Post embedding updated successfully"})
}
