package main

import (
	"bgt_boost/internal/api"
	"bgt_boost/internal/config"
	"bgt_boost/internal/repository"
	"bgt_boost/internal/services"
	"bgt_boost/internal/utils"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/robfig/cron/v3"
)

func main() {
	config := config.LoadConfig()
	db, err := repository.ConnectToDb(config)
	if err != nil {
		panic(fmt.Sprintf("cannot connect to db: %s", err))
	}
	defer db.Disconnect()

	go func() {
		api.SetupValidator()
		server := api.NewServer(config, &db)
		if err := server.ListenAndServe(); err != nil {
			panic(fmt.Sprintf("cannot start server: %s", err))
		}
	}()

	ethClient, err := ethclient.Dial(config.RPC_URL)
	if err != nil {
		panic(fmt.Sprintf("cannot connect to eth client: %s", err))
	}
	defer ethClient.Close()
	ethRepository := repository.NewEthRepository(ethClient, config)

	requestRepository := repository.NewRequestRepository([]int{})
	signerService := services.NewSignerService(config.Web3SignerURL, &requestRepository)
	boostService := services.NewBoostService(config, &db, &ethRepository, &signerService)

	c := cron.New(cron.WithSeconds())
	_, err = c.AddFunc(config.CronSchedule, func() {
		err = boostService.BoostValidator(context.Background())
		if err != nil {
			panic(fmt.Sprintf("cannot boost validator: %s", err))
		}
		utils.PrintNextExecution(c)
	})

	err = boostService.BoostValidator(context.Background())
	if err != nil {
		panic(fmt.Sprintf("cannot boost validator: %s", err))
	}
	c.Start()
	utils.PrintNextExecution(c)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	// Cleanup
	log.Println("Shutting down gracefully...")
	c.Stop()
}
