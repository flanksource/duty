package dummy

import (
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

var EchoConfigRun1 = models.PlaybookRun{
	ID:         uuid.MustParse("17ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	PlaybookID: EchoConfig.ID,
	ConfigID:   &KubernetesNodeA.ID,
	Status:     models.PlaybookRunStatusCompleted,
	Spec:       []byte("{}"),
	CreatedAt:  DummyCreatedAt.Add(time.Minute),
}

var EchoConfigRun2 = models.PlaybookRun{
	ID:         uuid.MustParse("27ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	PlaybookID: EchoConfig.ID,
	ConfigID:   &EC2InstanceA.ID,
	Status:     models.PlaybookRunStatusCompleted,
	Spec:       []byte("{}"),
	CreatedAt:  DummyCreatedAt.Add(time.Minute * 10),
}

var RestartPodRun1 = models.PlaybookRun{
	ID:         uuid.MustParse("37ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	PlaybookID: RestartPod.ID,
	ConfigID:   &LogisticsAPIDeployment.ID,
	Status:     models.PlaybookRunStatusCompleted,
	Spec:       []byte("{}"),
	CreatedAt:  DummyCreatedAt.Add(time.Minute * 20),
}

var RestartPodRun2 = models.PlaybookRun{
	ID:         uuid.MustParse("47ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	ConfigID:   &LogisticsAPIDeployment.ID,
	PlaybookID: RestartPod.ID,
	Status:     models.PlaybookRunStatusFailed,
	Spec:       []byte("{}"),
	CreatedAt:  DummyCreatedAt.Add(time.Minute * 30),
}

var AllDummyPlaybookRuns = []models.PlaybookRun{
	EchoConfigRun1,
	EchoConfigRun2,
	RestartPodRun1,
	RestartPodRun2,
}
