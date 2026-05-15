package utils

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var PhoneRegex = regexp.MustCompile(`^\+?[0-9\s\-\(\)]{7,20}$`)

func ValidateParamsId(c *gin.Context, params string) (uint64, error) {
	newParams := params
	if len(strings.TrimSpace(params)) == 0 {
		newParams = "id"
	}

	idStr := c.Param(newParams)
	if len(strings.TrimSpace(idStr)) == 0 {
		return 0, errors.New("ID de producto no proporcionado")
	}

	id, newErr := strconv.ParseUint(idStr, 10, 64)
	if newErr != nil {
		return 0, errors.New("ID de producto inválido: " + newErr.Error())
	}

	return id, nil
}

func ValidateParamsQuery(c *gin.Context, paramName string) (uint64, error) {
	paramStr := c.DefaultQuery(paramName, "0")

	if len(strings.TrimSpace(paramStr)) == 0 {
		return 0, fmt.Errorf("Parámetro de consulta '%s' no proporcionado", paramName)
	}

	param, err := strconv.ParseUint(paramStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Parámetro de consulta '%s' inválido: %v", paramName, err)
	}

	return param, nil
}

func ExtractedParamsJwt(c *gin.Context) (string, uint64, error) {
	userIDIface, exists := c.Get("userID")
	if !exists {
		return "", 0, fmt.Errorf("userID no existe en el contexto")
	}

	userIDStr, ok := userIDIface.(string)
	if !ok {
		return "", 0, fmt.Errorf("userID no es string")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		log.Printf("Error al convertir userID '%s' a uint64: %v", userIDStr, err)
		return "", 0, fmt.Errorf("userID inválido")
	}

	tokenIface, exists := c.Get("token")
	if !exists {
		return "", 0, fmt.Errorf("token no existe en el contexto")
	}

	tokenStr, ok := tokenIface.(string)
	if !ok {
		return "", 0, fmt.Errorf("token no es string")
	}

	return tokenStr, userID, nil
}

func ValidateQueryPagination(c *gin.Context) (int, int, string, error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	searchQuery := c.DefaultQuery("search", "")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, 0, "", errors.New("Parámetro 'page' inválido: debe ser un número entero positivo")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		return 0, 0, "", errors.New("Parámetro 'page_size' inválido: debe ser un número entero positivo")
	}

	return page, pageSize, searchQuery, nil
}
