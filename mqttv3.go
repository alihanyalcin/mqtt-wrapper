// Package mqtt_wrapper provides easy-to-use MQTT connection for projects.
package mqtt_wrapper

import (
	"time"

	mqttv3 "github.com/eclipse/paho.mqtt.golang"
)

type MQTTv3 struct {
	client     mqttv3.Client
	state      ConnectionState
	config     MQTTConfig
	messages   chan mqttv3.Message
	disconnect chan bool
}

func newMQTTv3(config *MQTTConfig) (MQTT, error) {
	m := MQTTv3{
		state:      Disconnected,
		config:     *config,
		messages:   make(chan mqttv3.Message),
		disconnect: make(chan bool, 1),
	}

	err := m.connect()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *MQTTv3) Handle(f func(string, []byte)) {
	go func() {
		for {
			select {
			case <-m.disconnect:
				return
			case msg := <-m.messages:
				f(msg.Topic(), msg.Payload())
			}
		}
	}()
}

// Publish will send a message to broker with a specific topic.
func (m *MQTTv3) Publish(topic string, payload interface{}) error {
	token := m.client.Publish(topic, byte(m.config.QoS), m.config.Retained, payload)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	return nil
}

// GetConnectionStatus returns the connection status: Connected or Disconnected
func (m *MQTTv3) GetConnectionStatus() ConnectionState {
	return m.state
}

// Disconnect will close the connection to broker.
func (m *MQTTv3) Disconnect() {
	m.client.Disconnect(0)
	m.client = nil
	m.state = Disconnected
	m.disconnect <- true
}

func (m *MQTTv3) connect() error {
	options, err := m.createOptions()
	if err != nil {
		return err
	}

	m.client = mqttv3.NewClient(options)

	token := m.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.state = Connected

	return nil
}

func (m *MQTTv3) createOptions() (*mqttv3.ClientOptions, error) {
	options := mqttv3.NewClientOptions()

	for _, broker := range m.config.Brokers {
		options.AddBroker(broker)
	}

	if m.config.ClientID == "" {
		m.config.ClientID = "mqtt-client"
	}
	options.SetClientID(m.config.ClientID)

	if m.config.Username != "" {
		options.SetUsername(m.config.Username)
	}

	if m.config.Password != "" {
		options.SetPassword(m.config.Password)
	}

	if m.config.AutoReconnect {
		if m.config.MaxReconnectInterval == 0 {
			m.config.MaxReconnectInterval = time.Minute * 10
		}
		options.SetMaxReconnectInterval(m.config.MaxReconnectInterval)
	}
	// TLS Config
	if m.config.TLSCA != "" || (m.config.TLSKey != "" && m.config.TLSCert != "") {
		tlsConf, err := tlsConfig(m.config)
		if err != nil {
			return nil, err
		}
		options.SetTLSConfig(tlsConf)
	}

	options.SetAutoReconnect(m.config.AutoReconnect)
	options.SetKeepAlive(time.Second * 60)
	options.SetCleanSession(!m.config.PersistentSession)
	options.SetConnectionLostHandler(m.onConnectionLost)
	options.SetOnConnectHandler(m.onConnect)

	return options, nil
}

func (m *MQTTv3) onConnectionLost(c mqttv3.Client, err error) {
	m.state = Disconnected
}

func (m *MQTTv3) onConnect(c mqttv3.Client) {
	if len(m.config.Topics) != 0 {
		topics := make(map[string]byte)
		for _, topic := range m.config.Topics {
			if topic == "" {
				continue
			}
			topics[topic] = byte(m.config.QoS)
		}
		m.client.SubscribeMultiple(topics, m.onMessageReceived)
	}
}

func (m *MQTTv3) onMessageReceived(c mqttv3.Client, msg mqttv3.Message) {
	// Send received msg to messages channel
	m.messages <- msg
}
