package Endpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "gains/Data"
	"gains/Data/JsonParser"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SchwabAPI struct {
	BaseURL     string
	BearerToken string
	HttpClient  *http.Client
}

func NewSchwabAPI(bearerToken string) *SchwabAPI {
	return &SchwabAPI{
		BaseURL:     "https://api.schwabapi.com/trader/v1",
		BearerToken: bearerToken,
		HttpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// DoRequest is a generic method to interact with Schwab endpoints
func (api *SchwabAPI) DoRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", api.BaseURL, endpoint)
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+api.BearerToken)

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		err = ReportError(resp.StatusCode, resp.Status)
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// GetAllOrders requests the Schwab API to retrieve any orders in the last 6 months
func (api *SchwabAPI) GetAllOrders(hashedAccountId string) ([]byte, error) {
	var epochTime int64 = 1682705999

	// Convert the epoch time to time.Time
	startDateTimeUtc := time.Unix(epochTime, 0).UTC().Format("2006-01-02T15:04:05.000Z")
	currentDateTimeUtc := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	timeMap := map[string]string{
		"fromEnteredTime": startDateTimeUtc,
		"toEnteredTime":   currentDateTimeUtc,
	}
	return api.GetAllOrdersApi(hashedAccountId, timeMap)
}

// GetRecentOrders requests the Schwab API to retrieve any orders in the past 5 minutes
func (api *SchwabAPI) GetRecentOrders(hashedAccountId string) ([]byte, error) {
	startDateTimeUtc := time.Now().UTC().Add(-5 * time.Minute).Format("2006-01-02T15:04:05.000Z")
	currentDateTimeUtc := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	timeMap := map[string]string{
		"fromEnteredTime": startDateTimeUtc,
		"toEnteredTime":   currentDateTimeUtc,
	}
	return api.GetAllOrdersApi(hashedAccountId, timeMap)
}

// GetAllOrdersApi send the API request to retrieve any orders in a specific timeframe
func (api *SchwabAPI) GetAllOrdersApi(hashedAccountId string, params map[string]string) ([]byte, error) {
	endpoint := "/accounts/" + hashedAccountId + "/orders"
	query := url.Values{}
	for key, value := range params {
		query.Add(key, value)
	}

	if query.Encode() != "" {
		endpoint += "?" + query.Encode()
	}
	return api.DoRequest("GET", endpoint, nil)
}

// GetAccountNumbers send the API request to retrieve account numbers associated with the active bearer token
func (api *SchwabAPI) GetAccountNumbers() (JsonParser.Account, error) {
	endpoint := "/accounts/accountNumbers"
	response, err := api.DoRequest("GET", endpoint, nil)
	if err != nil {
		//slog.Error("Error retrieving account numbers: ", err)
		return JsonParser.Account{}, err
	}

	accounts, err := JsonParser.ParseAccounts(response)
	if err != nil {
		return JsonParser.Account{}, err
	}
	return accounts, nil
}

type ApiError struct {
	Code    int
	Message string
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

func ReportError(code int, message string) error {
	return &ApiError{
		Code:    code,
		Message: message,
	}
}
