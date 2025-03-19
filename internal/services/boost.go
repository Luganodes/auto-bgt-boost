package services

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/models"
	"bgt_boost/internal/repository"
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BoostService interface {
	BoostValidator(ctx context.Context) error
}

type boostService struct {
	config        *config.Config
	dbRepository  *repository.DbRepository
	ethRepository *repository.EthRepository
	signerService *SignerService
}

func NewBoostService(config *config.Config, dbRepository *repository.DbRepository, ethRepository *repository.EthRepository, signerService *SignerService) BoostService {
	return &boostService{
		config:        config,
		dbRepository:  dbRepository,
		ethRepository: ethRepository,
		signerService: signerService,
	}
}

func (s *boostService) BoostValidator(ctx context.Context) error {
	validators, err := (*s.dbRepository).GetValidators(ctx)
	if err != nil {
		return err
	}

	log.Printf("Found %d validators", len(validators))
	for _, validator := range validators {
		log.Println("Processing validator: ", validator.Pubkey)
		if err := s.processValidator(ctx, validator); err != nil {
			return err
		}
	}
	return nil
}

func (s *boostService) processValidator(ctx context.Context, validator models.Validator) error {
	if err := s.checkAndQueueBoost(ctx, validator); err != nil {
		return err
	}
	return s.checkAndActivateBoost(ctx, validator)
}

func (s *boostService) checkAndQueueBoost(ctx context.Context, validator models.Validator) error {
	unboostedBalance, err := (*s.ethRepository).GetUnboostedBalance(ctx, common.HexToAddress(validator.OperatorAddress))
	if err != nil {
		return err
	}
	log.Println("Checking queue boost condition")
	log.Printf("Unboosted balance: %s", unboostedBalance.String())
	log.Printf("Boost threshold: %s", validator.BoostThreshold)
	boostThreshold, ok := big.NewInt(0).SetString(validator.BoostThreshold, 10)
	if !ok {
		return errors.New("invalid boostThreshold")
	}
	if unboostedBalance.Cmp(boostThreshold) > 0 {
		log.Printf("Queueing boost: %s", unboostedBalance.String())
		transactionInfo, err := s.queueBoost(ctx, validator.OperatorAddress, validator.Pubkey, unboostedBalance)
		if err != nil {
			return err
		}
		log.Printf("Queued boost: %s", transactionInfo.TransactionHash)
		return s.recordQueueBoost(ctx, validator, unboostedBalance, transactionInfo)
	}
	log.Printf("Queue boost condition not met")
	return nil
}

func (s *boostService) recordQueueBoost(ctx context.Context, validator models.Validator, amount *big.Int, transactionInfo repository.TransactionInfo) error {
	return (*s.dbRepository).AddQueueBoost(ctx, models.QueueBoost{
		ValidatorPubkey: validator.Pubkey,
		OperatorAddress: validator.OperatorAddress,
		Amount:          amount.String(),
		TransactionHash: transactionInfo.TransactionHash,
		BlockNumber:     transactionInfo.BlockNumber,
		BlockTimestamp:  transactionInfo.BlockTimestamp,
		Fee:             transactionInfo.TransactionFee,
		TransactionFrom: validator.OperatorAddress,
		ToContract:      s.config.BGTContract.Address.Hex(),
	})
}

func (s *boostService) checkAndActivateBoost(ctx context.Context, validator models.Validator) error {
	log.Println("Checking activate boost condition")
	boostedQueue, err := (*s.ethRepository).GetBoostedQueue(ctx, common.HexToAddress(validator.OperatorAddress), validator.Pubkey)
	if err != nil {
		return err
	}
	log.Printf("Boosted queue balance: %s", boostedQueue.Balance.String())
	log.Printf("Boosted queue block number: %d", boostedQueue.BlockNumber)

	currentBlock, err := (*s.ethRepository).GetLatestBlock(ctx)
	if err != nil {
		return err
	}
	activationDelay, err := (*s.ethRepository).GetActivateBoostDelay(ctx)
	if err != nil {
		return err
	}
	log.Printf("Activation delay: %d", activationDelay)
	log.Printf("Current block: %d", currentBlock)

	if boostedQueue.Balance.Cmp(big.NewInt(0)) > 0 && currentBlock > boostedQueue.BlockNumber+activationDelay {
		log.Printf("Activating boost: %s", boostedQueue.Balance.String())
		transactionInfo, err := s.activateBoost(ctx, validator.OperatorAddress, validator.Pubkey)
		if err != nil {
			return err
		}
		log.Printf("Activated boost: %s", transactionInfo.TransactionHash)
		return s.recordActivateBoost(ctx, validator, boostedQueue, transactionInfo)
	}
	log.Printf("Activate boost condition not met")
	return nil
}

func (s *boostService) recordActivateBoost(ctx context.Context, validator models.Validator, boostedQueue repository.BoostedQueue, transactionInfo repository.TransactionInfo) error {
	return (*s.dbRepository).AddActivateBoost(ctx, models.ActivateBoost{
		Amount:          boostedQueue.Balance.String(),
		ValidatorPubkey: validator.Pubkey,
		OperatorAddress: validator.OperatorAddress,
		TransactionHash: transactionInfo.TransactionHash,
		BlockNumber:     transactionInfo.BlockNumber,
		BlockTimestamp:  transactionInfo.BlockTimestamp,
		Fee:             transactionInfo.TransactionFee,
		TransactionFrom: validator.OperatorAddress,
		ToContract:      s.config.BGTContract.Address.Hex(),
	})
}

func (s *boostService) queueBoost(ctx context.Context, operatorAddress string, pubkey string, amount *big.Int) (repository.TransactionInfo, error) {
	data, err := s.config.BGTContract.ABI.Pack("queueBoost", common.FromHex(pubkey), amount)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to pack data: %w", err)
	}
	tx, err := (*s.ethRepository).CreateTransaction(ctx, common.HexToAddress(operatorAddress), s.config.BGTContract.Address, data)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to create transaction: %w", err)
	}
	signedTx, err := (*s.signerService).SignTransaction(ctx, operatorAddress, tx)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Convert hex string back to transaction
	signedTxBytes := common.FromHex(signedTx)
	decodedTx := new(types.Transaction)
	if err := decodedTx.UnmarshalBinary(signedTxBytes); err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to decode signed transaction: %w", err)
	}

	txInfo, err := (*s.ethRepository).SendTransaction(ctx, decodedTx)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to send transaction: %w", err)
	}
	return txInfo, nil
}

func (s *boostService) activateBoost(ctx context.Context, operatorAddress string, pubkey string) (repository.TransactionInfo, error) {
	data, err := s.config.BGTContract.ABI.Pack("activateBoost", common.HexToAddress(operatorAddress), common.FromHex(pubkey))
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to pack data: %w", err)
	}
	tx, err := (*s.ethRepository).CreateTransaction(ctx, common.HexToAddress(operatorAddress), s.config.BGTContract.Address, data)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to create transaction: %w", err)
	}
	signedTx, err := (*s.signerService).SignTransaction(ctx, operatorAddress, tx)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Convert hex string back to transaction
	signedTxBytes := common.FromHex(signedTx)
	decodedTx := new(types.Transaction)
	if err := decodedTx.UnmarshalBinary(signedTxBytes); err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to decode signed transaction: %w", err)
	}

	txInfo, err := (*s.ethRepository).SendTransaction(ctx, decodedTx)
	if err != nil {
		return repository.TransactionInfo{}, fmt.Errorf("failed to send transaction: %w", err)
	}
	return txInfo, nil
}
