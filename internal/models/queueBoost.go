package models

import "time"

type QueueBoost struct {
	ValidatorPubkey string    `bson:"validatorPubkey"`
	OperatorAddress string    `bson:"operatorAddress"`
	BlockNumber     uint64    `bson:"blockNumber"`
	Amount          string    `bson:"amount"`
	TransactionHash string    `bson:"transactionHash"`
	BlockTimestamp  time.Time `bson:"blockTimestamp"`
	Fee             float64   `bson:"fee"`
	TransactionFrom string    `bson:"transactionFrom"`
	ToContract      string    `bson:"toContract"`
}
