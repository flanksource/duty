package pubsub

import (
	"fmt"
	"net/url"

	"github.com/flanksource/duty/connection"
)

// +kubebuilder:object:generate=true
type QueueConfig struct {
	SQS      *SQSConfig    `json:"sqs,omitempty"`
	PubSub   *PubSubConfig `json:"pubsub,omitempty"`
	RabbitMQ *RabbitConfig `json:"rabbitmq,omitempty"`
	Memory   *MemoryConfig `json:"memory,omitempty"`
	Kafka    *KafkaConfig  `json:"kafka,omitempty"`
	NATS     *NATSConfig   `json:"nats,omitempty"`
}

func (c QueueConfig) GetQueue() fmt.Stringer {
	if c.SQS != nil {
		return *c.SQS
	}
	if c.PubSub != nil {
		return *c.PubSub
	}
	if c.RabbitMQ != nil {
		return *c.RabbitMQ
	}
	if c.Memory != nil {
		return *c.Memory
	}
	if c.Kafka != nil {
		return *c.Kafka
	}
	if c.NATS != nil {
		return *c.NATS
	}
	return nil
}

// +kubebuilder:object:generate=true
type SQSConfig struct {
	QueueArn    string `json:"queue"`
	RawDelivery bool   `json:"raw"`
	// Time in seconds to long-poll for messages, Default to 15, max is 20
	WaitTime                 int `json:"waitTime,omitempty"`
	connection.AWSConnection `json:",inline"`
}

func (s SQSConfig) String() string {
	return s.QueueArn
}

// +kubebuilder:object:generate=true
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
	Topic   string   `json:"topic"`
	Group   string   `json:"group"`
}

func (k KafkaConfig) String() string {
	return fmt.Sprintf("kafka://%s", k.Topic)
}

// +kubebuilder:object:generate=true
type PubSubConfig struct {
	ProjectID                string `json:"project_id"`
	Subscription             string `json:"subscription"`
	connection.GCPConnection `json:",inline"`
}

func (p PubSubConfig) String() string {
	return fmt.Sprintf("gcppubsub://projects/%s/subscriptions/%s", p.ProjectID, p.Subscription)
}

// +kubebuilder:object:generate=true
type NATSConfig struct {
	URL     string `json:"url,omitempty"`
	Subject string `json:"subject"`
	Queue   string `json:"queue,omitempty"`
}

func (n NATSConfig) String() string {
	return fmt.Sprintf("nats://%s", n.Subject)
}

// +kubebuilder:object:generate=true
type RabbitConfig struct {
	URL   `json:",inline"`
	Queue string `json:"queue"`
}

func (r RabbitConfig) String() string {
	return fmt.Sprintf("rabbit://%s", r.Queue)
}

// +kubebuilder:object:generate=true
type MemoryConfig struct {
	QueueName string `json:"queue"`
}

func (m MemoryConfig) String() string {
	return fmt.Sprintf("mem://%s", m.QueueName)
}

// +kubebuilder:object:generate=true
type URL struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (u URL) String() string {
	_url := url.URL{
		Host: u.Host,
	}
	if u.Username != "" {
		_url.User = url.UserPassword(u.Username, u.Password)

	}
	return _url.String()
}
