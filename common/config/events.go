package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

func StartListener(appName string, amqpServer string, exchangeName string) {
	err := NewConsumer(amqpServer, exchangeName, "topic", "config-event-queue", exchangeName, appName)
	if err != nil {
		logrus.Fatalf("%s", err)
	}

	logrus.Infof("running forever")
	select {} // Yet another way to stop a Goroutine from finishing...
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queue, key, ctag string) error {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	var err error

	logrus.Infof("dialing %s", amqpURI)
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return fmt.Errorf("Dial: %s", err)
	}

	logrus.Infof("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	logrus.Infof("got Channel, declaring Exchange (%s)", exchange)
	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}

	logrus.Infof("declared Exchange, declaring Queue (%s)", queue)
	state, err := c.channel.QueueDeclare(
		queue, // name of the queue
		false, // durable
		false, // delete when usused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("Queue Declare: %s", err)
	}

	logrus.Infof("declared Queue (%d messages, %d consumers), binding to Exchange (key '%s')",
		state.Messages, state.Consumers, key)

	if err = c.channel.QueueBind(
		queue,    // name of the queue
		key,      // bindingKey
		exchange, // sourceExchange
		false,    // noWait
		nil,      // arguments
	); err != nil {
		return fmt.Errorf("Queue Bind: %s", err)
	}

	logrus.Infof("Queue bound to Exchange, starting Consume (consumer tag '%s')", c.tag)
	deliveries, err := c.channel.Consume(
		queue, // name
		c.tag, // consumerTag,
		false, // noAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("Queue Consume: %s", err)
	}

	go handle(deliveries, c.done)

	return nil
}

func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	defer logrus.Infof("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		logrus.Infof(
			"got %dB consumer: [%v] delivery: [%v] routingkey: [%v] %s",
			len(d.Body),
			d.ConsumerTag,
			d.DeliveryTag,
			d.RoutingKey,
			d.Body,
		)
		HandleRefreshEvent(d)
		d.Ack(false)
	}
	logrus.Infof("handle: deliveries channel closed")
	done <- nil
}

func HandleRefreshEvent(d amqp.Delivery) {
	body := d.Body
	consumerTag := d.ConsumerTag
	updateToken := &UpdateToken{}
	err := json.Unmarshal(body, updateToken)
	if err != nil {
		logrus.Infof("Problem parsing UpdateToken: %v", err.Error())
	} else {
		if strings.Contains(updateToken.DestinationService, consumerTag) {
			logrus.Infoln("Reloading Viper config from Spring Cloud Config server")

			LoadConfigurationFromBranch(
				viper.GetString("configServerUrl"),
				consumerTag,
				viper.GetString("profile"),
				viper.GetString("configBranch"),
			)
		}
	}
}

type UpdateToken struct {
	Type               string `json:"type"`
	Timestamp          int    `json:"timestamp"`
	OriginService      string `json:"originService"`
	DestinationService string `json:"destinationService"`
	Id                 string `json:"id"`
}
