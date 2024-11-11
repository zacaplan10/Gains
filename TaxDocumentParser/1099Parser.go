package TaxDocumentParser

import (
	"encoding/csv"
	"fmt"
	"gains/Data"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func Import1099(accountId int, db *Data.DatabaseHelper) {
	// Open the CSV file
	file, err := os.Open("C:/Users/zacap/output.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Parse the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize grand total for tax year, and the tax year
	var grandTotal float64
	var taxYear int

	// Iterate over rows and calculate short-term capital gains/losses
	for i, record := range records {
		// Skip header
		if i == 0 {
			continue
		}

		// Parse acquisition and sold dates
		acquiredDate, err := time.Parse("01/02/06", record[4]) // Adjust index if necessary
		if err != nil {
			log.Fatalf("Error parsing acquired_date on row %d: %v", i, err)
		}
		soldDate, err := time.Parse("01/02/06", record[1]) // Adjust index if necessary
		if err != nil {
			log.Fatalf("Error parsing sold_date on row %d: %v", i, err)
		}
		if taxYear == 0 {
			taxYear = acquiredDate.Year()
		}

		// Calculate holding period in days
		holdingPeriod := soldDate.Sub(acquiredDate).Hours() / 24

		// Parse proceeds and cost
		var proceeds float64
		proceeds, err = strconv.ParseFloat(record[7], 64)
		if err != nil {
			proceeds, err = parseAmount(record[7])
			if err != nil {
				log.Fatalf("Error parsing proceeds on row %d: %v", i, err)
			}
		}

		// Calculate gain or loss
		gainOrLoss := proceeds
		if holdingPeriod <= 365 {
			fmt.Printf("Row %d: Short-term gain/loss: $%.2f\n", i, gainOrLoss)
			grandTotal += gainOrLoss
		}
	}

	// Print the grand total
	fmt.Printf("Grand Total Short-term Gain/Loss: $%.2f\n", grandTotal)
	db.UpsertCapitalGainsBalance(accountId, taxYear, int64(grandTotal*100), 0)
}

// Remove commas from the string and return the cleaned string as a float
func parseAmount(amountStr string) (float64, error) {

	cleaned := strings.ReplaceAll(amountStr, ",", "")
	return strconv.ParseFloat(cleaned, 64)
}
