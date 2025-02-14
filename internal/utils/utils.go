package utils

import (
	"log"
	"math/big"

	"github.com/robfig/cron/v3"
)

func ConvertWeiToEther(wei *big.Int) float64 {
	etherValue := new(big.Float).SetInt(wei)
	etherValue.Quo(etherValue, big.NewFloat(1e18)) // Divide by 10^18
	result, _ := etherValue.Float64()              // Convert big.Float to float64
	return result
}

func PrintNextExecution(c *cron.Cron) {
	entries := c.Entries()
	if len(entries) > 0 {
		nextRun := entries[0].Next
		log.Printf("Next cron execution scheduled for: %v", nextRun)
	}
}
