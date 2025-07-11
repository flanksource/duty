//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package k8s

import (
	"github.com/flanksource/duty/types"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Request) DeepCopyInto(out *Request) {
	*out = *in
	out.LogsRequestBase = in.LogsRequestBase
	if in.Pods != nil {
		in, out := &in.Pods, &out.Pods
		*out = make(types.ResourceSelectors, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = make(types.MatchExpressions, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Request.
func (in *Request) DeepCopy() *Request {
	if in == nil {
		return nil
	}
	out := new(Request)
	in.DeepCopyInto(out)
	return out
}
