package hack

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

// Variables used to aid testing.
//
// It's better to fire up a single embedded database instance
// for the entire test suite.
// The variables are here so they can be imported by other packages as well.
var (
	TestDB       *gorm.DB
	TestDBPGPool *pgxpool.Pool
	TestClient   kubernetes.Interface
)
