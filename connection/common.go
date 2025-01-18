package connection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// tokenSafetyMargin is the duration to deduct when caching tokens.
// Example: if a token expires in 15 minutes then we cache it with a
// TTL of 15 minutes - tokenSafetyMargin = 10 minutes.
//
// This is to prevent using a token that immediately expires.
const tokenSafetyMargin = time.Minute * 5

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
