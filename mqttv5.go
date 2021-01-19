package mqtt_wrapper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

type mqttv5 struct {
	client     *paho.Client
	state      ConnectionState
	config     MQTTConfig
	messages   chan *paho.Publish
	disconnect chan bool
	requests   map[string]chan *paho.Publish
	sync.Mutex
}

func newMQTTv5(config *MQTTConfig) (MQTT, error) {
	m := mqttv5{
		state:      Disconnected,
		config:     *config,
		messages:   make(chan *paho.Publish),
		disconnect: make(chan bool, 1),
		requests:   make(map[string]chan *paho.Publish),
	}

	err := m.connect()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// Handle new messages
func (m *mqttv5) Handle(h handler) {
	go func() {
		for {
			select {
			case <-m.disconnect:
				return
			case msg := <-m.messages:
				h(msg.Topic, msg.Payload)
			}
		}
	}()
}

// Publish will send a message to broker with a specific topic.
func (m *mqttv5) Publish(topic string, payload interface{}) error {
	p, err := m.checkPayload(payload)
	if err != nil {
		return err
	}

	_, err = m.client.Publish(context.Background(), &paho.Publish{
		Topic:   topic,
		QoS:     byte(m.config.QoS),
		Retain:  m.config.Retained,
		Payload: p,
	})

	if err != nil {
		return err
	}

	return nil
}

func (m *mqttv5) Request(topic string, payload interface{}, timeout time.Duration, h handler) error {
	p, err := m.checkPayload(payload)
	if err != nil {
		return err
	}

	correlationID := fmt.Sprintf("%d", time.Now().UnixNano())
	response := make(chan *paho.Publish)

	m.setRequest(correlationID, response)

	_, err = m.client.Publish(context.Background(), &paho.Publish{
		Properties: &paho.PublishProperties{
			CorrelationData: []byte(correlationID),
			ResponseTopic:   fmt.Sprintf("%s/responses", m.client.ClientID),
		},
		Topic:   topic,
		Payload: p,
	})
	if err != nil {
		return err
	}

	select {
	case <-time.After(timeout):
		resp := m.getRequest(correlationID)
		if resp != nil {
			close(resp)
		}
		return errors.New("request timeout")
	case resp := <-response:
		h(resp.Topic, resp.Payload)
		return nil
	}
}

// GetConnectionStatus returns the connection status: Connected or Disconnected
func (m *mqttv5) GetConnectionStatus() ConnectionState {
	return m.state
}

// Disconnect will close the connection to broker.
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

	// subscribe topics
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

	// subscribe for request/response
	responseTopic := fmt.Sprintf("%s/responses", c.ClientID)

	c.Router.RegisterHandler(responseTopic, m.responseHandler)
	_, err = c.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			responseTopic: {QoS: 1},
		},
	})
	if err != nil {
		return err
	}

	m.client = c
	m.messages = msg
	m.state = Connected

	return nil
}

func (m *mqttv5) createOptions() (*paho.Connect, error) {

	if m.config.TLSCA != "" || (m.config.TLSKey != "" && m.config.TLSCert != "") {
		return nil, errors.New("TLS support is not available yet.")
	}

	if m.config.ClientID == "" {
		m.config.ClientID = "mqttv5-client"
	}

	options := &paho.Connect{
		ClientID:   m.config.ClientID,
		Username:   m.config.Username,
		Password:   []byte(m.config.Password),
		KeepAlive:  m.config.KeepAlive,
		CleanStart: m.config.PersistentSession,
	}

	if m.config.Username != "" {
		options.UsernameFlag = true
	}

	if m.config.Password != "" {
		options.PasswordFlag = true
	}

	return options, nil
}

func (m *mqttv5) responseHandler(p *paho.Publish) {
	if p.Properties == nil || p.Properties.CorrelationData == nil {
		return
	}

	response := m.getRequest(string(p.Properties.CorrelationData))
	if response == nil {
		return
	}

	response <- p
}

func (m *mqttv5) checkPayload(payload interface{}) ([]byte, error) {
	switch p := payload.(type) {
	case string:
		return []byte(p), nil
	case []byte:
		return p, nil
	case bytes.Buffer:
		return p.Bytes(), nil
	default:
		return nil, errors.New("unknown payload type")
	}
}

func (m *mqttv5) setRequest(id string, r chan *paho.Publish) {
	m.Lock()
	defer m.Unlock()

	m.requests[id] = r
}

func (m *mqttv5) getRequest(id string) chan *paho.Publish {
	m.Lock()
	defer m.Unlock()

	response, ok := m.requests[id]
	if ok {
		delete(m.requests, id)

		return response
	}
	return nil
}
