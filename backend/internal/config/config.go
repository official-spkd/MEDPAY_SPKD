package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config menampung seluruh konfigurasi aplikasi yang dibaca sekali saat startup.
type Config struct {
	AppEnv             string
	AppPort            string
	CORSAllowedOrigins []string
	DB                 DBConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

// DSN membentuk connection string key=value untuk PostgreSQL.
// TimeZone sengaja tidak dikutip: driver gorm mem-parsing nilainya sendiri
// dan kutip akan ikut terkirim ke server.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		quoteDSN(c.Host), quoteDSN(c.Port), quoteDSN(c.User), quoteDSN(c.Password),
		quoteDSN(c.Name), quoteDSN(c.SSLMode), c.TimeZone,
	)
}

// quoteDSN membungkus nilai dengan kutip tunggal agar spasi, kutip, atau
// backslash pada password tidak merusak connection string.
func quoteDSN(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v) + "'"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load membaca .env (jika ada) lalu environment variable, dan memvalidasi
// variabel wajib. Tanpa file .env, nilai diambil dari environment OS.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("membaca file .env: %w", err)
	}

	var missing []string
	required := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            getEnv("APP_PORT", "8080"),
		CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		DB: DBConfig{
			Host:     required("DB_HOST"),
			Port:     required("DB_PORT"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		},
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("environment variable wajib belum diisi: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
