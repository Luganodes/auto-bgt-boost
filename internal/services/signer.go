package services

import (
	"bgt_boost/internal/repository"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

type SignerService interface {
	SignTransaction(ctx context.Context, fromAddress string, tx *types.Transaction) (string, error)
}

type signerService struct {
	signerURL         string
	requestRepository *repository.RequestRepository
}

func NewSignerService(signerURL string, requestRepository *repository.RequestRepository) SignerService {
	return &signerService{
		signerURL:         signerURL,
		requestRepository: requestRepository,
	}
}

type Transaction struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	Gas                  string `json:"gas"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	Value                string `json:"value"`
	Nonce                string `json:"nonce"`
	Data                 string `json:"data"`
}

type SignTransactionRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []Transaction `json:"params"`
	ID      int           `json:"id"`
}

type SignTransactionResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

func (s *signerService) SignTransaction(ctx context.Context, fromAddress string, tx *types.Transaction) (string, error) {
	transaction := Transaction{
		From:                 fromAddress,
		To:                   tx.To().Hex(),
		Gas:                  fmt.Sprintf("0x%x", tx.Gas()),
		MaxFeePerGas:         fmt.Sprintf("0x%x", tx.GasPrice()),
		MaxPriorityFeePerGas: fmt.Sprintf("0x%x", tx.GasPrice()),
		Value:                fmt.Sprintf("0x%x", tx.Value()),
		Nonce:                fmt.Sprintf("0x%x", tx.Nonce()),
		Data:                 hex.EncodeToString(tx.Data()),
	}

	body := SignTransactionRequest{
		JSONRPC: "2.0",
		Method:  "eth_signTransaction",
		Params:  []Transaction{transaction},
		ID:      1,
	}

	response, err := (*s.requestRepository).Post(ctx, s.signerURL, nil, body)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	var responseBody SignTransactionResponse
	err = json.Unmarshal(response, &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return responseBody.Result, nil
}
