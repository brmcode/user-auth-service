package config

import (
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

func InitOAuth(config *OAuth) {
	var providers []goth.Provider

	if config.GoogleClientID != "" && config.GoogleClientSecret != "" {
		providers = append(providers, google.New(
			config.GoogleClientID,
			config.GoogleClientSecret,
			config.GoogleCallbackURL,
			"email", "profile",
		))
	}

	if config.GithubClientID != "" && config.GithubClientSecret != "" {
		providers = append(providers, github.New(
			config.GithubClientID,
			config.GithubClientSecret,
			config.GithubCallbackURL,
			"user:email",
		))
	}

	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}

}
