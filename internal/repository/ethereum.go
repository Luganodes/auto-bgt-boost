package repository

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/utils"
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthRepository interface {
	GetLatestBlock(ctx context.Context) (uint64, error)
	GetBlockTimestamp(ctx context.Context, blockNumber uint64) (time.Time, error)
	GetActivateBoostDelay(ctx context.Context) (uint64, error)
	GetUnboostedBalance(ctx context.Context, operatorAddress common.Address) (*big.Int, error)
	GetBoostedQueue(ctx context.Context, operatorAddress common.Address, validatorPubkey string) (BoostedQueue, error)
	CreateTransaction(ctx context.Context, fromAddress common.Address, toAddress common.Address, data []byte) (*types.Transaction, error)
	SendTransaction(ctx context.Context, signedTx *types.Transaction) (TransactionInfo, error)
}

type ethRepository struct {
	client *ethclient.Client
	config *config.Config
}

func NewEthRepository(client *ethclient.Client, config *config.Config) EthRepository {
	return &ethRepository{
		client: client,
		config: config,
	}
}

func (r *ethRepository) GetLatestBlock(ctx context.Context) (uint64, error) {
	operation := func() (uint64, error) {
		block, err := r.client.BlockNumber(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch block number: %w", err)
		}
		return block, nil
	}
	return backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
}

func (r *ethRepository) GetBlockTimestamp(ctx context.Context, blockNumber uint64) (time.Time, error) {
	operation := func() (time.Time, error) {
		block, err := r.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to fetch block: %w", err)
		}
		return time.Unix(int64(block.Time()), 0), nil
	}
	return backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
}

func (r *ethRepository) callContract(ctx context.Context, callMsg ethereum.CallMsg) ([]byte, error) {
	operation := func() ([]byte, error) {
		return r.client.CallContract(ctx, callMsg, nil)
	}
	return backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
}

func (r *ethRepository) GetActivateBoostDelay(ctx context.Context) (uint64, error) {
	callMsg := ethereum.CallMsg{
		To:   &r.config.BGTContract.Address,
		Data: r.config.BGTContract.ABI.Methods["activateBoostDelay"].ID,
	}

	response, err := r.callContract(ctx, callMsg)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %w", err)
	}
	delay := new(big.Int).SetBytes(response)
	return delay.Uint64(), nil
}

func (r *ethRepository) GetUnboostedBalance(ctx context.Context, operatorAddress common.Address) (*big.Int, error) {
	data, err := r.config.BGTContract.ABI.Pack("unboostedBalanceOf", operatorAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}
	callMsg := ethereum.CallMsg{
		To:   &r.config.BGTContract.Address,
		Data: data,
	}

	response, err := r.callContract(ctx, callMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}
	balance := new(big.Int).SetBytes(response)
	return balance, nil
}

type BoostedQueue struct {
	Balance     *big.Int
	BlockNumber uint64
}

func (r *ethRepository) GetBoostedQueue(ctx context.Context, operatorAddress common.Address, pubkey string) (BoostedQueue, error) {
	data, err := r.config.BGTContract.ABI.Pack("boostedQueue", operatorAddress, common.FromHex(pubkey))
	if err != nil {
		return BoostedQueue{}, fmt.Errorf("failed to pack data: %w", err)
	}
	callMsg := ethereum.CallMsg{
		To:   &r.config.BGTContract.Address,
		Data: data,
	}

	response, err := r.callContract(ctx, callMsg)
	if err != nil {
		return BoostedQueue{}, fmt.Errorf("failed to call contract: %w", err)
	}
	result, err := r.config.BGTContract.ABI.Methods["boostedQueue"].Outputs.UnpackValues(response)
	if err != nil {
		return BoostedQueue{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return BoostedQueue{
		Balance:     result[1].(*big.Int),
		BlockNumber: uint64(result[0].(uint32)),
	}, nil
}

type TransactionInfo struct {
	TransactionHash string
	TransactionFee  float64
	BlockNumber     uint64
	BlockTimestamp  time.Time
}

func (r *ethRepository) CreateTransaction(ctx context.Context, fromAddress common.Address, toAddress common.Address, data []byte) (*types.Transaction, error) {
	operation := func() (*types.Transaction, error) {
		nonce, err := r.client.PendingNonceAt(ctx, fromAddress)
		if err != nil {
			log.Println("failed to get nonce: ", err.Error())
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		gasPrice, err := r.client.SuggestGasPrice(ctx)
		if err != nil {
			log.Println("failed to get gas price: ", err.Error())
			return nil, fmt.Errorf("failed to get gas price: %w", err)
		}
		tipCap, err := r.client.SuggestGasTipCap(ctx)
		if err != nil {
			log.Println("failed to get gas tip cap: ", err.Error())
			return nil, fmt.Errorf("failed to get gas tip cap: %w", err)
		}
		tx := types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			GasTipCap: tipCap,
			GasFeeCap: gasPrice,
			Gas:       uint64(r.config.GasLimit),
			To:        &toAddress,
			Value:     big.NewInt(0),
			Data:      data,
		})
		return tx, nil
	}
	return backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
}

func (r *ethRepository) SendTransaction(ctx context.Context, signedTx *types.Transaction) (TransactionInfo, error) {
	operation := func() (*types.Receipt, error) {
		err := r.client.SendTransaction(ctx, signedTx)
		if err != nil {
			log.Println("failed to send transaction: ", err.Error())
			return nil, fmt.Errorf("failed to send transaction: %w", err)
		}
		log.Println("Transaction sent: ", signedTx.Hash().Hex())

		receipt, err := bind.WaitMinedHash(ctx, r.client, signedTx.Hash())
		if err != nil {
			log.Println("failed to wait for transaction to be mined: ", err.Error())
			return nil, fmt.Errorf("failed to wait for transaction to be mined: %w", err)
		}
		return receipt, nil
	}
	receipt, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to wait for transaction to be mined: %w", err)
	}
	if receipt.Status == 0 {
		return TransactionInfo{}, fmt.Errorf("transaction failed")
	}

	log.Println("Transaction mined in block: ", receipt.BlockNumber.Uint64())

	gasUsed := receipt.GasUsed
	effectiveGasPrice := receipt.EffectiveGasPrice
	transactionFee := new(big.Int).Mul(big.NewInt(int64(gasUsed)), effectiveGasPrice)
	blockTimestamp, err := r.GetBlockTimestamp(ctx, receipt.BlockNumber.Uint64())
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to get block timestamp: %w", err)
	}
	return TransactionInfo{
		TransactionHash: signedTx.Hash().Hex(),
		TransactionFee:  utils.ConvertWeiToEther(transactionFee),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		BlockTimestamp:  blockTimestamp,
	}, nil
}
