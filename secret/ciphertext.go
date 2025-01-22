package secret

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const ciphertextPrefix = "enc:"

type Ciphertext []byte

func (t Ciphertext) String() string {
	return fmt.Sprintf("%s%s", ciphertextPrefix, base64.StdEncoding.EncodeToString(t))
}

func (t Ciphertext) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t Ciphertext) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func ParseCiphertext(s string) (Ciphertext, error) {
	if !strings.HasPrefix(s, ciphertextPrefix) {
		return nil, fmt.Errorf("invalid ciphertext prefix")
	}

	encoded := s[len(ciphertextPrefix):]
	if encoded == "" {
		return nil, fmt.Errorf("empty ciphertext")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return Ciphertext(data), nil
}
