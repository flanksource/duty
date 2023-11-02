package testutils

import (
	"github.com/flanksource/duty/context"
	"k8s.io/client-go/kubernetes"
)

// Variables used to aid testing.
//
// It's better to fire up a single embedded database instance
// for the entire test suite.
// The variables are here so they can be imported by other packages as well.
var (
	TestClient     kubernetes.Interface
	DefaultContext context.Context
)
