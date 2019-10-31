// Package mqtt-wrapper provides easy-to-use MQTT connection for projects.
package mqtt_wrapper

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"io/ioutil"
	"time"
)

// Connection state of the Client
type ConnectionState int

const (
	Disconnected ConnectionState = iota // No connection to broker
	Connected                           // Connection established to broker
)

// MQTTConfig contains configurable options for connecting to broker(s).
type MQTTConfig struct {
	Brokers              []string          // MQTT Broker address. Format: scheme://host:port
	ClientID             string            // Client ID
	Username             string            // Username to connect the broker(s)
	Password             string            // Password to connect the broker(s)
	Topics               []string          // Topics for subscription
	QoS                  int               // QoS
	AutoReconnect        bool              // Reconnect if connection is lost
	MaxReconnectInterval time.Duration     // maximum time that will be waited between reconnection attempts
	PersistentSession    bool              // Set session is persistent
	TLSCA                string            // CA file path
	TLSCert              string            // Cert file path
	TLSKey               string            // Key file path
	Messages             chan MQTT.Message // Channel for received message

	client  MQTT.Client
	options *MQTT.ClientOptions
	state   ConnectionState
}

// CreateConnection will automatically create connection to broker(s) with MQTTConfig parameters.
func (m *MQTTConfig) CreateConnection() error {

	if m.client != nil {
		return errors.New("mqtt client already initialized")
	}

	m.state = Disconnected

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

	// Create Channel for Subscribed Messages
	if len(m.Topics) != 0 && m.Messages == nil {
		m.Messages = make(chan MQTT.Message)
	}

	return nil
}

// Disconnect will close the connection to broker.
func (m *MQTTConfig) Disconnect() {
	m.client.Disconnect(0)
	m.client = nil
	m.state = Disconnected
}

// GetConnectionStatus returns the connection status: Connected or Disconnected
func (m *MQTTConfig) GetConnectionStatus() ConnectionState {
	return m.state
}

// Publish will send a message to broker with specific topic.
func (m *MQTTConfig) Publish(topic string, payload interface{}) error {
	token := m.client.Publish(topic, byte(m.QoS), false, payload)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
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

	if m.AutoReconnect {
		if m.MaxReconnectInterval == 0 {
			m.MaxReconnectInterval = time.Minute * 10
		}
		options.SetMaxReconnectInterval(m.MaxReconnectInterval)
	}
	// TLS Config
	if m.TLSCA != "" || (m.TLSKey != "" && m.TLSCert != "") {
		tlsConf, err := m.tlsConfig()
		if err != nil {
			return nil, err
		}
		options.SetTLSConfig(tlsConf)
	}

	options.SetAutoReconnect(m.AutoReconnect)
	options.SetKeepAlive(time.Second * 60)
	options.SetCleanSession(!m.PersistentSession)
	options.SetConnectionLostHandler(m.onConnectionLost)
	options.SetOnConnectHandler(m.onConnect)

	return options, nil
}

func (m *MQTTConfig) tlsConfig() (*tls.Config, error) {

	tlsConfig := &tls.Config{}

	if m.TLSCA != "" {
		pool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(m.TLSCA)
		if err != nil {
			return nil, err
		}
		check := pool.AppendCertsFromPEM(pem)
		if !check {
			return nil, errors.New("certificate can not added to pool")
		}
		tlsConfig.RootCAs = pool
	}

	if m.TLSCert != "" && m.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(m.TLSCert, m.TLSKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}
	return tlsConfig, nil
}

func (m *MQTTConfig) onConnectionLost(c MQTT.Client, err error) {
	m.state = Disconnected
}

func (m *MQTTConfig) onConnect(c MQTT.Client) {
	if len(m.Topics) != 0 {
		topics := make(map[string]byte)
		for _, topic := range m.Topics {
			if topic == "" {
				continue
			}
			topics[topic] = byte(m.QoS)
		}
		m.client.SubscribeMultiple(topics, m.onMessageReceived)
	}
}

func (m *MQTTConfig) onMessageReceived(c MQTT.Client, msg MQTT.Message) {
	// Send received msg to Messages channel
	m.Messages <- msg
}
