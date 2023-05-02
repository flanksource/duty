package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

// GenerateJSONMD5Hash marshals the object into JSON and generates its md5 hash
func GenerateJSONMD5Hash(obj any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}
