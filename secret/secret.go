package secret

import (
	"fmt"

	"github.com/flanksource/duty/context"
)

func Encrypt(ctx context.Context, sensitive Sensitive) (Ciphertext, error) {
	keeper, err := createOrGetKeeper(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret keeper from connection (%s): %w", KMSConnection, err)
	}

	ciphertext, err := keeper.Encrypt(ctx, []byte(sensitive.PlainText()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	return ciphertext, nil
}

func Decrypt(ctx context.Context, ciphertext Ciphertext) (Sensitive, error) {
	keeper, err := createOrGetKeeper(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret keeper from connection (%s): %w", KMSConnection, err)
	}

	decrypted, err := keeper.Decrypt(ctx, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return Sensitive(decrypted), nil
}
