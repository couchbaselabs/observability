// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metacfg

import (
	"fmt"
	"regexp"

	"github.com/creasty/defaults"
	val "github.com/go-playground/validator/v10"
)

type Validator struct {
	validator *val.Validate
}

var cfgValidator = new(Validator)

// ValidateWithDefaults applies the default values for any missing fields in the config,
// then validates them all. Returns an error if there are any validation failures.
func (v *Validator) ValidateWithDefaults(cfg *Config) error {
	if err := defaults.Set(cfg); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}

	if v.validator == nil {
		validator := val.New()
		v.registerCustomValidations(validator)
		v.validator = validator
	}

	return v.validator.Struct(cfg)
}

var promLabelRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (v *Validator) registerCustomValidations(validate *val.Validate) {
	if err := validate.RegisterValidation("prometheus_label", func(fl val.FieldLevel) bool {
		return promLabelRegex.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Errorf("failed to register validation prometheus_label: %w", err))
	}
}
