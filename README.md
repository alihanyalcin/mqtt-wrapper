# MQTT-Wrapper
MQTT-Wrapper package provides easy-to-use MQTT connection for GO projects.

# Usage

* Import the package 
```go
import( 
    mqtt "github.com/alihanyalcin/mqtt-wrapper"
)
```
* Create a local variable using MQTTConfig struct and fill the necessary fields for the connection.
```go
m := mqtt.MQTTConfig{
		Brokers:              []string{"192.168.0.99:1883"},
		ClientID:             "mqtt-client",
		Username:             "",
		Password:             "",
		Topics:               []string{"topic"},
		QoS:                  0,
		AutoReconnect:        false,
		MaxReconnectInterval: 0,
		PersistentSession:    false,
		TLSCA:                "",
		TLSCert:              "",
		TLSKey:               "",
		Messages:             nil,
}
```
* Then, create a connection to MQTT broker(s).
````go
err := m.CreateConnection()
if err != nil {
    fmt.Println(err.Error())
}
````
* To disconnect from broker(s)
````go
m.Disconnect()
````
* To publish a message to broker(s) with a specific topic
````go
err := m.Publish("topic","message")
if err != nil {
	fmt.Println(err.Error())
}
````
* Any message comes from the subscribed topic will send into _**Message**_ channel
````go
msg := <- m.Messages
fmt.Println("Topic :",msg.Topic(), "Message :",string(msg.Payload()))
````

# References
Project | Link
------------ | -------------
Eclipse Paho MQTT Go client | https://github.com/eclipse/paho.mqtt.golang
Telegraf MQTT Consumer Input Plugin | https://github.com/influxdata/telegraf/tree/master/plugins/inputs/mqtt_consumer
EdgeX Device MQTT Go | https://github.com/edgexfoundry/device-mqtt-go
