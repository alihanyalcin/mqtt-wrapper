package mqtt_wrapper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/eclipse/paho.golang/paho"
)

type mqttv5 struct {
	client     *paho.Client
	state      ConnectionState
	config     MQTTConfig
	messages   chan *paho.Publish
	disconnect chan bool
}

func newMQTTv5(config *MQTTConfig) (MQTT, error) {
	m := mqttv5{
		state:      Disconnected,
		config:     *config,
		messages:   make(chan *paho.Publish),
		disconnect: make(chan bool, 1),
	}

	err := m.connect()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *mqttv5) Handle(f func(string, []byte)) {
	go func() {
		for {
			select {
			case <-m.disconnect:
				return
			case msg := <-m.messages:
				f(msg.Topic, msg.Payload)
			}
		}
	}()
}

func (m *mqttv5) Publish(topic string, payload interface{}) error {
	var publishPayload []byte
	switch p := payload.(type) {
	case string:
		publishPayload = []byte(p)
	case []byte:
		publishPayload = p
	case bytes.Buffer:
		publishPayload = p.Bytes()
	default:
		return errors.New("unknown payload type")
	}

	_, err := m.client.Publish(context.Background(), &paho.Publish{
		Topic:   topic,
		QoS:     byte(m.config.QoS),
		Retain:  m.config.Retained,
		Payload: publishPayload,
	})

	if err != nil {
		return err
	}

	return nil
}

func (m *mqttv5) GetConnectionStatus() ConnectionState {
	return m.state
}

func (m *mqttv5) Disconnect() {
	m.client.Disconnect(&paho.Disconnect{
		ReasonCode: 0,
	})
	m.client = nil
	m.state = Disconnected
	m.disconnect <- true
}

func (m *mqttv5) connect() error {
	options, err := m.createOptions()
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", m.config.Brokers[0])
	if err != nil {
		log.Fatalf("Failed to connect to %s: %s", m.config.Brokers[0], err)
	}

	msg := make(chan *paho.Publish)

	c := paho.NewClient()
	c.Conn = conn
	c.Router = paho.NewSingleHandlerRouter(func(m *paho.Publish) {
		msg <- m
	})

	ca, err := c.Connect(context.Background(), options)
	if err != nil {
		return err
	}

	if ca.ReasonCode != 0 {
		return errors.New(
			fmt.Sprintf("Failed to connect to %s : %d - %s",
				m.config.Brokers[0],
				ca.ReasonCode,
				ca.Properties.ReasonString),
		)
	}

	if len(m.config.Topics) != 0 {
		topics := make(map[string]paho.SubscribeOptions)
		for _, t := range m.config.Topics {
			if t == "" {
				continue
			}
			topics[t] = paho.SubscribeOptions{
				QoS: byte(m.config.QoS),
			}
		}
		sa, err := c.Subscribe(context.Background(), &paho.Subscribe{
			Subscriptions: topics,
		})

		if err != nil {
			return err
		}

		if sa.Reasons[0] != byte(m.config.QoS) {
			return errors.New(fmt.Sprintf("Failed to subscribe: %d", sa.Reasons[0]))
		}
	}

	m.client = c
	m.messages = msg
	m.state = Connected

	return nil
}

func (m *mqttv5) createOptions() (*paho.Connect, error) {

	if m.config.ClientID == "" {
		m.config.ClientID = "mqttv5-client"
	}

	if m.config.TLSCA != "" || (m.config.TLSKey != "" && m.config.TLSCert != "") {
		return nil, errors.New("TLS support is not available yet.")
	}

	options := &paho.Connect{
		ClientID:   m.config.ClientID,
		Username:   m.config.Username,
		Password:   []byte(m.config.Password),
		KeepAlive:  60,
		CleanStart: true, // CleanSession???
	}

	if m.config.Username != "" {
		options.UsernameFlag = true
	}

	if m.config.Password != "" {
		options.PasswordFlag = true
	}

	return options, nil
}
