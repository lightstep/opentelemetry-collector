// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestUnmarshalEmpty(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	_, err = unmarshal(confmap.New(), factories)
	assert.NoError(t, err)
}

func TestUnmarshalEmptyAllSections(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	conf := confmap.NewFromStringMap(map[string]interface{}{
		"receivers":  nil,
		"processors": nil,
		"exporters":  nil,
		"extensions": nil,
		"service":    nil,
	})
	cfg, err := unmarshal(conf, factories)
	assert.NoError(t, err)

	zapProdCfg := zap.NewProductionConfig()
	assert.Equal(t, telemetry.LogsConfig{
		Level:             zapProdCfg.Level.Level(),
		Development:       zapProdCfg.Development,
		Encoding:          "console",
		DisableCaller:     zapProdCfg.DisableCaller,
		DisableStacktrace: zapProdCfg.DisableStacktrace,
		OutputPaths:       zapProdCfg.OutputPaths,
		ErrorOutputPaths:  zapProdCfg.ErrorOutputPaths,
		InitialFields:     zapProdCfg.InitialFields,
	}, cfg.Service.Telemetry.Logs)
}

func TestUnmarshalUnknownTopLevel(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	conf := confmap.NewFromStringMap(map[string]interface{}{
		"unknown_section": nil,
	})
	_, err = unmarshal(conf, factories)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'' has invalid keys: unknown_section")
}

func TestConfigServicePipelineUnmarshalError(t *testing.T) {
	var testCases = []struct {
		// test case name (also file name containing config yaml)
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectError string
	}{
		{
			name: "duplicate-pipeline",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"traces/ pipe": nil,
				"traces /pipe": nil,
			}),
			expectError: "duplicate name",
		},
		{
			name: "invalid-pipeline-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"metrics/": nil,
			}),
			expectError: "in \"metrics/\" id: the part after / should not be empty",
		},
		{
			name: "invalid-pipeline-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"traces": map[string]interface{}{
					"unknown_section": nil,
				},
			}),
			expectError: "'[traces]' has invalid keys: unknown_section",
		},
		{
			name: "invalid-pipeline-sub-config",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"traces": "string",
			}),
			expectError: "'[traces]' expected a map, got 'string'",
		},
		{
			name: "invalid-pipeline-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"/metrics": nil,
			}),
			expectError: "in \"/metrics\" id: the part before / should not be empty",
		},
		{
			name: "invalid-sequence-value",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"traces": map[string]interface{}{
					"receivers": map[string]interface{}{
						"nop": map[string]interface{}{
							"some": "config",
						},
					},
				},
			}),
			expectError: "'[traces].receivers[0]' has invalid keys: nop",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			pipelines := make(map[config.ComponentID]ConfigServicePipeline)
			err := tt.conf.Unmarshal(&pipelines, confmap.WithErrorUnused())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestServiceUnmarshalError(t *testing.T) {
	var testCases = []struct {
		// test case name (also file name containing config yaml)
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectError string
	}{
		{
			name: "invalid-logs-level",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"telemetry": map[string]interface{}{
					"logs": map[string]interface{}{
						"level": "UNKNOWN",
					},
				},
			}),
			expectError: "error decoding 'telemetry.logs.level': unrecognized level: \"UNKNOWN\"",
		},
		{
			name: "invalid-metrics-level",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"telemetry": map[string]interface{}{
					"metrics": map[string]interface{}{
						"level": "unknown",
					},
				},
			}),
			expectError: "error decoding 'telemetry.metrics.level': unknown metrics level \"unknown\"",
		},
		{
			name: "invalid-service-extensions-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"extensions": []interface{}{
					map[string]interface{}{
						"nop": map[string]interface{}{
							"some": "config",
						},
					},
				},
			}),
			expectError: "'extensions[0]' has invalid keys: nop",
		},
		{
			name: "invalid-service-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"unknown_section": "string",
			}),
			expectError: "'' has invalid keys: unknown_section",
		},
		{
			name: "invalid-pipelines-config",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"pipelines": "string",
			}),
			expectError: "'pipelines' expected a map, got 'string'",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Unmarshal(&ConfigService{}, confmap.WithErrorUnused())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}
