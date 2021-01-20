// Package mqtt provides easy-to-use MQTT connection for projects.
package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"time"
)

type handler func(topic string, payload []byte)
type responseHandler func(responseTopic string, payload []byte, id []byte)

// MQTT is an interface for mqttv3 and mqttv5 structs.
type MQTT interface {
	// Handle handles new messages to subscribed topics.
	Handle(handler)
	// Publish sends a message to broker with a specific topic.
	Publish(string, interface{}) error
	// Request sends a message to broker and waits for the response.
	Request(string, interface{}, time.Duration, handler) error
	// SubscribeResponse creates new subscription for response topic.
	SubscribeResponse(string) error
	// Respond sends message to response topic with correlation id (use inside HandleRequest).
	Respond(string, interface{}, []byte) error
	// HandleRequest handles imcoming request.
	HandleRequest(responseHandler)
	// GetConnectionStatus returns the connection status: Connected or Disconnected
	GetConnectionStatus() ConnectionState
	// Disconnect will close the connection to broker.
	Disconnect()
}

// Version of the client
type Version int

const (
	// V3 is MQTT Version 3
	V3 Version = iota
	// V5 is MQTT Version 5
	V5
)

// ConnectionState of the Client
type ConnectionState int

const (
	// Disconnected : no connection to broker
	Disconnected ConnectionState = iota
	// Connected : connection established to broker
	Connected
)

// Config contains configurable options for connecting to broker(s).
type Config struct {
	Brokers              []string      // MQTT Broker address. Format: scheme://host:port
	ClientID             string        // Client ID
	Username             string        // Username to connect the broker(s)
	Password             string        // Password to connect the broker(s)
	Topics               []string      // Topics for subscription
	QoS                  int           // QoS
	Retained             bool          // Retain Message
	AutoReconnect        bool          // Reconnect if connection is lost
	MaxReconnectInterval time.Duration // Maximum time that will be waited between reconnection attempts
	PersistentSession    bool          // Set persistent(clean start for v5) of session
	KeepAlive            uint16        // Keep Alive time in sec
	TLSCA                string        // CA file path
	TLSCert              string        // Cert file path
	TLSKey               string        // Key file path
	Version              Version       // MQTT Version of client
}

// CreateConnection will automatically create connection to broker(s) with MQTTConfig parameters.
func (m *Config) CreateConnection() (MQTT, error) {

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

		return client, nil
	case V5:
		client, err := newMQTTv5(m)
		if err != nil {
			return nil, err
		}

		return client, nil
	}

	return nil, nil
}

func (m *Config) tlsConfig() (*tls.Config, error) {

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
