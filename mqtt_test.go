package mqtt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test succesfull connection
func TestSuccesfullConnection(t *testing.T) {

	config := Config{
		Brokers:  []string{"127.0.0.1:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   nil,
		QoS:      0,
	}

	_, err := config.CreateConnection()
	if err != nil {
		t.Error("Error occurred:", err)
	}
}

// Test QoS only get 0,1,2
func TestQoSValue(t *testing.T) {
	config := Config{
		Brokers:  []string{"127.0.0.1:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   nil,
		QoS:      -1,
	}

	_, err := config.CreateConnection()
	assert.Equal(t, "value of qos must be 0, 1, 2", err.Error())

	config.QoS = 3
	_, err = config.CreateConnection()
	assert.Equal(t, "value of qos must be 0, 1, 2", err.Error())

}

// Test empty topic will not recognized
func TestSubscribe(t *testing.T) {
	config := Config{
		Brokers:  []string{"127.0.0.1:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   []string{"", "", "A", "B"},
		QoS:      0,
	}
	_, err := config.CreateConnection()
	assert.NoError(t, err)
}

func TestPublish(t *testing.T) {
	config := Config{
		Brokers:  []string{"127.0.0.1:1883"},
		ClientID: "",
		Username: "",
		Password: "",
		Topics:   []string{"", "", "A", "B"},
		QoS:      0,
	}
	client, err := config.CreateConnection()
	err = client.Publish("topic", "message")
	assert.NoError(t, err)

}
