package pubsub

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/awssnssqs"
	"gocloud.dev/pubsub/kafkapubsub"
	"gocloud.dev/pubsub/natspubsub"
)

func Subscribe(ctx context.Context, c QueueConfig) (*pubsub.Subscription, error) {
	if c.SQS != nil {

		if c.SQS.WaitTime == 0 {
			c.SQS.WaitTime = 5
		}
		if err := c.SQS.AWSConnection.Populate(ctx); err != nil {
			return nil, err
		}
		ctx = ctx.WithName("aws")
		ctx.Logger.SetMinLogLevel(logger.Trace)
		ctx.Logger.SetLogLevel(logger.Info)
		sess, err := c.SQS.AWSConnection.Client(ctx)
		if err != nil {
			return nil, err
		}
		arn, err := ParseArn(c.SQS.QueueArn)
		if err != nil {
			return nil, err
		}

		client := sqs.NewFromConfig(sess, func(o *sqs.Options) {
			if c.SQS.Endpoint != "" {
				o.BaseEndpoint = &c.SQS.Endpoint
			}
		})
		ctx.Infof("Connecting to SQS queue: %s", arn.ToQueueURL())

		return awssnssqs.OpenSubscriptionV2(ctx, client, arn.ToQueueURL(), &awssnssqs.SubscriptionOptions{
			Raw:      c.SQS.RawDelivery,
			WaitTime: time.Second * time.Duration(c.SQS.WaitTime),
		}), nil
	}

	if c.PubSub != nil {
		if c.PubSub.ProjectID == "" || c.PubSub.Subscription == "" {
			return nil, fmt.Errorf("project_id and subscription are required for GCP Pub/Sub")
		}
		return pubsub.OpenSubscription(ctx, fmt.Sprintf("gcppubsub://projects/%s/subscriptions/%s", c.PubSub.ProjectID, c.PubSub.Subscription))
	}
	if c.Kafka != nil {
		return kafkapubsub.OpenSubscription(c.Kafka.Brokers, nil, c.Kafka.Group, []string{c.Kafka.Topic}, nil)
	}

	if c.RabbitMQ != nil {
		return pubsub.OpenSubscription(ctx, fmt.Sprintf("rabbit://%s", c.RabbitMQ.Queue))
	}

	if c.NATS != nil {
		conn, err := nats.Connect(c.NATS.URL)
		if err != nil {
			return nil, err
		}

		return natspubsub.OpenSubscriptionV2(conn, c.NATS.Subject, &natspubsub.SubscriptionOptions{
			Queue: c.NATS.Queue,
		})
	}

	if c.Memory != nil {
		return pubsub.OpenSubscription(ctx, fmt.Sprintf("mem://%s", c.Memory.QueueName))
	}

	return nil, fmt.Errorf("no queue configuration provided")
}
