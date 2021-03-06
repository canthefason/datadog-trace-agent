package config

import (
	"os"
	"strings"

	"github.com/stretchr/testify/assert"

	"testing"

	"gopkg.in/ini.v1"
)

func TestGetStrArray(t *testing.T) {
	assert := assert.New(t)
	f, _ := ini.Load([]byte("[Main]\n\nports = 10,15,20,25"))
	conf := File{
		f,
		"some/path",
	}

	ports, err := conf.GetStrArray("Main", "ports", ",")
	assert.Nil(err)
	assert.Equal(ports, []string{"10", "15", "20", "25"})
}

func TestDefaultConfig(t *testing.T) {
	assert := assert.New(t)
	agentConfig := NewDefaultAgentConfig()

	// assert that some sane defaults are set
	assert.Equal(agentConfig.ReceiverHost, "localhost")
	assert.Equal(agentConfig.ReceiverPort, 7777)

	assert.Equal(agentConfig.StatsdHost, "localhost")
	assert.Equal(agentConfig.StatsdPort, 8125)

	assert.Equal(agentConfig.LogLevel, "INFO")
}

func TestOnlyEnvConfig(t *testing.T) {
	// setting an API Key should be enough to generate valid config
	os.Setenv("DD_API_KEY", "apikey_from_env")

	agentConfig, _ := NewAgentConfig(nil, nil)
	assert.Equal(t, []string{"apikey_from_env"}, agentConfig.APIKeys)

	os.Setenv("DD_API_KEY", "")
}

func TestOnlyDDAgentConfig(t *testing.T) {
	assert := assert.New(t)

	// absent an override by legacy config, reading from dd-agent config should do the right thing
	ddAgentConf, _ := ini.Load([]byte(strings.Join([]string{
		"[Main]",
		"hostname = thing",
		"api_key = apikey_12",
		"bind_host = 0.0.0.0",
		"dogstatsd_port = 28125",
		"log_level = DEBUG",
	}, "\n")))
	configFile := &File{instance: ddAgentConf, Path: "whatever"}
	agentConfig, _ := NewAgentConfig(configFile, nil)

	assert.Equal("thing", agentConfig.HostName)
	assert.Equal([]string{"apikey_12"}, agentConfig.APIKeys)
	assert.Equal("0.0.0.0", agentConfig.ReceiverHost)
	assert.Equal(28125, agentConfig.StatsdPort)
	assert.Equal("DEBUG", agentConfig.LogLevel)
}

func TestDDAgentMultiAPIKeys(t *testing.T) {
	assert := assert.New(t)
	ddAgentConf, _ := ini.Load([]byte("[Main]\n\napi_key=foo, bar "))
	configFile := &File{instance: ddAgentConf, Path: "whatever"}

	agentConfig, _ := NewAgentConfig(configFile, nil)
	assert.Equal([]string{"foo", "bar"}, agentConfig.APIKeys)
}

func TestDDAgentConfigWithLegacy(t *testing.T) {
	assert := assert.New(t)

	defaultConfig := NewDefaultAgentConfig()

	// check that legacy conf file overrides dd-agent.conf
	dd, _ := ini.Load([]byte("[Main]\n\nhostname=thing\napi_key=apikey_12"))
	legacy, _ := ini.Load([]byte(strings.Join([]string{
		"[trace.api]",
		"api_key = pommedapi",
		"endpoint = an_endpoint",
		"[trace.concentrator]",
		"extra_aggregators=resource,error",
		"[trace.sampler]",
		"extra_sample_rate=0.33",
	}, "\n")))

	conf := &File{instance: dd, Path: "whatever"}
	legacyConf := &File{instance: legacy, Path: "whatever"}

	agentConfig, _ := NewAgentConfig(conf, legacyConf)

	// Properly loaded attributes
	assert.Equal([]string{"pommedapi"}, agentConfig.APIKeys)
	assert.Equal([]string{"an_endpoint"}, agentConfig.APIEndpoints)
	assert.Equal([]string{"resource", "error"}, agentConfig.ExtraAggregators)
	assert.Equal(0.33, agentConfig.ExtraSampleRate)

	// Check some defaults
	assert.Equal(defaultConfig.BucketInterval, agentConfig.BucketInterval)
	assert.Equal(defaultConfig.StatsdHost, agentConfig.StatsdHost)
}

func TestDDAgentConfigWithNewOpts(t *testing.T) {
	assert := assert.New(t)
	// check that providing trace.* options in the dd-agent conf file works
	dd, _ := ini.Load([]byte(strings.Join([]string{
		"[Main]",
		"hostname = thing",
		"api_key = apikey_12",
		"[trace.concentrator]",
		"extra_aggregators=resource,error",
		"[trace.sampler]",
		"extra_sample_rate=0.33",
	}, "\n")))

	conf := &File{instance: dd, Path: "whatever"}
	agentConfig, _ := NewAgentConfig(conf, nil)
	assert.Equal([]string{"resource", "error"}, agentConfig.ExtraAggregators)
	assert.Equal(0.33, agentConfig.ExtraSampleRate)
}
