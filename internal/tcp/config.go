package tcp

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env        EnvConfig
	Server     ServerConfig `json:"server"`
	ClientKeys ClientKeys   `json:"client_keys"`
}

type EnvConfig struct {
	TCPServerHost string
	TCPServerPort string
}

type ServerConfig struct {
	MaxConnections            int         `json:"max_connections"`
	MaxConnectionAttempts     int         `json:"max_connection_attempts"`
	MaxRetriesOnError         int         `json:"max_retries_on_error"`
	MaxMessageSize            int         `json:"max_message_size"`
	ConnectionAttemptDelay    Duration    `json:"connection_attempt_delay"`
	ConnectionLockoutDuration Duration    `json:"connection_lockout_duration"`
	HandshakeTimeout          Duration    `json:"handshake_timeout"`
	ReadTimeout               Duration    `json:"read_timeout"`
	WriteTimeout              Duration    `json:"write_timeout"`
	RateLimiter               RateLimiter `json:"rate_limiter"`
}

type RateLimiter struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	BurstSize         int  `json:"burst_size"`
}

type ClientKeys struct {
	OxrecyclerPublicKeyRSA string `json:"oxrecycler_public_key_rsa"`
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	str := string(b)
	str = str[1 : len(str)-1]
	duration, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", str)
	}
	*d = Duration(duration)
	return nil
}

func LoadConfig() (*Config, error) {
	//Load and Parse config.json
	configFile, err := os.Open("internal/tcp/config.json")
	if err != nil {
		return nil, fmt.Errorf("error opening config.json file: %v", err)
	}
	defer configFile.Close()

	var config Config
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config.json file: %v", err)
	}

	//Load .env variables
	var env EnvConfig
	if host := os.Getenv("SERVER_HOST"); host != "" {
		env.TCPServerHost = host
	} else {
		env.TCPServerHost = "localhost"
	}
	if port := os.Getenv("SERVER_PORT_TCP"); port != "" {
		env.TCPServerPort = port
	} else {
		env.TCPServerPort = ":3000"
	}
	config.Env = env
	return &config, nil
}
