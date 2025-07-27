package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()

	RegisterValidation(v)

	return &Validator{validate: v}
}

func (v *Validator) Validate(s interface{}) error {
	sVal := reflect.ValueOf(s)
	sKind := sVal.Kind()

	if sKind == reflect.Slice || sKind == reflect.Array {
		for i := 0; i < sVal.Len(); i++ {
			item := sVal.Index(i).Interface()
			err := v.Validate(item)
			if err != nil {
				return fmt.Errorf("index %d: %w", i, err)
			}
		}
		return nil
	}

	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok || len(validationErrors) == 0 {
		return fmt.Errorf("validation failed")
	}

	// Return only the first error
	ve := validationErrors[0]
	fieldType := ve.Kind() // reflect.Kind, not string
	tag := ve.Tag()
	param := ve.Param()

	msgKey := tag
	if tag == "min" || tag == "max" || tag == "len" {
		switch fieldType {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			msgKey = tag + "_number"
		}
	}

	msgTemplate, ok := ShortMessages[msgKey]
	if !ok {
		msgTemplate = fmt.Sprintf("is not valid (%s)", tag)
	}

	var msg string
	if param != "" {
		msg = fmt.Sprintf(msgTemplate, param)
	} else {
		msg = msgTemplate
	}

	field := strings.ToLower(ve.Field())
	return fmt.Errorf("%s %s", field, msg)
}
