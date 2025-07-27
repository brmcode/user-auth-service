package validator

var ShortMessages = map[string]string{
	"required": "is required",

	// String field messages
	"min": "must be at least %s characters",
	"max": "must be at most %s characters",
	"len": "must be exactly %s characters",

	// Number field messages
	"min_number": "must be at least %s",
	"max_number": "must be at most %s",
	"len_number": "must be exactly %s",

	"oneof": "must be one of [%s]",

	"email":    "must be a valid email",
	"alphanum": "must contain only letters and numbers",

	"gte": "must be at least %s",
	"lte": "must be at most %s",
	"gt":  "must be greater than %s",
	"lt":  "must be less than %s",

	"eqfield": "must match %s",
	"nefield": "must not match %s",

	"uuid":     "must be a valid UUID",
	"datetime": "must be a valid date/time",
	"url":      "must be a valid URL",

	"boolean": "must be true or false",

	// Custom tag
	"hexlower":     "must be a valid hexadecimal string (lowercase)",
	"optional_url": "must be a valid URL",
	"date":         "must be a valid date in one of the formats: YYYY-MM-DD, DD/MM/YYYY, DD-MM-YYYY, or YYYY/MM/DD",
}
