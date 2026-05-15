package middleware

import (
	"context"
	"fmt"
	"log"
	"publimd/internal/shared/utils"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware(maker *utils.PasetoMaker, rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		log.Printf("Starting AuthMiddleware for path: %s", c.FullPath())
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Printf("Invalid Authorization header: %s", authHeader)
			c.AbortWithStatusJSON(401, gin.H{"msg": "missing or invalid Authorization header"})
			return
		}

		token := strings.TrimSpace(authHeader[7:])
		if token == "" || len(token) < 20 {
			log.Printf("Invalid token length: %d", len(token))
			c.AbortWithStatusJSON(401, gin.H{"msg": "invalid token"})
			return
		}

		log.Printf("Received token: %s", token[:8]+"...")

		payload, err := maker.VerifyToken(token)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			c.AbortWithStatusJSON(401, gin.H{"msg": "invalid token"})
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("payload", payload)
		c.Set("token", token)
		c.Next()
	}

}

func PreAuthMiddleware(rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("pre_auth_session")
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación requerida"})
			return
		}

		key := fmt.Sprintf("preauth:%s", sessionID)
		vals, err := rd.HGetAll(context.Background(), key).Result()
		if err != nil || len(vals) == 0 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación expirada o inválida"})
			return
		}

		authStage, ok := vals["auth_stage"]
		if !ok || authStage != "pending_2fa" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "estado de autenticación inválido"})
			return
		}

		c.Set("pre_auth_session_id", sessionID)
		c.Next()
	}
}
