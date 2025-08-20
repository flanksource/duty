package dummy

import (
	"github.com/flanksource/duty/models"
)

var AllDummyConfigLocations = []models.ConfigLocation{
	{ID: KubernetesNodeA.ID, Location: "cluster://aws/us-east-1/production-eks"},
	{ID: KubernetesNodeA.ID, Location: "cluster://kubernetes/demo"},

	{ID: MissionControlNamespace.ID, Location: "cluster://kubernetes/demo"},

	{ID: LogisticsUIDeployment.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsUIDeployment.ID, Location: "namespace://kubernetes/demo/missioncontrol"},

	{ID: LogisticsAPIDeployment.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsAPIDeployment.ID, Location: "namespace://kubernetes/demo/missioncontrol"},

	{ID: LogisticsAPIPodConfig.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsAPIPodConfig.ID, Location: "deployment://kubernetes/demo/missioncontrol/logistics-api"},
	{ID: LogisticsAPIPodConfig.ID, Location: "namespace://kubernetes/demo/missioncontrol"},
	{ID: LogisticsAPIPodConfig.ID, Location: "node://kubernetes/demo/node-a"},
	{ID: LogisticsAPIPodConfig.ID, Location: "replicaset://kubernetes/demo/missioncontrol/logistics-api-7df4c7f6b7"},

	{ID: LogisticsAPIReplicaSet.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsAPIReplicaSet.ID, Location: "namespace://kubernetes/demo/missioncontrol"},
	{ID: LogisticsAPIReplicaSet.ID, Location: "deployment://kubernetes/demo/missioncontrol/logistics-api"},

	{ID: LogisticsUIReplicaSet.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsUIReplicaSet.ID, Location: "namespace://kubernetes/demo/missioncontrol"},
	{ID: LogisticsUIReplicaSet.ID, Location: "deployment://kubernetes/demo/missioncontrol/logistics-ui"},

	{ID: LogisticsUIPodConfig.ID, Location: "cluster://kubernetes/demo"},
	{ID: LogisticsUIPodConfig.ID, Location: "deployment://kubernetes/demo/missioncontrol/logistics-ui"},
	{ID: LogisticsUIPodConfig.ID, Location: "namespace://kubernetes/demo/missioncontrol"},
	{ID: LogisticsUIPodConfig.ID, Location: "replicaset://kubernetes/demo/missioncontrol/logistics-ui-6c8f9b4d5e"},
	{ID: LogisticsUIPodConfig.ID, Location: "node://kubernetes/demo/node-a"},

	{ID: EKSCluster.ID, Location: "account://aws/flanksource"},
	{ID: EKSCluster.ID, Location: "region://aws/us-east-1"},

	{ID: KubernetesNodeAKSPool1.ID, Location: "cluster://kubernetes/demo"},
}
