package repository

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/utils"
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthRepository interface {
	GetLatestBlock(ctx context.Context) (uint64, error)
	getBlockTimestamp(ctx context.Context, blockNumber uint64) (time.Time, error)
	GetActivateBoostDelay(ctx context.Context) (uint64, error)
	GetUnboostedBalance(ctx context.Context, operatorAddress common.Address) (*big.Int, error)
	QueueBoost(ctx context.Context, privateKey string, pubkey string, amount *big.Int) (TransactionInfo, error)
	ActivateBoost(ctx context.Context, privateKey string, pubkey string) (TransactionInfo, error)
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
	block, err := r.client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch block number: %w", err)
	}
	return block, nil
}

func (r *ethRepository) getBlockTimestamp(ctx context.Context, blockNumber uint64) (time.Time, error) {
	block, err := r.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to fetch block: %w", err)
	}
	return time.Unix(int64(block.Time()), 0), nil
}

func (r *ethRepository) GetActivateBoostDelay(ctx context.Context) (uint64, error) {
	callMsg := ethereum.CallMsg{
		To:   &r.config.BGTContract.Address,
		Data: r.config.BGTContract.ABI.Methods["activateBoostDelay"].ID,
	}

	response, err := r.client.CallContract(ctx, callMsg, nil)
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

	response, err := r.client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}
	balance := new(big.Int).SetBytes(response)
	return balance, nil
}

type TransactionInfo struct {
	TransactionHash string
	TransactionFee  float64
	BlockNumber     uint64
	BlockTimestamp  time.Time
}

func (r *ethRepository) sendTransaction(ctx context.Context, privateKey *ecdsa.PrivateKey, contractAddress common.Address, data []byte) (TransactionInfo, error) {
	chainID, err := r.client.NetworkID(ctx)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to get chain ID: %w", err)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := r.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to get nonce: %w", err)
	}
	gasPrice, err := r.client.SuggestGasPrice(ctx)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to get gas price: %w", err)
	}
	tipCap, err := r.client.SuggestGasTipCap(context.Background())
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to get gas tip cap: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to create transactor: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(r.config.GasLimit)
	auth.GasPrice = gasPrice
	auth.GasTipCap = tipCap

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: gasPrice,
		Gas:       auth.GasLimit,
		To:        &contractAddress,
		Value:     auth.Value,
		Data:      data,
	})

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	log.Println("Sending transaction: ", signedTx.Hash().Hex())
	err = r.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to send transaction: %w", err)
	}
	receipt, err := bind.WaitMinedHash(context.Background(), r.client, signedTx.Hash())
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed waiting for transaction to be mined: %w", err)
	}
	if receipt.Status == 0 {
		return TransactionInfo{}, fmt.Errorf("transaction failed")
	}
	log.Println("Transaction mined in block: ", receipt.BlockNumber.Uint64())
	gasUsed := receipt.GasUsed
	effectiveGasPrice := receipt.EffectiveGasPrice
	transactionFee := new(big.Int).Mul(big.NewInt(int64(gasUsed)), effectiveGasPrice)
	blockTimestamp, err := r.getBlockTimestamp(ctx, receipt.BlockNumber.Uint64())
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

func (r *ethRepository) QueueBoost(ctx context.Context, privateKey string, pubkey string, amount *big.Int) (TransactionInfo, error) {
	ecdsaPrivateKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
	}
	data, err := r.config.BGTContract.ABI.Pack("queueBoost", common.FromHex(pubkey), amount)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to pack data: %w", err)
	}
	return r.sendTransaction(ctx, ecdsaPrivateKey, r.config.BGTContract.Address, data)
}

func (r *ethRepository) ActivateBoost(ctx context.Context, privateKey string, pubkey string) (TransactionInfo, error) {
	ecdsaPrivateKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to convert private key to ECDSA: %w", err)
	}
	operatorAddress := crypto.PubkeyToAddress(ecdsaPrivateKey.PublicKey)
	data, err := r.config.BGTContract.ABI.Pack("activateBoost", operatorAddress, common.FromHex(pubkey))
	if err != nil {
		return TransactionInfo{}, fmt.Errorf("failed to pack data: %w", err)
	}
	return r.sendTransaction(ctx, ecdsaPrivateKey, r.config.BGTContract.Address, data)
}
