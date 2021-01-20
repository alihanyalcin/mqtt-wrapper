package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	mqtt "github.com/alihanyalcin/mqtt-wrapper"
)

func main() {
	var server = flag.String("s", "127.0.0.1:1883", "The MQTT server URL")
	var help = flag.Bool("h", false, "Show help message")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *help {
		showUsage(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		showUsage(1)
	}

	topic := args[0]

	config := mqtt.MQTTConfig{
		Brokers:  []string{*server},
		ClientID: "mqtt-subscibe",
		Topics:   []string{topic},
		Version:  mqtt.V5,
	}

	client, err := config.CreateConnection()
	if err != nil {
		log.Fatal(err)
	}

	client.Handle(func(topic string, payload []byte) {
		log.Printf("Received on [%s]: '%s'", topic, string(payload))
	})

	log.Printf("Subscribed to [%s]", topic)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	client.Disconnect()
}

func usage() {
	log.Printf("Usage: subscibe [-s server] <topic>\n")
	flag.PrintDefaults()
}

func showUsage(ec int) {
	usage()
	os.Exit(ec)
}
