package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
)

func TestSecretString(t *testing.T) {
	const secretKey = "my_secret_string"

	t.Run("simple .String()", func(t *testing.T) {
		secret := Sensitive(secretKey)
		if secret.String() != sensitivePlaceholder {
			t.Errorf("Expected secret.String() to return %s", sensitivePlaceholder)
		}
	})

	t.Run("formatted", func(t *testing.T) {
		secret := Sensitive(secretKey)
		if fmt.Sprintf("%s.", secret) != sensitivePlaceholder+"." { // added a period to avoid LSP warning
			t.Errorf("Expected secret.String() to return %s", sensitivePlaceholder)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		type myJSON struct {
			Secret Sensitive
		}

		m := myJSON{
			Secret: Sensitive(secretKey),
		}
		marshalled, err := json.Marshal(m)
		if err != nil {
			t.Errorf("Failed to marshal JSON: %s", err)
		}

		if string(marshalled) != fmt.Sprintf(`{"Secret":"%s"}`, sensitivePlaceholder) {
			t.Errorf("Expected marshalled JSON to contain redacted")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		secret := Sensitive(secretKey)
		secret.Clear()
		if len(secret) != 0 {
			t.Errorf("Expected secret to be cleared")
		}
	})

	t.Run("PlainText", func(t *testing.T) {
		secret := Sensitive(secretKey)
		if secret.PlainText() != secretKey {
			t.Errorf("Expected secret to match plain text")
		}
	})

	t.Run("Logger", func(t *testing.T) {
		var buffer bytes.Buffer
		myLogger := slog.New(slog.NewTextHandler(&buffer, nil))
		myLogger.Info("secret: %s", slog.Any("secret", Sensitive(secretKey)))
		if bytes.Contains(buffer.Bytes(), []byte(secretKey)) || !bytes.Contains(buffer.Bytes(), []byte(sensitivePlaceholder)) {
			t.Errorf("Expected log to contain redacted")
		}
	})
}
