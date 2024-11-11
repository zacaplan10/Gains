package Properties

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	AppKey             string `json:"AppKey"`
	AppSecret          string `json:"AppSecret"`
	DBConnectionString string `json:"DBConnectionString"`
	BearerToken        string `json:"BearerToken"`
	RefreshToken       string `json:"RefreshToken"`
}

// LoadConfig reads the configuration from the JSON file
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig writes the configuration back to the JSON file
func SaveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func (c *Config) UpdateTokens(bearerToken, refreshToken string) error {
	c.BearerToken = bearerToken
	c.RefreshToken = refreshToken
	return SaveConfig("config.json", c)
}
