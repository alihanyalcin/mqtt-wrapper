package mqtt_wrapper

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"time"
)

type MQTT interface {
	Handle(func(string, []byte))
	Publish(string, interface{}) error
	GetConnectionStatus() ConnectionState
	Disconnect()
}

type Version int

const (
	V3 Version = iota
	V5
)

// ConnectionState of the Client
type ConnectionState int

const (
	Disconnected ConnectionState = iota // No connection to broker
	Connected                           // Connection established to broker
)

// MQTTConfig contains configurable options for connecting to broker(s).
type MQTTConfig struct {
	Brokers              []string // MQTT Broker address. Format: scheme://host:port
	ClientID             string   // Client ID
	Username             string   // Username to connect the broker(s)
	Password             string   // Password to connect the broker(s)
	Topics               []string // Topics for subscription
	QoS                  int      // QoS
	Retained             bool
	AutoReconnect        bool          // Reconnect if connection is lost
	MaxReconnectInterval time.Duration // maximum time that will be waited between reconnection attempts
	PersistentSession    bool          // Set session is persistent
	TLSCA                string        // CA file path
	TLSCert              string        // Cert file path
	TLSKey               string        // Key file path
	Version              Version

	client MQTT
}

// CreateConnection will automatically create connection to broker(s) with MQTTConfig parameters.
func (m *MQTTConfig) CreateConnection() (MQTT, error) {

	if m.client != nil {
		return nil, errors.New("mqtt client already initialized")
	}

	if len(m.Brokers) == 0 {
		return nil, errors.New("no broker address to connect")
	}

	if m.QoS > 2 || m.QoS < 0 {
		return nil, errors.New("value of qos must be 0, 1, 2")
	}

	switch m.Version {
	case V3:
		client, err := newMQTTv3(m)
		if err != nil {
			return nil, err
		}

		m.client = client

		return client, nil
	case V5:
		client, err := newMQTTv5(m)
		if err != nil {
			return nil, err
		}

		m.client = client

		return client, nil
	}

	return nil, nil
}

func tlsConfig(config MQTTConfig) (*tls.Config, error) {

	tlsConfig := &tls.Config{}

	if config.TLSCA != "" {
		pool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(config.TLSCA)
		if err != nil {
			return nil, err
		}
		check := pool.AppendCertsFromPEM(pem)
		if !check {
			return nil, errors.New("certificate can not added to pool")
		}
		tlsConfig.RootCAs = pool
	}

	if config.TLSCert != "" && config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}
	return tlsConfig, nil
}
