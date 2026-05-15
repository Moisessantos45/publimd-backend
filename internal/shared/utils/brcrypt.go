package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword genera un hash bcrypt de una contraseña.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error al generar el hash de la contraseña: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compara una contraseña en texto plano con su hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
