package secret

import (
	"context"
	"testing"

	"gocloud.dev/secrets"
	_ "gocloud.dev/secrets/localsecrets"
)

func TestDecrypt(t *testing.T) {
	ctx := context.Background()
	keeper, err := secrets.OpenKeeper(ctx, "base64key://")
	if err != nil {
		t.Errorf("Failed to open keeper: %s", err)
	}

	const passphrase = "my_secret_passphrase"
	ciphertext, err := keeper.Encrypt(ctx, []byte(passphrase))
	if err != nil {
		t.Errorf("Failed to encrypt: %s", err)
	}

	sensitive, err := Decrypt(ctx, keeper, ciphertext)
	if err != nil {
		t.Errorf("Failed to decrypt: %s", err)
	}

	if sensitive.PlainText() != passphrase {
		t.Errorf("Expected decrypted passphrase to match")
	}
}
