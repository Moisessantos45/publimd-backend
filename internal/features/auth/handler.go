package auth

import (
	"errors"
	"log"
	"net/http"
	"publimd/internal/shared/errorsx"
	"publimd/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	s AuthService
}

func NewAuthHandler(s AuthService) *AuthHandler {
	return &AuthHandler{
		s: s,
	}
}

func (h *AuthHandler) Verify2FALogin(c *gin.Context) {
	ctx := c.Request.Context()

	sessionID, err := c.Cookie("pre_auth_session")
	if err != nil || sessionID == "" {
		c.JSON(401, gin.H{"message": "Sesión de pre-autenticación no encontrada o expirada"})
		return
	}

	type Verify2FARequest struct {
		Code string `json:"code" binding:"required"`
	}

	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	user, refreshToken, err := h.s.Verify2FALogin(ctx, sessionID, req.Code)
	if err != nil {
		c.JSON(401, gin.H{"message": err.Error()})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("pre_auth_session", "", -1, "/", "", true, true)
	c.SetCookie("refresh_token", refreshToken, 8*60*60, "/", "", true, true)

	c.JSON(200, gin.H{"data": user, "message": "Login exitoso"})
}

func (h *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	type RegisterRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	err := h.s.Register(ctx, req.Email, req.Password)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error registering user: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{"message": "Usuario registrado exitosamente por favor verifica tu correo para activar tu cuenta"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	type LoginRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	user, tokenOrSession, err := h.s.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, errorsx.ErrCodeEmailNotVerified) {
			c.JSON(403, gin.H{
				"message": "Correo no verificado",
				"code":    "EMAIL_NOT_VERIFIED",
			})
			return
		}

		if errors.Is(err, errorsx.ErrPending2FA) {
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie("pre_auth_session", tokenOrSession, 5*60, "/", "", true, true)
			c.JSON(202, gin.H{
				"message": "Se requiere verificación de dos factores",
				"code":    "PENDING_2FA",
			})
			return
		}

		c.JSON(401, gin.H{"message": "Invalid email or password"})
		return
	}

	log.Printf("User %s logged in successfully", user.Token)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", tokenOrSession, 8*60*60, "/", "", true, true)

	c.JSON(200, gin.H{"data": user, "message": "Login successful"})
}

func (h *AuthHandler) GetSession(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	log.Printf("Getting session for user ID: %d", userID)

	session, err := h.s.GetSession(userID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error getting session: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": session, "message": "Session retrieved successfully"})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx := c.Request.Context()

	token, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(400, gin.H{"message": "No refresh token provided"})
		return
	}
	newToken, err := h.s.RefreshToken(ctx, token)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error refreshing token: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Token refreshed successfully", "data": map[string]string{"token": newToken}})
}

func (h *AuthHandler) ForwardEmailVerification(c *gin.Context) {
	ctx := c.Request.Context()
	type ForwardEmailRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req ForwardEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	err := h.s.ForwardEmailVerification(ctx, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error forwarding email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Verification email forwarded successfully"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	type VerifyEmailRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	err := h.s.VerifyEmail(req.Email)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error verifying email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Verification email sent successfully", "data": map[string]bool{"exist": true}})
}

func (h *AuthHandler) ConfirmAccount(c *gin.Context) {
	ctx := c.Request.Context()
	token, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	err = h.s.ConfirmAccount(ctx, userID, token)
	if err != nil {
		c.JSON(500, gin.H{"message": "No se pudo confirmar la cuenta: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Tu cuenta ha sido confirmada exitosamente, ya puedes iniciar sesión"})
}

func (h *AuthHandler) SendPasswordReset(c *gin.Context) {
	ctx := c.Request.Context()

	type SendPasswordResetRequest struct {
		Email string `json:"email" binding:"required,email"`
	}

	var req SendPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	err := h.s.SendPasswordReset(ctx, req.Email)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error sending password reset email: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Se ha enviado un correo para restablecer tu contraseña, por favor revisa tu bandeja de entrada"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	_, userId, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	type ResetPasswordRequest struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	log.Printf("Resetting password for user ID: %d", userId)

	err = h.s.ResetPassword(userId, req.NewPassword)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error resetting password: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Tu contraseña ha sido restablecida exitosamente, ya puedes iniciar sesión con tu nueva contraseña"})
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	type UpdatePasswordRequest struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	err = h.s.UpdatePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error updating password: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Tu contraseña ha sido actualizada exitosamente, ya puedes iniciar sesión con tu nueva contraseña"})
}

func (h *AuthHandler) EnableTWOFA(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	uri, secret, err := h.s.EnableTWOFA(userID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error enabling 2FA: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "2FA habilitado exitosamente", "data": map[string]string{"secret": secret, "uri": uri}})
}

func (h *AuthHandler) TestingCrateTWOFA(c *gin.Context) {
	uri, secret, err := utils.GenerateTOTP("moy@gmail.com", "uaslp")
	if err != nil {
		c.JSON(500, gin.H{"message": "Error generating TOTP: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "TOTP generated successfully", "data": map[string]string{"secret": secret, "uri": uri}})
}

func (h *AuthHandler) TestVerifyTOTP(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	type VerifyTOTPRequest struct {
		Code string `json:"code" binding:"required"`
	}

	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("err", err)
		c.JSON(400, gin.H{"message": "Invalid request: " + err.Error()})
		return
	}

	valid, err := h.s.VerifyTOTP(userID, req.Code)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error verifying TOTP: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "TOTP verification completed", "data": map[string]bool{"valid": valid}})
}

func (h *AuthHandler) DisableTWOFA(c *gin.Context) {
	_, userID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid token: " + err.Error()})
		return
	}

	err = h.s.DisableTWOFA(userID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Error disabling 2FA: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "2FA deshabilitado exitosamente"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	_, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(400, gin.H{"message": "No refresh token provided"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	c.JSON(200, gin.H{"message": "Sesión cerrada exitosamente"})
}
