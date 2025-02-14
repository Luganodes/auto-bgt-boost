package api

import (
	"bgt_boost/internal/models"
	"errors"
	"math/big"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
)

var validate *validator.Validate

func SetupValidator() error {
	validate = validator.New()
	return nil
}

func validateStruct(s interface{}) error {
	return validate.Struct(s)
}

type UpdateValidatorRequest struct {
	OperatorAddress *string `json:"operatorAddress"`
	BoostThreshold  *string `json:"boostThreshold"`
}

func ValidateAddValidatorRequest(c *gin.Context) (models.Validator, error) {
	var body models.Validator
	if err := c.ShouldBindJSON(&body); err != nil {
		return models.Validator{}, err
	}
	if err := validateStruct(body); err != nil {
		return models.Validator{}, err
	}

	boostThreshold, ok := big.NewInt(0).SetString(body.BoostThreshold, 10)
	if !ok {
		return models.Validator{}, errors.New("invalid boostThreshold")
	}
	if boostThreshold.Cmp(big.NewInt(1e18)) < 0 {
		return models.Validator{}, errors.New("boostThreshold should be greater than 1e18")
	}
	return body, nil
}

func ValidateUpdateValidatorRequest(c *gin.Context) (UpdateValidatorRequest, error) {
	var body UpdateValidatorRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		return UpdateValidatorRequest{}, err
	}
	if err := validateStruct(body); err != nil {
		return UpdateValidatorRequest{}, err
	}

	boostThreshold, ok := big.NewInt(0).SetString(*body.BoostThreshold, 10)
	if !ok {
		return UpdateValidatorRequest{}, errors.New("invalid boostThreshold")
	}
	if boostThreshold.Cmp(big.NewInt(1e18)) < 0 {
		return UpdateValidatorRequest{}, errors.New("boostThreshold should be greater than 1e18")
	}
	return body, nil
}
