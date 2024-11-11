package JsonParser

import (
	"encoding/json"
	"strconv"
)

type Account struct {
	AccountNumber int    `json:"-"`
	HashValue     string `json:"hashValue"`
}

func ParseAccounts(data []byte) (Account, error) {
	var accounts Account
	err := accounts.UnmarshalJSON(data)
	if err != nil {
		return accounts, err
	}

	return accounts, nil
}

func (a *Account) UnmarshalJSON(data []byte) error {
	var aux []struct {
		AccountNumber string `json:"accountNumber"`
		HashValue     string `json:"hashValue"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	accountNumber, err := strconv.Atoi(aux[0].AccountNumber)
	if err != nil {
		return err
	}

	a.AccountNumber = accountNumber
	a.HashValue = aux[0].HashValue
	return nil
}
