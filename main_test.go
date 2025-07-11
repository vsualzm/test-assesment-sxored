package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(ErrorHandlingMiddleware())
	r.Use(RequestLogger())

	r.POST("/login", LoginHandler)
	auth := r.Group("/")
	auth.Use(JWTMiddleware())
	auth.POST("/loan-applications", CreateLoanApplication)

	return r
}

func TestLoginSuccess(t *testing.T) {
	router := setupRouter()

	payload := map[string]string{
		"username": "officer",
		"password": "123456",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	assert.NotEmpty(t, resp["token"])
}

func TestCreateLoanApplicationUnauthorized(t *testing.T) {
	router := setupRouter()

	payload := map[string]interface{}{
		"applicant_name": "Jane Smith",
		"applicant_ssn":  "987-65-4321",
		"loan_amount":    20000,
		"loan_purpose":   "Kendaraan",
		"annual_income":  90000,
		"credit_score":   700,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/loan-applications", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Tidak menyertakan token

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// RUNING TESTING : go test -v
