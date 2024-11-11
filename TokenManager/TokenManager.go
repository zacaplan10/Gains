package TokenManager

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type TokenManager struct {
	AppKey       string
	AppSecret    string
	BearerToken  string
	RefreshToken string
}

func NewTokenManager(appKey, appSecret string) *TokenManager {
	return &TokenManager{
		AppKey:    appKey,
		AppSecret: appSecret,
	}
}

// ConstructInitAuthURL generates the authorization URL and returns credentials and the URL.
func (tm *TokenManager) ConstructInitAuthURL() string {
	authURL := fmt.Sprintf("https://api.schwabapi.com/v1/oauth/authorize?client_id=%s&redirect_uri=https://127.0.0.1", tm.AppKey)

	log.Println("Click to authenticate:")
	log.Println(authURL)

	return authURL
}

func ConstructHeaders(appKey, appSecret string) map[string]string {
	// Encode the client credentials
	credentials := appKey + ":" + appSecret
	base64Credentials := base64.StdEncoding.EncodeToString([]byte(credentials))

	// Set headers
	headers := map[string]string{
		"Authorization": "Basic " + base64Credentials,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	return headers
}

func ConstructPayload(data map[string]string) url.Values {
	payload := url.Values{}
	for key, value := range data {
		payload.Set(key, value)
	}
	return payload
}

// ConstructHeadersAndPayload builds the headers and payload for the token request.
func ConstructHeadersAndPayload(returnedURL, appKey, appSecret string) (map[string]string, url.Values) {
	// Extract the authorization code from the URL
	codeIndex := strings.Index(returnedURL, "code=") + 5
	endIndex := strings.Index(returnedURL, "%40")
	responseCode := returnedURL[codeIndex:endIndex]
	responseCode = responseCode + "@"
	log.Println(responseCode)
	payloadData := map[string]string{
		"grant_type":   "authorization_code",
		"code":         responseCode,
		"redirect_uri": "https://127.0.0.1",
	}

	headers := ConstructHeaders(appKey, appSecret)
	payload := ConstructPayload(payloadData)

	return headers, payload
}

// RetrieveTokens makes the token request and returns the tokens as a map.
func RetrieveTokens(headers map[string]string, payload url.Values) (map[string]interface{}, error) {
	req, err := http.NewRequest("POST", "https://api.schwabapi.com/v1/oauth/token", strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokens map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (tm *TokenManager) GetAuthTokens() {
	authURL := tm.ConstructInitAuthURL()
	fmt.Println("Open this URL in your browser:", authURL)

	// Prompt for returned URL
	fmt.Print("Paste Returned URL: ")
	var returnedURL string
	fmt.Fscanln(os.Stdin, &returnedURL)

	headers, payload := ConstructHeadersAndPayload(returnedURL, tm.AppKey, tm.AppSecret)
	tokens, err := RetrieveTokens(headers, payload)
	if err != nil {
		log.Fatalf("Error retrieving tokens: %v", err)
	}
	var bearerToken string
	var refreshToken string
	var ok bool
	if bearerToken, ok = tokens["access_token"].(string); !ok {
		log.Fatalf("Error parsing bearer token: %v", err)
	}
	if refreshToken, ok = tokens["refresh_token"].(string); !ok {
		log.Fatalf("Error parsing refresh token: %v", err)
	}

	tm.BearerToken = bearerToken
	tm.RefreshToken = refreshToken
}

func (tm *TokenManager) SetAuthTokens(bearerToken string, refreshToken string) {
	tm.BearerToken = bearerToken
	tm.RefreshToken = refreshToken
}

func (tm *TokenManager) RefreshTokens() error {
	payloadData := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": tm.RefreshToken,
	}
	payload := ConstructPayload(payloadData)
	headers := ConstructHeaders(tm.AppKey, tm.AppSecret)

	req, err := http.NewRequest("POST", "https://api.schwabapi.com/v1/oauth/token", strings.NewReader(payload.Encode()))
	if err != nil {
		slog.Error("Error creating new request:", err)
		return err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error executing request:", err)
		return err
	}
	defer resp.Body.Close()

	var tokens map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		slog.Error("Error parsing response body:", err)
		return err
	}

	var bearerToken string
	var ok bool
	if bearerToken, ok = tokens["access_token"].(string); ok {
		tm.BearerToken = bearerToken
		return nil
	} else {
		slog.Error("Error parsing bearer token")
		return err
	}
}
