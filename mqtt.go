package mqtt_wrapper

import (
	"errors"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"time"
)

type ConnectionState int
const (
	Disconnected ConnectionState = iota
	Connected
)

type MQTTConfig struct {
	Brokers []string
	ClientID string
	Username string
	Password string
	Topics []string
	QoS int

	client MQTT.Client
	options *MQTT.ClientOptions
	state ConnectionState
}

func (m *MQTTConfig) createConnection() error {
	m.state =  Disconnected

	if len(m.Brokers) == 0 {
		return errors.New("no broker address to connect")
	}

	if m.QoS > 2 || m.QoS < 0 {
		return errors.New("value of qos must be 0, 1, 2")
	}

	var err error
	m.options, err = m.createOptions()
	if err != nil {
		return err
	}

	err = m.connect()
	if err != nil {
		return err
	}

	return nil
}

func (m *MQTTConfig) connect() error {
	m.client = MQTT.NewClient(m.options)

	token := m.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.state = Connected

	if len(m.Topics) != 0 {
		var topics map[string]byte
		for _, topic := range m.Topics {
			topics[topic] = byte(m.QoS)
		}
		subscribeToken := m.client.SubscribeMultiple(topics, m.onMessageReceived)
		if subscribeToken.Wait() && subscribeToken.Error() != nil {
			return subscribeToken.Error()
		}
	}

	return nil
}

func (m *MQTTConfig) createOptions() (*MQTT.ClientOptions, error) {
	options := MQTT.NewClientOptions()

	for _, broker := range m.Brokers {
		options.AddBroker(broker)
	}

	if m.ClientID == "" {
		m.ClientID = "mqtt-client"
	}
	options.SetClientID(m.ClientID)

	if m.Username != "" {
		options.SetUsername(m.Username)
	}

	if m.Password != "" {
		options.SetPassword(m.Password)
	}

	options.SetAutoReconnect(false)
	options.SetKeepAlive(time.Second * 60)
	options.SetCleanSession(true)
	options.SetConnectionLostHandler(m.onConnectionLost)

	return options, nil
}

func (m *MQTTConfig) onConnectionLost(c MQTT.Client, err error) {
	m.state = Disconnected
}

func (m *MQTTConfig) onMessageReceived(c MQTT.Client, msg MQTT.Message) {

}
