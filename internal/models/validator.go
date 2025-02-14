package models

type Validator struct {
	Pubkey          string `bson:"pubkey,unique" json:"pubkey" validate:"required"`
	OperatorAddress string `bson:"operatorAddress" json:"operatorAddress" validate:"required"`
	BoostThreshold  string `bson:"boostThreshold" json:"boostThreshold" validate:"required"`
}
