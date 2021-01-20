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
	if len(args) != 2 {
		showUsage(1)
	}

	config := mqtt.MQTTConfig{
		Brokers:  []string{*server},
		ClientID: "mqtt-response",
		Version:  mqtt.V5,
	}

	client, err := config.CreateConnection()
	if err != nil {
		log.Fatal(err)
	}

	topic, msg := args[0], []byte(args[1])

	err = client.SubscribeResponse(topic)
	if err != nil {
		log.Fatal(err)
	}

	client.HandleRequest(func(responseTopic string, payload []byte, id []byte) {
		log.Printf("Received on [%s]: '%s'\n", responseTopic, payload)

		client.Respond(responseTopic, msg, id)
	})

	log.Printf("Waiting for request [%s]", topic)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	client.Disconnect()

}

func usage() {
	log.Printf("Usage: response [-s server] <topic> <msg>\n")
	flag.PrintDefaults()
}

func showUsage(ec int) {
	usage()
	os.Exit(ec)
}
