package mqtt

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttv3 struct {
	client     mqtt.Client
	state      ConnectionState
	config     Config
	messages   chan mqtt.Message
	disconnect chan bool
}

func newMQTTv3(config *Config) (MQTT, error) {
	m := mqttv3{
		state:      Disconnected,
		config:     *config,
		messages:   make(chan mqtt.Message),
		disconnect: make(chan bool, 1),
	}

	err := m.connect()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// Handle handles new messages to subscribed topics.
func (m *mqttv3) Handle(h handler) {
	go func() {
		for {
			select {
			case <-m.disconnect:
				return
			case msg := <-m.messages:
				h(msg.Topic(), msg.Payload())
			}
		}
	}()
}

// Publish will send a message to broker with a specific topic.
func (m *mqttv3) Publish(topic string, payload interface{}) error {
	token := m.client.Publish(topic, byte(m.config.QoS), m.config.Retained, payload)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Request sends a message to broker and waits for the response.
func (m *mqttv3) Request(topic string, payload interface{}, timeout time.Duration, h handler) error {
	panic("unimplemented for mqttv3")
}

// RequestWith sends a message to broker with specific response topic,
// and waits for the response.
func (m *mqttv3) RequestWith(topic, responseTopic string, payload interface{}, timeout time.Duration, h handler) error {
	panic("unimplemented for mqttv3")
}

// SubscribeResponse creates new subscription for response topic.
func (m *mqttv3) SubscribeResponse(topic string) error {
	panic("not implemented for mqttv3")
}

// Respond sends message to response topic with correlation id (use inside HandleRequest).
func (m *mqttv3) Respond(responseTopic string, payload interface{}, id []byte) error {
	panic("not implemented for mqttv3")
}

// HandleRequest handles imcoming request.
func (m *mqttv3) HandleRequest(h responseHandler) {
	panic("not implemented for mqttv3")
}

// GetConnectionStatus returns the connection status: Connected or Disconnected
func (m *mqttv3) GetConnectionStatus() ConnectionState {
	return m.state
}

// Disconnect will close the connection to broker.
func (m *mqttv3) Disconnect() {
	m.client.Disconnect(0)
	m.client = nil
	m.state = Disconnected
	m.disconnect <- true
}

func (m *mqttv3) connect() error {
	options, err := m.createOptions()
	if err != nil {
		return err
	}

	m.client = mqtt.NewClient(options)

	token := m.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.state = Connected

	return nil
}

func (m *mqttv3) createOptions() (*mqtt.ClientOptions, error) {
	options := mqtt.NewClientOptions()

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
		tlsConf, err := m.config.tlsConfig()
		if err != nil {
			return nil, err
		}
		options.SetTLSConfig(tlsConf)
	}

	options.SetAutoReconnect(m.config.AutoReconnect)
	options.SetKeepAlive(time.Second * time.Duration(m.config.KeepAlive))
	options.SetCleanSession(!m.config.PersistentSession)
	options.SetConnectionLostHandler(m.onConnectionLost)
	options.SetOnConnectHandler(m.onConnect)

	return options, nil
}

func (m *mqttv3) onConnectionLost(c mqtt.Client, err error) {
	m.state = Disconnected
}

func (m *mqttv3) onConnect(c mqtt.Client) {
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

func (m *mqttv3) onMessageReceived(c mqtt.Client, msg mqtt.Message) {
	// Send received msg to messages channel
	m.messages <- msg
}
