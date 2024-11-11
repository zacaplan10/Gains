package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"net/url"
	"os"
	"strings"
)

// App struct
type App struct {
	ctx context.Context
	db  *DatabaseHelper
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.Println("This is a log message.")
	a.db, _ = NewDatabaseHelperFromConnectionString("postgres://postgres:Kasbo143@localhost:5432/postgres")
}

// GetCapitalGainsBalance returns a greeting for the given name
func (a *App) GetCapitalGainsBalance(accountId string) string {
	capitalGainsBalance := float64(GetShortTermCapitalGainsBalance(accountId, a.db)) / 100
	if capitalGainsBalance >= 0 {
		return fmt.Sprintf("Account %s has $%.2f in lifetime capital gains", accountId, capitalGainsBalance)
	}
	return fmt.Sprintf("Account %s has $%.2f in usable capital losses", accountId, capitalGainsBalance)
}

func GetShortTermCapitalGainsBalance(accountNumber string, db *DatabaseHelper) int64 {
	var netCapitalChange int64
	//accountId, _ := strconv.ParseInt(accountNumber, 10, 64)
	accountId := 12345678
	query := `SELECT
    SUM(
            CASE
                WHEN net_capital_change < 0 THEN net_capital_change + carryover_loss
                ELSE net_capital_change - LEAST(net_capital_change, carryover_loss)
                END
    ) AS grand_total
FROM
    capital_gains_balance
WHERE account_id = $1`
	err := db.Database.QueryRow(query, accountId).Scan(&netCapitalChange)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal("Query failed: ", err)
	}
	return netCapitalChange
}

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
		log.Fatalf("failed to parse connection string: %v", err)
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
		log.Fatalf("failed to open database connection: %v", err)
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
