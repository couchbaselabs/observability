package metacfg

import (
	"fmt"
	"github.com/creasty/defaults"
	val "github.com/go-playground/validator/v10"
	"regexp"
)

type Validator struct {
	validator *val.Validate
}

// ValidateWithDefaults applies the default values for any missing fields in the config,
// then validates them all. Returns an error if there are any validation failures.
func (v *Validator) ValidateWithDefaults(cfg *Config) error {
	if err := defaults.Set(cfg); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}

	if v.validator == nil {
		val := val.New()
		v.registerCustomValidations(val)
		v.validator = val
	}

	err := v.validator.Struct(cfg)
	if err != nil {
		return err
	}
	return nil
}

var promLabelRegex = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)

func (v *Validator) registerCustomValidations(validate *val.Validate) {
	validate.RegisterValidation("prometheus_label", func(fl val.FieldLevel) bool {
		return promLabelRegex.MatchString(fl.Field().String())
	})
}
