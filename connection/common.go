package connection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// caches cloud tokens. eg: EKS token, GKE token, ...
var tokenCache = cache.New(time.Hour, time.Hour)

func tokenCacheKey(cloud string, cred any, identifiers string) string {
	switch v := cred.(type) {
	case string:
		return fmt.Sprintf("%s-%s-%s", cloud, v, identifiers)
	default:
		m, _ := json.Marshal(v)
		return fmt.Sprintf("%s-%s-%s", cloud, m, identifiers)
	}
}
