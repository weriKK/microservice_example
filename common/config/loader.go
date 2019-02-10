package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Loads config from for example http://configserver:8888/accountservice/test/P8
func LoadConfigurationFromBranch(configServerUrl string, appName string, profile string, branch string) {
	url := fmt.Sprintf("%s/%s/%s/%s", configServerUrl, appName, profile, branch)
	logrus.Infof("Loading config from %s\n", url)
	body, err := fetchConfiguration(url)
	if err != nil {
		panic("Couldn't load configuration, cannot start. Terminating. Error: " + err.Error())
	}
	parseConfiguration(body)
}

// Make HTTP request to fetch configuration from config server
func fetchConfiguration(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		panic("Couldn't load configuration, cannot start. Terminating. Error: " + err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

// Pass JSON bytes into struct and then into Viper
func parseConfiguration(body []byte) {
	var cloudConfig springCloudConfig
	err := json.Unmarshal(body, &cloudConfig)
	if err != nil {
		panic("Cannot parse configuration, message: " + err.Error())
	}

	for key, value := range cloudConfig.PropertySources[0].Source {
		viper.Set(key, value)
		logrus.Infof("Loading config property %v => %v\n", key, value)
	}
	if viper.IsSet("server_name") {
		logrus.Infof("Successfully loaded configuration for service %s\n", viper.GetString("server_name"))
	}
}

// Structs having same structure as response from Spring Cloud Config
type springCloudConfig struct {
	Name            string           `json:"name"`
	Profiles        []string         `json:"profiles"`
	Label           string           `json:"label"`
	Version         string           `json:"version"`
	PropertySources []propertySource `json:"propertySources"`
}

type propertySource struct {
	Name   string                 `json:"name"`
	Source map[string]interface{} `json:"source"`
}
