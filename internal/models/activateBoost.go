package models

import "time"

type ActivateBoost struct {
	Amount          string    `bson:"amount"`
	ValidatorPubkey string    `bson:"validatorPubkey"`
	OperatorAddress string    `bson:"operatorAddress"`
	TransactionHash string    `bson:"transactionHash"`
	BlockNumber     uint64    `bson:"blockNumber"`
	BlockTimestamp  time.Time `bson:"blockTimestamp"`
	Fee             float64   `bson:"fee"`
	TransactionFrom string    `bson:"transactionFrom"`
	ToContract      string    `bson:"toContract"`
}
