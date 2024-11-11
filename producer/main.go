package main

import (
	"context"
	"fmt"
	"gains/Endpoints"
	"gains/Properties"
	"gains/TokenManager"
	"github.com/segmentio/kafka-go"
	"log"
	"log/slog"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	sigs := make(chan os.Signal, 1)

	config, err := Properties.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}
	tm := TokenManager.NewTokenManager(config.AppKey, config.AppSecret)

	conn, err := kafka.DialLeader(ctx, "tcp", "localhost:9092", "my-topic", 0)
	if err != nil {
		log.Fatalf("failed to connect to Kafka: %v", err)
	}

	defer conn.Close()

	schwabAPI, err := initializeTokens(config, tm)
	if err != nil {
		slog.Error("Failed to initialize tokens:", err)
	}
	accountNumbers, _ := schwabAPI.GetAccountNumbers()

	// Start a separate goroutine to check for any recent orders for selected schwab account
	go func() {
		for {
			fmt.Println(time.Now().String())
			orders, err := schwabAPI.GetRecentOrders(accountNumbers.HashValue)
			if err != nil {
				slog.Error("Failed to get recent orders:", err)
			} else {
				_, err = conn.WriteMessages(kafka.Message{
					Value: orders,
				})
			}
			time.Sleep(60 * time.Second)
		}
	}()

	// Start a separate goroutine to refresh bearer token every 25 minutes
	go func() {
		for {
			tm.RefreshTokens()
			time.Sleep(1500 * time.Second)
		}
	}()

	// Wait for interrupt signal
	<-sigs
	fmt.Println("Shutting down gracefully...")
}

// initializeTokens checks if Schwab auth tokens in config are still valid. If not, retrieve new ones.
func initializeTokens(config *Properties.Config, tm *TokenManager.TokenManager) (*Endpoints.SchwabAPI, error) {
	var schwabAPI *Endpoints.SchwabAPI

	// Set tokens if available
	if config.BearerToken != "" && config.RefreshToken != "" {
		tm.SetAuthTokens(config.BearerToken, config.RefreshToken)
		schwabAPI = Endpoints.NewSchwabAPI(tm.BearerToken)

		if _, err := schwabAPI.GetAccountNumbers(); err != nil {
			slog.Warn("Cached tokens are invalid, need to grab new ones.")
			err = tm.RefreshTokens()
			if err != nil {
				tm.GetAuthTokens()
			}
			err := config.UpdateTokens(tm.BearerToken, tm.RefreshToken)
			if err != nil {
				return nil, err
			}
			schwabAPI = Endpoints.NewSchwabAPI(tm.BearerToken)
		}
	} else {
		tm.GetAuthTokens()
		schwabAPI = Endpoints.NewSchwabAPI(tm.BearerToken)
		err := config.UpdateTokens(tm.BearerToken, tm.RefreshToken)
		if err != nil {
			return nil, err
		}
	}

	return schwabAPI, nil
}
