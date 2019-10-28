package mqtt_wrapper

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSuccesfullConnection(t *testing.T) {
	
	client := MQTTConfig{
		Brokers:  []string{"tcp://192.168.0.99:1883", "192.168.0.99"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   nil,
		QoS:      0,
	}

	err := client.CreateConnection()
	if err != nil {
		t.Error("Error occurred:",err)
	}
}

func TestDoubleConnecion(t *testing.T) {
	client := MQTTConfig{
		Brokers:  []string{"192.168.0.99:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   nil,
		QoS:      0,
	}
	_ = client.CreateConnection()
	err := client.CreateConnection()
	assert.Equal(t,"mqtt client already initialized", err.Error())
}

func TestQoSValue(t *testing.T) {
	client := MQTTConfig{
		Brokers:  []string{"192.168.0.99:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   nil,
		QoS:     -1,
	}

	err := client.CreateConnection()
	assert.Equal(t, "value of qos must be 0, 1, 2", err.Error())

	client.QoS = 3
	err = client.CreateConnection()
	assert.Equal(t, "value of qos must be 0, 1, 2", err.Error())

	for i := 0; i <= 2; i++ {
		client.QoS = i
		err = client.CreateConnection()
		assert.NoError(t, err)
		client.Disconnect()
	}
}

func TestSubscribe(t *testing.T) {
	client := MQTTConfig{
		Brokers:  []string{"192.168.0.99:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   []string{"","","A","B"},
		QoS:      0,
	}
	err := client.CreateConnection()
	assert.NoError(t,err)
}