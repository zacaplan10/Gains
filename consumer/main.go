package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gains/Data/JsonParser"
	"gains/TokenManager"
	"github.com/segmentio/kafka-go"
	"log"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"gains/Data"
	"gains/Endpoints"
	"gains/Properties"

	"os"
	"os/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals to allow for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	// Create a new reader for the Kafka topic
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "my-topic",
		GroupID: "my-group",
	})

	defer reader.Close()
	config, err := Properties.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}
	tm := TokenManager.NewTokenManager(config.AppKey, config.AppSecret)

	//Initialize Schwab api struct by grabbing tokens and get account numbers for this user
	schwabAPI, err := initializeTokens(config, tm)
	if err != nil {
		slog.Error("Failed to initialize tokens:", err)
	}
	accountNumbers, _ := schwabAPI.GetAccountNumbers()
	if accountNumbers.HashValue == "" {
		slog.Error("Error getting  account values")
	}

	// Connect to the database
	db, _ := Data.NewDatabaseHelperFromConnectionString(config.DBConnectionString)
	accountNumber := accountNumbers.AccountNumber
	hashedAccountNumber := db.GetHashedAccountNumber(accountNumber)
	if hashedAccountNumber == "" {
		db.InsertHashedAccountNumber(accountNumber, accountNumbers.HashValue)
	}

	// Start a separate goroutine to read messages from kafka stream
	go func() {
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("failed to read message: %v", err)
				return
			}
			var orders []JsonParser.Order
			err = json.Unmarshal(msg.Value, &orders)
			if err != nil {
				slog.Warn("Error parsing JSON: %v", err)
				continue
			}
			rowsInserted := db.InsertTransactionData(orders)
			if rowsInserted == 0 {
				continue
			}
			if containsSellOrder(orders) {
				transactions, err := db.GetUnmatchedTransactionsByAccountID(accountNumber)
				if err != nil {
					slog.Error("Error getting transactions for ticker:", err)
				}
				year := time.Now().UTC().Year()
				netChange := matchOrders(transactions, db)
				capitalGains := db.GetCapitalGainsBalanceForYear(accountNumber, year)
				fmt.Println("Net capital gains/losses for year " + strconv.Itoa(year) + " is: " + strconv.FormatInt(capitalGains, 10) + " after a change of: " + strconv.FormatInt(netChange, 10))
			}
			if err := reader.CommitMessages(context.Background(), msg); err != nil {
				log.Fatal(err)
			}
		}
	}()

	// Start a separate goroutine to refresh bearer token every 25 minutes
	go func() {
		for {
			err := tm.RefreshTokens()
			if err != nil {
				return
			}
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

// matchOrders takes in a list of transactions and matches any sells to buys (partial or fully) and updates the DB
func matchOrders(transactions []Data.TransactionData, db *Data.DatabaseHelper) int64 {
	tickerMap := make(map[string][]Data.TransactionData)
	tickerGainsMap := make(map[string]int64)

	// Populate the map
	for _, transaction := range transactions {
		tickerMap[transaction.StockTicker] = append(tickerMap[transaction.StockTicker], transaction)
	}

	var netChange int64 = 0
	var matchedActivityIds []int64
	for ticker, transactionsForTicker := range tickerMap {
		sort.Slice(transactionsForTicker, func(i, j int) bool {
			return transactionsForTicker[i].ActivityDate.Before(transactionsForTicker[j].ActivityDate)
		})

		tickerGainsMap[ticker] = 0
		buyQueue := []Data.TransactionData{}

		for _, transaction := range transactionsForTicker {
			if transaction.Matched == true {
				continue
			} else if transaction.OrderType == "BUY" {
				// Add BUY transactions to the queue without adjusting gains
				buyQueue = append(buyQueue, transaction)
			} else if transaction.OrderType == "SELL" {
				sharesToSell := transaction.ShareCount
				var gain int64
				// Process the sell by matching with buys in the queue
				for sharesToSell > 0 && len(buyQueue) > 0 {
					buy := &buyQueue[0] // Reference the first buy in the queue
					if buy.ShareCount <= sharesToSell {
						// Full match
						gain = (transaction.StockPrice - buy.StockPrice) * int64(buy.ShareCount)
						tickerGainsMap[ticker] += gain
						sharesToSell -= buy.ShareCount
						// Remove the buy from the queue as it is fully matched
						buyQueue = buyQueue[1:]
						additionalIDs := []int64{buy.ActivityId, transaction.ActivityId}
						matchedActivityIds = append(matchedActivityIds, additionalIDs...)
						capitalGainsBalance := float64(gain) / 100
						log.Printf("Found new capital gain/loss for stock ticker: %s for $%.2f", ticker, capitalGainsBalance)
						log.Println()
					} else {
						// Partial match
						gain = (transaction.StockPrice - buy.StockPrice) * int64(sharesToSell)
						tickerGainsMap[ticker] += gain
						buy.ShareCount -= sharesToSell
						sharesToSell = 0
					}
				}
			}
		}
		netChange += tickerGainsMap[ticker]
	}

	db.MatchTransactions(transactions[0].AccountId, matchedActivityIds)
	db.UpsertCapitalGainsBalance(transactions[0].AccountId, transactions[0].ActivityDate.Year(), netChange, 0)
	return netChange
}

// containsSellOrder returns true if list of newly received orders contains any sell orders
func containsSellOrder(orders []JsonParser.Order) bool {
	for _, order := range orders {
		if order.OrderLegCollection[0].Instruction == "SELL" {
			return true
		}
	}
	return false
}
