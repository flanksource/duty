package connection

import (
	gocontext "context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/flanksource/duty/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	clusterIDHeader   = "x-k8s-aws-id"
	emptyStringSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	v1Prefix          = "k8s-aws-v1."
)

// +kubebuilder:object:generate=true
type EKSConnection struct {
	AWSConnection `json:",inline" yaml:",inline"`

	Cluster string `json:"cluster"`
}

func (t *EKSConnection) Populate(ctx ConnectionContext) error {
	return t.AWSConnection.Populate(ctx)
}

func (t *EKSConnection) KubernetesClient(ctx context.Context, freshToken bool) (kubernetes.Interface, *rest.Config, error) {
	awsConfig, err := t.AWSConnection.Client(ctx)
	if err != nil {
		return nil, nil, err
	}

	eksEndpoint, ca, err := eksClusterDetails(ctx, t.Cluster, awsConfig)
	if err != nil {
		return nil, nil, err
	}

	token, err := getEKSToken(ctx, t.Cluster, awsConfig, freshToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get token for EKS: %w", err)
	}

	restConfig := &rest.Config{
		Host:        eksEndpoint,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	return clientset, restConfig, nil
}

func eksClusterDetails(ctx gocontext.Context, clusterName string, conf aws.Config) (string, []byte, error) {
	eksClient := eks.NewFromConfig(conf)
	cluster, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &clusterName})
	if err != nil {
		return "", nil, fmt.Errorf("unable to get cluster info: %w", err)
	}

	ca, err := base64.URLEncoding.DecodeString(*cluster.Cluster.CertificateAuthority.Data)
	if err != nil {
		return "", nil, fmt.Errorf("unable to base64 decode ca: %w", err)
	}

	return *cluster.Cluster.Endpoint, ca, nil
}

func getEKSToken(ctx gocontext.Context, cluster string, conf aws.Config, freshToken bool) (string, error) {
	cred, err := conf.Credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrive credentials from aws config: %w", err)
	}

	cacheKey := tokenCacheKey("eks", cred, cluster)
	if !freshToken {
		if found, ok := tokenCache.Get(cacheKey); ok {
			return found.(string), nil
		}
	}

	signedURI, err := getSignedSTSURI(ctx, cluster, cred)
	if err != nil {
		return "", fmt.Errorf("failed to get signed URI: %w", err)
	}

	token := v1Prefix + base64.URLEncoding.EncodeToString([]byte(signedURI))
	tokenTTL := time.Minute * 15
	tokenCache.Set(cacheKey, token, tokenTTL-tokenSafetyMargin)
	return token, nil
}

func getSignedSTSURI(ctx gocontext.Context, cluster string, cred aws.Credentials) (string, error) {
	request, err := http.NewRequest(http.MethodGet, "https://sts.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15", nil)
	if err != nil {
		return "", err
	}

	request.Header.Add(clusterIDHeader, cluster)
	request.Header.Add("X-Amz-Expires", "86400") // 24 hours
	signer := signerv4.NewSigner()
	signedURI, _, err := signer.PresignHTTP(ctx, cred, request, emptyStringSha256, "sts", "us-east-1", time.Now())
	if err != nil {
		return "", err
	}

	return signedURI, nil
}
