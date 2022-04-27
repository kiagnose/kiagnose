package config_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatencycheck/internal/config"
)

func TestCreateConfigFromEnvShould(t *testing.T) {
	testEnv := map[string]string{
		config.ResultsConfigMapNameEnvVarName:          "results",
		config.ResultsConfigMapNamespaceEnvVarName:     "default",
		config.NetworkNameEnvVarName:                   "blue-net",
		config.NetworkNamespaceEnvVarName:              "default",
		config.SampleDurationSecondsEnvVarName:         "",
		config.DesiredMaxLatencyMillisecondsEnvVarName: "",
	}

	t.Run("succeed", func(t *testing.T) {
		testConfig, err := config.NewFromEnv(testEnv)
		assert.NotNil(t, testConfig)
		assert.NoError(t, err)
	})

	t.Run("set default sample duration when env var is missing", func(t *testing.T) {
		testConfig, err := config.NewFromEnv(testEnv)
		assert.NotNil(t, testConfig)
		assert.NoError(t, err)
		assert.Equal(t, testConfig.SampleDurationSeconds, config.DefaultSampleDurationSeconds)
	})

	t.Run("set default default max latency when env var is missing", func(t *testing.T) {
		testConfig, err := config.NewFromEnv(testEnv)
		assert.NotNil(t, testConfig)
		assert.NoError(t, err)
		assert.Equal(t, testConfig.DesiredMaxLatencyMilliseconds, config.DefaultDesiredMaxLatencyMilliseconds)
	})
}

func TestCreateConfigFromEnvShouldFailWhen(t *testing.T) {
	type configCreateTestCases struct {
		description   string
		expectedError error
		env           map[string]string
	}
	testCases := []configCreateTestCases{
		{
			"env is nil",
			config.ErrInvalidEnv,
			nil,
		},
		{
			"results ConfigMap name env var is missing",
			config.ErrResultsConfigMapNameMissing,
			map[string]string{
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"results ConfigMap name env var value is not valid",
			config.ErrInvalidResultsConfigMapName,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"results ConfigMap namespace env var is missing",
			config.ErrResultsConfigMapNamespaceMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName: "results",
				config.NetworkNameEnvVarName:          "blue-net",
				config.NetworkNamespaceEnvVarName:     "default",
			},
		},
		{
			"results ConfigMap namespace env var value is not valid",
			config.ErrInvalidResultsConfigMapNamespace,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network name env var is missing",
			config.ErrNetworkNameMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network name env var value is not valid",
			config.ErrInvalidNetworkName,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network namespace env var is missing",
			config.ErrNetworkNamespaceMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
			},
		},
		{
			"network namespace env var value is not valid",
			config.ErrInvalidNetworkNamespace,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testConfig, err := config.NewFromEnv(testCase.env)
			assert.Nil(t, testConfig)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}
