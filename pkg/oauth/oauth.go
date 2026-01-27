package oauth

import (
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

func Init(config *config.OAuth) {
	var providers []goth.Provider

	if config.GoogleClientID != "" && config.GoogleClientSecret != "" {
		providers = append(providers, google.New(
			config.GoogleClientID,
			config.GoogleClientSecret,
			config.GoogleCallbackURL,
			"email", "profile",
		))
	}

	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}
