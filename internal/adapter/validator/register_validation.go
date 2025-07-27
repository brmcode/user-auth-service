package validator

import (
	"net/url"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

func RegisterValidation(v *validator.Validate) {

	// Register custom validation tag "hexlower"
	v.RegisterValidation("hexlower", func(fl validator.FieldLevel) bool {
		code := fl.Field().String()
		match, _ := regexp.MatchString("^[a-f0-9]+$", code)
		return match
	})

	// Register custom validation tag "optional_url"
	v.RegisterValidation("optional_url", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		if val == "" {
			return true // allow empty
		}
		_, err := url.ParseRequestURI(val)
		return err == nil
	})
	// Register custom validation tag "date_supported"
	v.RegisterValidation("date", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		if val == "" {
			return false
		}
		layouts := []string{
			"2006-01-02", // ISO
			"02/01/2006", // Thai style D/M/Y
			"02-01-2006", // D-M-Y
			"2006/01/02", // Y/M/D
		}
		for _, layout := range layouts {
			if _, err := time.Parse(layout, val); err == nil {
				return true
			}
		}
		return false
	})
}
