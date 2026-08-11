package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	GoogleMapsAPIKey string
	GoogleMapsAPIUrl string
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword +
		"@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName +
		"?parseTime=true&charset=utf8mb4"
}

func Load() *Config {
	for _, path := range []string{".env", "../.env", "../../.env"} {
		if _, err := os.Stat(path); err == nil {
			return LoadFromFile(path)
		}
	}
	return LoadFromFile("")
}

func LoadFromFile(path string) *Config {
	fileValues := readEnvFile(path)

	cfg := &Config{
		Port:             get("PORT", fileValues, "8080"),
		DBHost:           get("DB_HOST", fileValues, "localhost"),
		DBPort:           get("DB_PORT", fileValues, "3306"),
		DBUser:           get("DB_USER", fileValues, "root"),
		DBPassword:       get("DB_PASSWORD", fileValues, "root"),
		DBName:           get("DB_NAME", fileValues, "delivery_db"),
		GoogleMapsAPIKey: get("GOOGLE_MAPS_API_KEY", fileValues, ""),
		GoogleMapsAPIUrl: get("GOOGLE_MAPS_API_URL", fileValues, "https://maps.googleapis.com/maps/api/distancematrix/json"),
	}

	return cfg
}

func get(key string, fileValues map[string]string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v, ok := fileValues[key]; ok && v != "" {
		return v
	}
	return fallback
}

func readEnvFile(path string) map[string]string {
	values := make(map[string]string)

	f, err := os.Open(path)
	if err != nil {
		return values
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if key != "" {
			values[key] = val
		}
	}

	return values
}
