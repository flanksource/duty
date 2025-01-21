package connection

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gocloud.dev/secrets"
	"gocloud.dev/secrets/awskms"
)

type AWSKMS struct {
	AWSConnection `json:",inline"`

	// keyID can be an alias (eg: alias/ExampleAlias?region=us-east-1) or the ARN
	KeyID string `json:"keyID,omitempty"`
}

func (t *AWSKMS) Populate(ctx ConnectionContext) error {
	return t.AWSConnection.Populate(ctx)
}

func (t *AWSKMS) FromModel(conn models.Connection) {
	t.AWSConnection.FromModel(conn)
	t.KeyID = conn.Properties["keyID"]
}

func (t *AWSKMS) SecretKeeper(ctx context.Context) (*secrets.Keeper, error) {
	awsConfig, err := t.AWSConnection.Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	kmsClient, err := awskms.DialV2(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS KMS client: %w", err)
	}

	keeper := awskms.OpenKeeperV2(kmsClient, t.KeyID, nil)
	return keeper, nil
}
