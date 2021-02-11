package mqtt

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

type mqttv5 struct {
	client     *paho.Client
	state      ConnectionState
	config     Config
	messages   chan *paho.Publish
	requests   map[string]chan *paho.Publish
	responses  chan *paho.Publish
	disconnect chan bool
	brokers    []*url.URL

	sync.Mutex
}

func newMQTTv5(config *Config) (MQTT, error) {
	m := mqttv5{
		state:      Disconnected,
		config:     *config,
		messages:   make(chan *paho.Publish),
		disconnect: make(chan bool, 1),
		requests:   make(map[string]chan *paho.Publish),
		responses:  make(chan *paho.Publish),
	}

	err := m.connect()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// Handle handles new messages to subscribed topics.
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

// Request sends a message to broker and waits for the response.
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
		return fmt.Errorf("failed to request: %v", err)
	}

	select {
	case <-time.After(timeout):
		_ = m.getRequest(correlationID)

		return errors.New("request timeout")
	case resp := <-response:
		h(resp.Topic, resp.Payload)
		return nil
	}
}

// SubscribeResponse creates new subscription for response topic.
func (m *mqttv5) SubscribeResponse(topic string) error {
	m.client.Router.RegisterHandler(topic, func(p *paho.Publish) {
		if p.Properties != nil && p.Properties.CorrelationData != nil && p.Properties.ResponseTopic != "" {
			m.responses <- p
		}
	})

	_, err := m.client.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("response subscribe failed: %v", err)
	}

	return nil
}

// Respond sends message to response topic with correlation id (use inside HandleRequest).
func (m *mqttv5) Respond(responseTopic string, payload interface{}, id []byte) error {
	p, err := m.checkPayload(payload)
	if err != nil {
		return err
	}

	_, err = m.client.Publish(context.Background(), &paho.Publish{
		Properties: &paho.PublishProperties{
			CorrelationData: id,
		},
		Topic:   responseTopic,
		Payload: p,
	})
	if err != nil {
		return fmt.Errorf("failed to respond: %v", err)
	}

	return nil
}

// HandleRequest handles imcoming request.
func (m *mqttv5) HandleRequest(h responseHandler) {
	go func() {
		for {
			select {
			case <-m.disconnect:
				return
			case resp := <-m.responses:
				h(resp.Properties.ResponseTopic, resp.Payload, resp.Properties.CorrelationData)
			}
		}
	}()
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

	var conn net.Conn
	c := paho.NewClient()
	for _, broker := range m.brokers {
		conn, err = m.openConnection(broker)
		if err != nil {
			continue
		}

		c.Conn = conn
		c.Router = paho.NewStandardRouter()
		ca, err := c.Connect(context.Background(), options)
		if err != nil {
			continue
		}

		if ca.ReasonCode == 0 {
			// connected
			break
		}

		if conn != nil {
			c.Conn = nil
			conn.Close()
		}
	}

	if c.Conn == nil {
		return fmt.Errorf("Failed to connect to %s :", m.brokers)
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

			c.Router.RegisterHandler(t, func(p *paho.Publish) {
				m.messages <- p
			})
		}
		sa, err := c.Subscribe(context.Background(), &paho.Subscribe{
			Subscriptions: topics,
		})

		if err != nil {
			return err
		}

		if sa.Reasons[0] != byte(m.config.QoS) {
			return fmt.Errorf("Failed to subscribe: %d", sa.Reasons[0])
		}
	}

	// subscribe for request/response
	responseTopic := fmt.Sprintf("%s/responses", c.ClientID)

	c.Router.RegisterHandler(responseTopic, func(p *paho.Publish) {
		if p.Properties == nil || p.Properties.CorrelationData == nil {
			return
		}

		response := m.getRequest(string(p.Properties.CorrelationData))
		if response == nil {
			return
		}

		response <- p
	})

	_, err = c.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			responseTopic: {QoS: 1},
		},
	})
	if err != nil {
		return err
	}

	m.client = c
	m.state = Connected

	return nil
}

func (m *mqttv5) openConnection(uri *url.URL) (net.Conn, error) {
	switch uri.Scheme {
	case "mqtt", "tcp":
		conn, err := net.DialTimeout("tcp", uri.Host, time.Second*30)
		if err != nil {
			return nil, err
		}
		return conn, nil
	case "ssl", "tls", "mqtts", "mqtt+ssl", "tcps":
		tlsConf, err := m.config.tlsConfig()
		if err != nil {
			return nil, err
		}

		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second * 30}, "tcp", uri.Host, tlsConf)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	return nil, errors.New("other protocols not implemented yet.")
}

func (m *mqttv5) createOptions() (*paho.Connect, error) {

	for _, broker := range m.config.Brokers {
		re := regexp.MustCompile(`%(25)?`)
		if len(broker) > 0 && broker[0] == ':' {
			broker = "127.0.0.1" + broker
		}
		if !strings.Contains(broker, "://") {
			broker = "tcp://" + broker
		}
		broker = re.ReplaceAllLiteralString(broker, "%25")
		brokerURI, err := url.Parse(broker)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse %q broker address: %s", broker, err)
		}
		m.brokers = append(m.brokers, brokerURI)
	}

	if m.config.ClientID == "" {
		m.config.ClientID = "mqttv5-client"
	}

	if m.config.KeepAlive == 0 {
		m.config.KeepAlive = 30
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
	if !ok {
		return nil
	}
	delete(m.requests, id)

	return response
}
