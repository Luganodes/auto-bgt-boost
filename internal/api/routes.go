package api

import (
	"bgt_boost/internal/repository"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) RegisterRoutes() http.Handler {
	mode := gin.DebugMode
	if s.config.Environment == "production" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	r := gin.Default()
	r.Use(cors.Default())
	r.Use(DatabaseMiddleware(s.dbRepository))

	// Public routes
	r.GET("/", s.HelloWorldHandler)

	// Admin routes group
	admin := r.Group("/")
	admin.Use(AdminMiddleware(s.config))
	{
		admin.GET("/validators", GetValidators)
		admin.POST("/validators", AddValidator)
		admin.PUT("/validators/:pubkey", UpdateValidator)
		admin.DELETE("/validators/:pubkey", DeleteValidator)
	}

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func GetValidators(c *gin.Context) {
	dbRepository, ok := c.MustGet("dbRepository").(*repository.DbRepository)
	if !ok {
		log.Println("Error getting dbRepository")
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	validators, err := (*dbRepository).GetValidators(c.Request.Context())
	if err != nil {
		log.Printf("Error getting validators: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"validators": validators})
}

func AddValidator(c *gin.Context) {
	dbRepository, ok := c.MustGet("dbRepository").(*repository.DbRepository)
	if !ok {
		log.Println("Error getting dbRepository")
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	body, err := ValidateAddValidatorRequest(c)
	if err != nil {
		UnprocessableEntityResponse(c, err.Error())
		return
	}
	exists, err := (*dbRepository).DoesValidatorExist(c.Request.Context(), body.Pubkey)
	if err != nil {
		log.Printf("Error checking if validator exists: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	if exists {
		BadRequestResponse(c, "Validator already exists")
		return
	}

	err = (*dbRepository).AddValidator(c.Request.Context(), body)
	if err != nil {
		log.Printf("Error adding validator: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	SuccessResponse(c, gin.H{"message": "Validator added successfully"})
}

func UpdateValidator(c *gin.Context) {
	dbRepository, ok := c.MustGet("dbRepository").(*repository.DbRepository)
	if !ok {
		log.Println("Error getting dbRepository")
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	body, err := ValidateUpdateValidatorRequest(c)
	if err != nil {
		log.Printf("Error validating update validator request: %v", err)
		UnprocessableEntityResponse(c, err.Error())
		return
	}
	validator, err := (*dbRepository).GetValidator(c.Request.Context(), c.Param("pubkey"))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			BadRequestResponse(c, "Validator does not exist")
			return
		}
		log.Printf("Error getting validator: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	if body.OperatorAddress != nil {
		validator.OperatorAddress = *body.OperatorAddress
	}
	if body.BoostThreshold != nil {
		validator.BoostThreshold = *body.BoostThreshold
	}
	err = (*dbRepository).UpdateValidator(c.Request.Context(), c.Param("pubkey"), validator)
	if err != nil {
		log.Printf("Error updating validator: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	SuccessResponse(c, gin.H{"message": "Validator updated successfully"})
}

func DeleteValidator(c *gin.Context) {
	dbRepository, ok := c.MustGet("dbRepository").(*repository.DbRepository)
	if !ok {
		log.Println("Error getting dbRepository")
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	pubkey := c.Param("pubkey")
	exists, err := (*dbRepository).DoesValidatorExist(c.Request.Context(), pubkey)
	if err != nil {
		log.Printf("Error checking if validator exists: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	if !exists {
		BadRequestResponse(c, "Validator does not exist")
		return
	}
	err = (*dbRepository).DeleteValidator(c.Request.Context(), pubkey)
	if err != nil {
		log.Printf("Error deleting validator: %v", err)
		InternalServerErrorResponse(c, "Internal server error")
		return
	}
	SuccessResponse(c, gin.H{"message": "Validator deleted successfully"})
}
