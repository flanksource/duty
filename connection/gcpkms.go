package connection

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gocloud.dev/secrets"
	"gocloud.dev/secrets/gcpkms"
)

type GCPKMS struct {
	GCPConnection `json:",inline"`

	// keyID points to the key in the format
	// projects/MYPROJECT/locations/MYLOCATION/keyRings/MYKEYRING/cryptoKeys/MYKEY
	KeyID string `json:"keyID,omitempty"`
}

func (t *GCPKMS) Populate(ctx ConnectionContext) error {
	return t.GCPConnection.HydrateConnection(ctx)
}

func (t *GCPKMS) FromModel(conn models.Connection) {
	t.GCPConnection.FromModel(conn)
	t.KeyID = conn.Properties["keyID"]
}

func (t *GCPKMS) SecretKeeper(ctx context.Context) (*secrets.Keeper, error) {
	oauthToken, err := t.GCPConnection.TokenSource(ctx, "https://www.googleapis.com/auth/cloudkms")
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP oauth2 token: %w", err)
	}

	kmsClient, _, err := gcpkms.Dial(ctx, oauthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP KMS client: %w", err)
	}

	keeper := gcpkms.OpenKeeper(kmsClient, t.KeyID, nil)
	return keeper, nil
}
