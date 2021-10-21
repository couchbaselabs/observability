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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type ConfigManager interface {
	Get() Config
	Set(config *Config) error
}

type EphemeralConfigManager struct {
	value *Config
	mux   sync.RWMutex
}

func (m *EphemeralConfigManager) Get() Config {
	m.mux.RLock()
	val := *m.value
	m.mux.RUnlock()
	return val
}

func (m *EphemeralConfigManager) Set(val *Config) error {
	m.mux.Lock()
	m.value = val
	m.mux.Unlock()
	return nil
}

func ReadConfigFromFile(filePath string, readOnly bool, allowDefault bool) (ConfigManager, error) {
	if !readOnly {
		return nil, fmt.Errorf("non-read-only config not supported yet")
	}
	var initialValue *Config
	cfgFile, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if allowDefault {
				initialValue = NewDefault()
			} else {
				return nil, fmt.Errorf("config file doesn't exist and default forbidden: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open configuration file: %w", err)
		}
	} else {
		defer cfgFile.Close()
		data, err := io.ReadAll(cfgFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read from configuration file: %w", err)
		}
		initialValue, err = FromYAMLValidate(data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration file: %w", err)
		}
	}
	if readOnly {
		return ReadOnlyConfigManager{value: initialValue}, nil
	} else {
		return nil, nil // Can't happen
	}
}

type ReadOnlyConfigManager struct {
	value *Config
}

func (f ReadOnlyConfigManager) Get() Config {
	return *f.value
}

func (f ReadOnlyConfigManager) Set(_ *Config) error {
	return fmt.Errorf("cannot set config for %T", f)
}
