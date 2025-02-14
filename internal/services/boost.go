package services

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/models"
	"bgt_boost/internal/repository"
	"context"
	"errors"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func BoostValidator(ctx context.Context, config *config.Config, dbRepository *repository.DbRepository, ethRepository *repository.EthRepository, infisicalRepository *repository.InfisicalRepository) error {
	validators, err := (*dbRepository).GetValidators(ctx)
	if err != nil {
		return err
	}
	// queue any available boosts
	for _, validator := range validators {
		alreadyBoosted, err := (*dbRepository).DoesQueueBoostExist(ctx, validator.Pubkey)
		if err != nil {
			return err
		}
		if alreadyBoosted {
			continue
		}
		unboostedBalance, err := (*ethRepository).GetUnboostedBalance(ctx, common.HexToAddress(validator.OperatorAddress))
		if err != nil {
			return err
		}
		boostThreshold, ok := big.NewInt(0).SetString(validator.BoostThreshold, 10)
		if !ok {
			return errors.New("invalid boostThreshold")
		}
		if unboostedBalance.Cmp(boostThreshold) > 0 {
			privateKey, err := (*infisicalRepository).GetPrivateKey(ctx, validator.OperatorAddress)
			if err != nil {
				return err
			}
			transactionInfo, err := (*ethRepository).QueueBoost(ctx, privateKey, validator.Pubkey, unboostedBalance)
			if err != nil {
				return err
			}
			log.Println("queueBoost", transactionInfo.TransactionHash)
			err = (*dbRepository).AddQueueBoost(ctx, models.QueueBoost{
				Activated:       false,
				ValidatorPubkey: validator.Pubkey,
				OperatorAddress: validator.OperatorAddress,
				Amount:          unboostedBalance.String(),
				TransactionHash: transactionInfo.TransactionHash,
				BlockNumber:     transactionInfo.BlockNumber,
				BlockTimestamp:  transactionInfo.BlockTimestamp,
				Fee:             transactionInfo.TransactionFee,
				TransactionFrom: validator.OperatorAddress,
				ToContract:      config.BGTContract.Address.Hex(),
			})
			if err != nil {
				return err
			}
		}

		// activate any inactive boosts
		inactiveBoosts, err := (*dbRepository).GetInActiveBoosts(ctx)
		if err != nil {
			return err
		}
		for _, inactiveBoost := range inactiveBoosts {
			activationDelay, err := (*ethRepository).GetActivateBoostDelay(ctx)
			if err != nil {
				return err
			}
			currentBlock, err := (*ethRepository).GetLatestBlock(ctx)
			if err != nil {
				return err
			}
			if currentBlock > inactiveBoost.BlockNumber+activationDelay {
				privateKey, err := (*infisicalRepository).GetPrivateKey(ctx, inactiveBoost.OperatorAddress)
				if err != nil {
					return err
				}
				transactionInfo, err := (*ethRepository).ActivateBoost(ctx, privateKey, inactiveBoost.ValidatorPubkey)
				if err != nil {
					return err
				}
				log.Println("activateBoost", transactionInfo.TransactionHash)
				err = (*dbRepository).AddActivateBoost(ctx, models.ActivateBoost{
					Amount:          inactiveBoost.Amount,
					ValidatorPubkey: inactiveBoost.ValidatorPubkey,
					OperatorAddress: inactiveBoost.OperatorAddress,
					TransactionHash: transactionInfo.TransactionHash,
					BlockNumber:     transactionInfo.BlockNumber,
					BlockTimestamp:  transactionInfo.BlockTimestamp,
					Fee:             transactionInfo.TransactionFee,
					TransactionFrom: inactiveBoost.TransactionFrom,
					ToContract:      config.BGTContract.Address.Hex(),
				})
				if err != nil {
					return err
				}
				err = (*dbRepository).MarkBoostAsActivated(ctx, inactiveBoost.TransactionHash)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
