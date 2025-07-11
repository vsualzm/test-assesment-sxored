package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// LoanApplication
type LoanApplication struct {
	ID                int        `json:"id"`
	ApplicantName     string     `json:"applicant_name"`
	ApplicantSSN      string     `json:"applicant_ssn"`
	MaskedSSN         string     `json:"masked_ssn,omitempty"`
	LoanAmount        float64    `json:"loan_amount"`
	LoanPurpose       string     `json:"loan_purpose"`
	AnnualIncome      float64    `json:"annual_income"`
	CreditScore       int        `json:"credit_score"`
	Status            string     `json:"status"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	DocumentsUploaded []string   `json:"documents_uploaded"`
}

var loanApplications = make(map[int]LoanApplication)
var currentID = 1
var jwtSecret = []byte("supersecretkey")

type DocumentJob struct {
	AppID    int
	FilePath string
}

var documentQueue = make(chan DocumentJob, 100) // worker queue
var processingStatus = make(map[int]string)     // track progress status

// RUNNING API: go run main.go
func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(ErrorHandlingMiddleware())
	r.Use(RequestLogger())

	r.POST("/login", LoginHandler)
	r.GET("/test", TestAPI)

	auth := r.Group("/")
	auth.Use(JWTMiddleware())
	auth.POST("/loan-applications", RoleRequired("loan_officer"), CreateLoanApplication)
	auth.GET("/loan-applications", RoleRequired("loan_officer", "underwriter"), GetLoanApplications)
	auth.GET("/loan-applications/:id", RoleRequired("loan_officer", "underwriter", "applicant"), GetLoanApplicationByID)
	auth.PUT("/loan-applications/:id/status", RoleRequired("underwriter"), UpdateStatus)
	auth.POST("/loan-applications/:id/documents", RoleRequired("loan_officer"), UploadDocuments)

	go func() {
		for job := range documentQueue {
			processDocument(job)
		}
	}()

	r.Run(":8080")
}

func processDocument(job DocumentJob) {
	log.Printf("[PROCESSING] AppID=%d, File=%s", job.AppID, job.FilePath)
	processingStatus[job.AppID] = "processing"

	// Dummy proses: di sini nanti tinggal panggil pdf extractor
	time.Sleep(3 * time.Second) // simulasi proses PDF

	// Update hasil ke status map
	log.Printf("[DONE] AppID=%d processed", job.AppID)
	processingStatus[job.AppID] = "completed"
}

// testing API
func TestAPI(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "API RUNNING"})
}

// sensor
func maskSSN(ssn string) string {
	if len(ssn) < 4 {
		return "***-**-xxxx"
	}
	return "***-**-" + ssn[len(ssn)-4:]
}

func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				appID := c.Param("id")
				log.Printf("[PANIC] AppID=%s: %v\n%s", appID, rec, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":          "Internal server error",
					"application_id": appID,
				})
			}
		}()
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[REQUEST] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func RoleRequired(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, allowed := range allowedRoles {
			if role == allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	}
}

func LoginHandler(c *gin.Context) {
	var logInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&logInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// hardcoded akun
	// jenis account ada 3
	accounts := map[string]struct {
		Password string
		Role     string
	}{
		"officer":     {"123456", "loan_officer"},
		"underwriter": {"123456", "underwriter"},
		"applicant":   {"123456", "applicant"},
	}
	account, ok := accounts[logInput.Username]
	if !ok || logInput.Password != account.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	claims := jwt.MapClaims{
		"user_id": logInput.Username,
		"role":    account.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(jwtSecret)
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func CreateLoanApplication(c *gin.Context) {
	var input LoanApplication
	if err := c.ShouldBindJSON(&input); err != nil || input.ApplicantName == "" || input.ApplicantSSN == "" || input.LoanAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	input.ID = currentID
	input.Status = "pending"
	input.MaskedSSN = maskSSN(input.ApplicantSSN)
	input.SubmittedAt = time.Now()
	loanApplications[currentID] = input
	currentID++
	c.JSON(http.StatusCreated, input)
}

func GetLoanApplicationByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	app, exists := loanApplications[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}
	role := c.GetString("role")
	userID := c.GetString("user_id")
	if role == "applicant" && userID != app.ApplicantName {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view this application"})
		return
	}
	app.MaskedSSN = maskSSN(app.ApplicantSSN)
	c.JSON(http.StatusOK, app)
}

// ini untuk cek request endpoint : GET /loan-applications?limit=5&offset=0
// GET /loan-applications?status=approved&name=john&limit=3&offset=3
func GetLoanApplications(c *gin.Context) {
	status := strings.ToLower(c.Query("status"))
	name := strings.ToLower(c.Query("name"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	var filtered []LoanApplication
	for _, app := range loanApplications {
		if status != "" && strings.ToLower(app.Status) != status {
			continue
		}
		if name != "" && !strings.Contains(strings.ToLower(app.ApplicantName), name) {
			continue
		}
		app.MaskedSSN = maskSSN(app.ApplicantSSN)
		filtered = append(filtered, app)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	c.JSON(http.StatusOK, gin.H{
		"total":   len(filtered),
		"limit":   limit,
		"offset":  offset,
		"results": filtered[offset:end],
	})
}

// ini untuk update status
func UpdateStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	app, exists := loanApplications[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}
	app.Status = body.Status
	now := time.Now()
	app.ProcessedAt = &now
	loanApplications[id] = app
	c.JSON(http.StatusOK, app)
}

// upload nya masih nama aja
// func UploadDocuments(c *gin.Context) {
// 	id, _ := strconv.Atoi(c.Param("id"))
// 	app, exists := loanApplications[id]
// 	if !exists {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
// 		return
// 	}
// 	var body struct {
// 		Documents []string `json:"documents_uploaded"`
// 	}
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}
// 	app.DocumentsUploaded = append(app.DocumentsUploaded, body.Documents...)
// 	loanApplications[id] = app
// 	c.JSON(http.StatusOK, app)
// }

func GetProcessingStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	status, exists := processingStatus[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "No processing job for this application"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"application_id": id, "status": status})
}

func UploadDocuments(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	app, exists := loanApplications[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	if !strings.HasSuffix(file.Filename, ".pdf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a PDF"})
		return
	}

	savePath := fmt.Sprintf("uploads/%d_%s", id, file.Filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	app.DocumentsUploaded = append(app.DocumentsUploaded, file.Filename)
	loanApplications[id] = app

	processingStatus[id] = "queued"

	documentQueue <- DocumentJob{
		AppID:    id,
		FilePath: savePath,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded and queued for processing",
	})
}
