package repository

import (
	"bgt_boost/internal/config"
	"context"
	"fmt"
	"log"

	infisical "github.com/infisical/go-sdk"
)

type InfisicalRepository interface {
	GetPrivateKey(ctx context.Context, operatorAddress string) (string, error)
}

type infisicalRepository struct {
	client *infisical.InfisicalClientInterface
	config *config.Config
}

func NewInfisicalRepository(client *infisical.InfisicalClientInterface, config *config.Config) InfisicalRepository {
	if _, err := (*client).Auth().UniversalAuthLogin(config.InfisicalConfig.ClientID, config.InfisicalConfig.ClientSecret); err != nil {
		panic(fmt.Sprintf("cannot login to infisical: %s", err))
	}
	log.Println("✅ Connected to Infisical")
	return &infisicalRepository{
		client: client,
		config: config,
	}
}

func (r *infisicalRepository) GetPrivateKey(ctx context.Context, operatorAddress string) (string, error) {
	encryptedPrivateKey, err := (*r.client).Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   operatorAddress,
		ProjectID:   r.config.InfisicalConfig.ProjectID,
		Environment: r.config.InfisicalConfig.Environment,
		SecretPath:  "/",
	})
	if err != nil {
		return "", err
	}
	return (*r.client).Kms().DecryptData(infisical.KmsDecryptDataOptions{
		KeyId:      r.config.InfisicalConfig.KeyId,
		Ciphertext: encryptedPrivateKey.SecretValue,
	})
}
