package dummy

import (
	"github.com/flanksource/duty/models"
)

var EKSClusterLocation = models.ConfigLocation{
	ID:       EKSCluster.ID,
	Location: "aws/us-east-1/clusters/production-eks",
}

var KubernetesNodeALocation = models.ConfigLocation{
	ID:       KubernetesNodeA.ID,
	Location: "aws/us-east-1/clusters/production-eks/nodes/node-a",
}

var KubernetesNodeBLocation = models.ConfigLocation{
	ID:       KubernetesNodeB.ID,
	Location: "aws/us-west-2/clusters/production-eks/nodes/node-b",
}

var EC2InstanceALocation = models.ConfigLocation{
	ID:       EC2InstanceA.ID,
	Location: "aws/us-east-1/instances/i-1234567890abcdef0",
}

var EC2InstanceBLocation = models.ConfigLocation{
	ID:       EC2InstanceB.ID,
	Location: "aws/us-west-2/instances/i-0987654321fedcba0",
}

var LogisticsAPIDeploymentLocation = models.ConfigLocation{
	ID:       LogisticsAPIDeployment.ID,
	Location: "kubernetes/logistics/deployments/logistics-api",
}

var LogisticsUIDeploymentLocation = models.ConfigLocation{
	ID:       LogisticsUIDeployment.ID,
	Location: "kubernetes/logistics/deployments/logistics-ui",
}

var LogisticsWorkerDeploymentLocation = models.ConfigLocation{
	ID:       LogisticsWorkerDeployment.ID,
	Location: "kubernetes/logistics/deployments/logistics-worker",
}

var LogisticsDBRDSLocation = models.ConfigLocation{
	ID:       LogisticsDBRDS.ID,
	Location: "aws/us-east-1/rds/logistics-db",
}

var NginxHelmReleaseLocation = models.ConfigLocation{
	ID:       NginxHelmRelease.ID,
	Location: "kubernetes/ingress-nginx/helm/nginx-ingress",
}

var RedisHelmReleaseLocation = models.ConfigLocation{
	ID:       RedisHelmRelease.ID,
	Location: "kubernetes/database/helm/redis",
}

var AllDummyConfigLocations = []models.ConfigLocation{
	{ID: LogisticsAPIPodConfig.ID, Location: "node://kubernetes/node-a"},
	{ID: LogisticsAPIPodConfig.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsAPIPodConfig.ID, Location: "namespace://kubernetes/demo/missioncontrol"},
	{ID: LogisticsAPIPodConfig.ID, Location: "deployment://kubernetes/demo/missioncontrol/logistics-api/logistics-api-7df4c7f6b7-x9k2m"},
	{ID: LogisticsAPIPodConfig.ID, Location: "replicaset://kubernetes/demo/missioncontrol/logistics-api-7df4c7f6b7/logistics-api-7df4c7f6b7-x9k2m"},

	EKSClusterLocation,
	KubernetesNodeALocation,
	KubernetesNodeBLocation,
	EC2InstanceALocation,
	EC2InstanceBLocation,
	LogisticsAPIDeploymentLocation,
	LogisticsUIDeploymentLocation,
	LogisticsWorkerDeploymentLocation,
	LogisticsDBRDSLocation,
	NginxHelmReleaseLocation,
	RedisHelmReleaseLocation,
}
