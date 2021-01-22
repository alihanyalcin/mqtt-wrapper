[![GoDoc](https://godoc.org/github.com/alihanyalcin/mqtt-wrapper?status.svg)](https://godoc.org/github.com/alihanyalcin/mqtt-wrapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/alihanyalcin/mqtt-wrapper)](https://goreportcard.com/report/github.com/alihanyalcin/mqtt-wrapper)
# MQTT-Wrapper
MQTT-Wrapper package provides easy-to-use **MQTTv3** and **MQTTv5** connection for Go projects.

It supports **Request/Response** pattern for **MQTTv5** connection.

# Usage

* Import the package 
```go
import( 
    mqtt "github.com/alihanyalcin/mqtt-wrapper"
)
```
* Create a local variable using MQTTConfig struct and fill the necessary fields for the connection.
```go
config := mqtt.Config{
    Brokers:              []string{"127.0.0.1:1883"},
    ClientID:             "",
    Username:             "",
    Password:             "",
    Topics:               []string{"topic"},
    QoS:                  0,
    Retained:             false,
    AutoReconnect:        false,
    MaxReconnectInterval: 5,
    PersistentSession:    false,
    KeepAlive:            15,
    TLSCA:                "",
    TLSCert:              "",
    TLSKey:               "",
    Version:              mqtt.V3, // use mqtt.V5 for MQTTv5 client.
}
```
* Then, create a connection to MQTT broker(s).
````go
client, err := config.CreateConnection()
if err != nil {
    log.Fatal(err)
}
````
* To disconnect from broker(s)
````go
client.Disconnect()
````
* To publish a message to broker(s) with a specific topic
````go
err = client.Publish("topic", "message")
if err != nil {
    log.Fatal(err)
}
````
* To handle new messages
````go
client.Handle(func(topic string, payload []byte) {
    log.Printf("Received on [%s]: '%s'", topic, string(payload))
})
````

Check [examples](https://github.com/alihanyalcin/mqtt-wrapper/tree/master/examples) for more information about MQTTv5 client.
