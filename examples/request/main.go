package main

import (
	"flag"
	"log"
	"os"
	"time"

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
	if len(args) != 2 {
		showUsage(1)
	}

	config := mqtt.Config{
		Brokers:  []string{*server},
		ClientID: "mqtt-request",
		Version:  mqtt.V5,
	}

	client, err := config.CreateConnection()
	if err != nil {
		log.Fatal(err)
	}

	topic, msg := args[0], []byte(args[1])

	log.Printf("Request published [%s] : '%s'", topic, msg)
	err = client.Request(topic, msg, 5*time.Second, func(respTopic string, payload []byte) {
		log.Printf("Received  [%v] : '%s'", respTopic, payload)
	})
	if err != nil {
		log.Fatal(err)
	}

	client.Disconnect()
}

func usage() {
	log.Printf("Usage: request [-s server] <topic> <msg>\n")
	flag.PrintDefaults()
}

func showUsage(ec int) {
	usage()
	os.Exit(ec)
}
