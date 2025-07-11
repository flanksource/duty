//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package dataquery

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusQuery) DeepCopyInto(out *PrometheusQuery) {
	*out = *in
	in.PrometheusConnection.DeepCopyInto(&out.PrometheusConnection)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusQuery.
func (in *PrometheusQuery) DeepCopy() *PrometheusQuery {
	if in == nil {
		return nil
	}
	out := new(PrometheusQuery)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Query) DeepCopyInto(out *Query) {
	*out = *in
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(PrometheusQuery)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Query.
func (in *Query) DeepCopy() *Query {
	if in == nil {
		return nil
	}
	out := new(Query)
	in.DeepCopyInto(out)
	return out
}
