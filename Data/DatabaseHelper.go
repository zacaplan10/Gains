package Data

import (
	"database/sql"
	"errors"
	"fmt"
	"gains/Data/JsonParser"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/lib/pq"
	"log"
	"net/url"
	"strings"
	"time"
)

type DatabaseHelper struct {
	ConnectionString string
	Username         string
	Password         string
	Host             string
	Port             string
	DatabaseName     string
	Database         *sql.DB
}

func NewDatabaseHelperFromConnectionString(connStr string) (*DatabaseHelper, error) {
	// Parse the connection string
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %v", err)
	}

	// Extract username and password
	user := u.User.Username()
	password, _ := u.User.Password()

	// Extract host and database name
	host := u.Hostname()
	databaseName := strings.TrimPrefix(u.Path, "/")

	// Open a connection to the database
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	// Create and return the DatabaseHelper instance
	return &DatabaseHelper{
		ConnectionString: connStr,
		Username:         user,
		Password:         password,
		Host:             host,
		DatabaseName:     databaseName,
		Database:         db,
	}, nil
}

func (db *DatabaseHelper) GetHashedAccountNumber(accountNumber int) string {
	var hashedAccountNumber string
	query := "SELECT hash_id FROM account_info WHERE account_id=$1"
	err := db.Database.QueryRow(query, accountNumber).Scan(&hashedAccountNumber)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal("Query failed: ", err)
	}
	return hashedAccountNumber
}

func (db *DatabaseHelper) InsertHashedAccountNumber(accountNumber int, hashedAccountNumber string) string {
	query := `
        INSERT INTO account_info (account_id, hash_id) 
        VALUES ($1, $2)
    `
	result, err := db.Database.Exec(query, accountNumber, hashedAccountNumber)
	if err != nil {
		log.Fatal("Query failed: ", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Inserted %d row(s) successfully.\n", rowsAffected)
	return hashedAccountNumber
}

func (db *DatabaseHelper) MatchTransactions(accountNumber int, matchedActivityIds []int64) int64 {
	query := `
        UPDATE transaction_history
        SET matched = true
        WHERE account_id = $1 AND activity_id = ANY($2)
    `
	result, err := db.Database.Exec(query, accountNumber, matchedActivityIds)
	if err != nil {
		log.Fatal("Query failed: ", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Updated %d row(s) successfully.\n", rowsAffected)
	return rowsAffected
}

func (db *DatabaseHelper) UpsertCapitalGainsBalance(accountId int, taxYear int, netCapitalChange int64, carryover int) {

	query := `SELECT upsertcapitalchangebalance($1, $2, $3, $4)`

	_, err := db.Database.Exec(query, accountId, taxYear, netCapitalChange, carryover)
	if err != nil {
		log.Fatal("Query failed: ", err)
	}
}

func (db *DatabaseHelper) GetCapitalGainsBalanceForYear(accountId int, taxYear int) int64 {

	var netCapitalChange int64
	query := "SELECT net_capital_change FROM capital_gains_balance WHERE account_id=$1 and tax_year=$2"
	err := db.Database.QueryRow(query, accountId, taxYear).Scan(&netCapitalChange)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal("Query failed: ", err)
	}
	return netCapitalChange
}

func (db *DatabaseHelper) InsertTransactionData(orders []JsonParser.Order) int64 {
	var rowsAffected int64
	for _, order := range orders {
		orderLeg := order.OrderLegCollection[0]
		if order.Status == "FILLED" && orderLeg.OrderLegType != "OPTION" {
			for _, activity := range order.OrderActivityCollection {
				activityId := activity.ActivityId
				activityType := orderLeg.Instruction
				shareCount := activity.Quantity
				stockPrice := int64(activity.ExecutionLegs[0].Price * 100)
				stockTicker := orderLeg.Instrument.Symbol
				activityDate := activity.ExecutionLegs[0].Time

				// Prepare the INSERT statement
				query := `
				INSERT INTO transaction_history (account_id, order_id, activity_id, stock_ticker, share_count, 
				                                 stock_price, order_type, activity_date, matched) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				`

				result, err := db.Database.Exec(query, order.AccountNumber, order.OrderId, activityId, stockTicker,
					int(shareCount), stockPrice, activityType, activityDate, false)
				if err != nil {
					//slog.Error("Query failed: ", err)
					return 0
				}
				rowInserted, err := result.RowsAffected()
				if err != nil {
				}
				rowsAffected += rowInserted
			}
		}
	}

	fmt.Printf("Inserted %d row(s) successfully.\n", rowsAffected)
	return rowsAffected
}

type TransactionData struct {
	AccountId    int
	OrderId      int64
	ActivityId   int64
	StockTicker  string
	ShareCount   int
	StockPrice   int64
	OrderType    string
	ActivityDate time.Time
	Matched      bool
}

func (db *DatabaseHelper) GetTransactionsByAccountID(accountId int) ([]TransactionData, error) {
	query := `
		SELECT account_id, order_id, activity_id, stock_ticker, share_count, stock_price, order_type, activity_date, matched 
		FROM transaction_history
		WHERE account_id = $1
	`

	rows, err := db.Database.Query(query, accountId)
	if err != nil {
		return nil, fmt.Errorf("error querying transactions: %w", err)
	}
	defer rows.Close()

	var transactions []TransactionData

	for rows.Next() {
		var transaction TransactionData

		err := rows.Scan(
			&transaction.AccountId,
			&transaction.OrderId,
			&transaction.ActivityId,
			&transaction.StockTicker,
			&transaction.ShareCount,
			&transaction.StockPrice,
			&transaction.OrderType,
			&transaction.ActivityDate,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return transactions, nil
}

func (db *DatabaseHelper) GetUnmatchedTransactionsByAccountID(accountId int) ([]TransactionData, error) {
	query := `
		SELECT account_id, order_id, activity_id, stock_ticker, share_count, stock_price, order_type, activity_date, matched
		FROM transaction_history
		WHERE account_id = $1 and matched = false
	`

	rows, err := db.Database.Query(query, accountId)
	if err != nil {
		return nil, fmt.Errorf("error querying transactions: %w", err)
	}
	defer rows.Close()

	var transactions []TransactionData

	for rows.Next() {
		var transaction TransactionData

		err := rows.Scan(
			&transaction.AccountId,
			&transaction.OrderId,
			&transaction.ActivityId,
			&transaction.StockTicker,
			&transaction.ShareCount,
			&transaction.StockPrice,
			&transaction.OrderType,
			&transaction.ActivityDate,
			&transaction.Matched,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return transactions, nil
}
