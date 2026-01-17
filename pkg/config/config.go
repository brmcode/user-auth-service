package config

import (
	"time"

	"github.com/spf13/viper"
)

type (
	Configuration struct {
		DB    *DB
		HTTP  *HTTP
		Auth  *Auth
		Redis *Redis
		OAuth *OAuth
	}

	DB struct {
		Connection string
		Host       string
		Port       string
		User       string
		Password   string
		Name       string
	}

	HTTP struct {
		Port string
	}

	Auth struct {
		SecretKey            string
		TokenType            string
		TokenDuration        time.Duration
		RefreshTokenDuration time.Duration
	}

	Redis struct {
		Addr     string
		Password string
		TTL      time.Duration
	}
	OAuth struct {
		GoogleClientID     string
		GoogleClientSecret string
		GoogleCallbackURL  string
		GithubClientID     string
		GithubClientSecret string
		GithubCallbackURL  string
	}
)

func New(path string) (config *Configuration, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()

	db := &DB{
		Connection: viper.GetString("DB_CONNECTION"),
		Host:       viper.GetString("DB_HOST"),
		Port:       viper.GetString("DB_PORT"),
		User:       viper.GetString("DB_USER"),
		Password:   viper.GetString("DB_PASSWORD"),
		Name:       viper.GetString("DB_NAME"),
	}

	http := &HTTP{
		Port: viper.GetString("HTTP_PORT"),
	}
	auth := &Auth{
		SecretKey:            viper.GetString("SECRET_KEY"),
		TokenType:            viper.GetString("TOKEN_TYPE"),
		TokenDuration:        viper.GetDuration("TOKEN_DURATION"),
		RefreshTokenDuration: viper.GetDuration("REFRESH_TOKEN_DURATION"),
	}

	redis := &Redis{
		Addr:     viper.GetString("REDIS_ADDR"),
		Password: viper.GetString("REDIS_PASSWORD"),
		TTL:      viper.GetDuration("REDIS_TTL"),
	}

	oAuth := &OAuth{
		GoogleClientID:     viper.GetString("OAUTH_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: viper.GetString("OAUTH_GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:  viper.GetString("OAUTH_GOOGLE_CALLBACK_URL"),
		GithubClientID:     viper.GetString("OAUTH_GITHUB_CLIENT_ID"),
		GithubClientSecret: viper.GetString("OAUTH_GITHUB_CLIENT_SECRET"),
		GithubCallbackURL:  viper.GetString("OAUTH_GITHUB_CALLBACK_URL"),
	}

	config = &Configuration{DB: db, HTTP: http, Auth: auth, Redis: redis, OAuth: oAuth}
	return
}
