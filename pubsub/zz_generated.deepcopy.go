//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package pubsub

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConfig) DeepCopyInto(out *KafkaConfig) {
	*out = *in
	if in.Brokers != nil {
		in, out := &in.Brokers, &out.Brokers
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConfig.
func (in *KafkaConfig) DeepCopy() *KafkaConfig {
	if in == nil {
		return nil
	}
	out := new(KafkaConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MemoryConfig) DeepCopyInto(out *MemoryConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MemoryConfig.
func (in *MemoryConfig) DeepCopy() *MemoryConfig {
	if in == nil {
		return nil
	}
	out := new(MemoryConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NATSConfig) DeepCopyInto(out *NATSConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NATSConfig.
func (in *NATSConfig) DeepCopy() *NATSConfig {
	if in == nil {
		return nil
	}
	out := new(NATSConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PubSubConfig) DeepCopyInto(out *PubSubConfig) {
	*out = *in
	in.GCPConnection.DeepCopyInto(&out.GCPConnection)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PubSubConfig.
func (in *PubSubConfig) DeepCopy() *PubSubConfig {
	if in == nil {
		return nil
	}
	out := new(PubSubConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueueConfig) DeepCopyInto(out *QueueConfig) {
	*out = *in
	if in.SQS != nil {
		in, out := &in.SQS, &out.SQS
		*out = new(SQSConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.PubSub != nil {
		in, out := &in.PubSub, &out.PubSub
		*out = new(PubSubConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.RabbitMQ != nil {
		in, out := &in.RabbitMQ, &out.RabbitMQ
		*out = new(RabbitConfig)
		**out = **in
	}
	if in.Memory != nil {
		in, out := &in.Memory, &out.Memory
		*out = new(MemoryConfig)
		**out = **in
	}
	if in.Kafka != nil {
		in, out := &in.Kafka, &out.Kafka
		*out = new(KafkaConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.NATS != nil {
		in, out := &in.NATS, &out.NATS
		*out = new(NATSConfig)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueueConfig.
func (in *QueueConfig) DeepCopy() *QueueConfig {
	if in == nil {
		return nil
	}
	out := new(QueueConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RabbitConfig) DeepCopyInto(out *RabbitConfig) {
	*out = *in
	out.URL = in.URL
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RabbitConfig.
func (in *RabbitConfig) DeepCopy() *RabbitConfig {
	if in == nil {
		return nil
	}
	out := new(RabbitConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SQSConfig) DeepCopyInto(out *SQSConfig) {
	*out = *in
	in.AWSConnection.DeepCopyInto(&out.AWSConnection)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SQSConfig.
func (in *SQSConfig) DeepCopy() *SQSConfig {
	if in == nil {
		return nil
	}
	out := new(SQSConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *URL) DeepCopyInto(out *URL) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new URL.
func (in *URL) DeepCopy() *URL {
	if in == nil {
		return nil
	}
	out := new(URL)
	in.DeepCopyInto(out)
	return out
}
