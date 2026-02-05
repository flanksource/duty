package pubsub

import (
	"fmt"
	"strings"
)

type Arn struct {
	Partition string
	Service   string
	Region    string
	Account   string
	Resource  string
}

// e.g. const queueURL = "https://sqs.us-east-2.amazonaws.com/123456789012/MyQueue"
func (arn Arn) ToQueueURL() string {
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", arn.Region, arn.Account, arn.Resource)
}

// e.g. arn:aws:sqs:us-east-2:444455556666:queue1
func ParseArn(s string) (Arn, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return Arn{}, fmt.Errorf("invalid ARN: %s", s)
	}
	return Arn{
		Partition: parts[1],
		Service:   parts[2],
		Region:    parts[3],
		Account:   parts[4],
		Resource:  parts[5],
	}, nil
}
