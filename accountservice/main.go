package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/weriK/goblog/accountservice/dbclient"
	"github.com/weriK/goblog/accountservice/service"
	"github.com/weriK/goblog/common/config"
	"github.com/weriK/goblog/common/messaging"
)

var appName = "accountservice"

func init() {
	profile := flag.String("profile", "test", "Environment profile")
	if *profile == "dev" {
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000",
			FullTimestamp:   true,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	configServerUrl := flag.String("configServerUrl", "http://configserver:8888", "Address to config server")
	configBranch := flag.String("configBranch", "master", "git branch to fetch configuration from")
	flag.Parse()

	viper.Set("profile", *profile)
	viper.Set("configServerUrl", *configServerUrl)
	viper.Set("configBranch", *configBranch)
}

func main() {
	logrus.Infof("Starting %v\n", appName)

	config.LoadConfigurationFromBranch(
		viper.GetString("configServerUrl"),
		appName,
		viper.GetString("profile"),
		viper.GetString("configBranch"),
	)

	initializeMessaging()

	initializeBoltClient()

	go config.StartListener(appName, viper.GetString("amqp_server_url"), viper.GetString("config_event_bus"))

	service.StartWebServer(viper.GetString("server_port"))
}

func initializeMessaging() {
	if !viper.IsSet("amqp_server_url") {
		panic("No 'amqp_server_url` set in configuration, cannot start")
	}

	service.MessagingClient = &messaging.MessagingClient{}
	service.MessagingClient.ConnectToBroker(viper.GetString("amqp_server_url"))
	service.MessagingClient.Subscribe(viper.GetString("config_event_bus"), "topic", appName, config.HandleRefreshEvent)
}

func initializeBoltClient() {
	service.DBClient = &dbclient.BoltClient{}
	service.DBClient.OpenBoltDb()
	service.DBClient.Seed()
}
