package tests

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

type RelatedConfigDirection string

const (
	RelatedConfigTypeIncoming RelatedConfigDirection = "incoming"
	RelatedConfigTypeOutgoing RelatedConfigDirection = "outgoing"
)

type RelatedConfig struct {
	Relation      string                 `json:"relation"`
	Direction     RelatedConfigDirection `json:"direction"`
	RelatedIDs    pq.StringArray         `json:"related_ids"`
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Tags          types.JSONStringMap    `json:"tags"`
	Changes       types.JSON             `json:"changes,omitempty"`
	Analysis      types.JSON             `json:"analysis,omitempty"`
	CostPerMinute *float64               `json:"cost_per_minute,omitempty"`
	CostTotal1d   *float64               `json:"cost_total_1d,omitempty"`
	CostTotal7d   *float64               `json:"cost_total_7d,omitempty"`
	CostTotal30d  *float64               `json:"cost_total_30d,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	AgentID       uuid.UUID              `json:"agent_id"`
	Status        *string                `json:"status" gorm:"default:null"`
	Ready         bool                   `json:"ready"`
	Health        *models.Health         `json:"health"`
}

var _ = ginkgo.Describe("Config relationship recursive", ginkgo.Ordered, func() {
	// Graph #1 (cylic)
	//
	//       A
	//      / \
	//     B   C
	//    / \   \
	//   D   E   F
	//  / \
	// G   H
	//    /
	//   A

	// Graph #2 (acyclic)
	//
	//        U
	//       / \
	//      V   W
	//     / \
	//    X   Y
	//   /
	//  Z

	// Graph #3 (multiple parent)
	//       L
	//      /|\
	//     M N O
	//      \|/
	//       p

	// Create a list of ConfigItems
	var (
		A = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("A"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		B = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("B"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		C = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("C"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		D = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("D"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		E = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("E"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		F = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("F"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		G = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("G"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		H = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("H"), Type: lo.ToPtr("A"), ConfigClass: "A"}

		L = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("L"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		M = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("M"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		N = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("N"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		O = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("O"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		P = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("p"), Type: lo.ToPtr("A"), ConfigClass: "A"}

		U = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("U"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		V = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("V"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		W = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("W"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		X = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("X"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		Y = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("Y"), Type: lo.ToPtr("A"), ConfigClass: "A"}
		Z = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-relationship"}, Name: lo.ToPtr("Z"), Type: lo.ToPtr("A"), ConfigClass: "A"}
	)
	configItems := []models.ConfigItem{
		A, B, C, D, E, F, G, H,
		L, M, N, O, P,
		U, V, W, X, Y, Z,
	}

	// Create relationships between ConfigItems
	relationships := []models.ConfigRelationship{
		{ConfigID: A.ID.String(), RelatedID: B.ID.String(), Relation: "test-relationship-AB"},
		{ConfigID: A.ID.String(), RelatedID: C.ID.String(), Relation: "test-relationship-AC"},
		{ConfigID: B.ID.String(), RelatedID: D.ID.String(), Relation: "test-relationship-BD"},
		{ConfigID: B.ID.String(), RelatedID: E.ID.String(), Relation: "test-relationship-BE"},
		{ConfigID: C.ID.String(), RelatedID: F.ID.String(), Relation: "test-relationship-CF"},
		{ConfigID: D.ID.String(), RelatedID: G.ID.String(), Relation: "test-relationship-DG"},
		{ConfigID: D.ID.String(), RelatedID: H.ID.String(), Relation: "test-relationship-DH"},
		{ConfigID: H.ID.String(), RelatedID: A.ID.String(), Relation: "test-relationship-HA"},

		{ConfigID: L.ID.String(), RelatedID: M.ID.String(), Relation: "test-relationship-LM"},
		{ConfigID: L.ID.String(), RelatedID: N.ID.String(), Relation: "test-relationship-LN"},
		{ConfigID: L.ID.String(), RelatedID: O.ID.String(), Relation: "test-relationship-LO"},
		{ConfigID: M.ID.String(), RelatedID: P.ID.String(), Relation: "test-relationship-MP"},
		{ConfigID: N.ID.String(), RelatedID: P.ID.String(), Relation: "test-relationship-NP"},
		{ConfigID: O.ID.String(), RelatedID: P.ID.String(), Relation: "test-relationship-OP"},

		{ConfigID: U.ID.String(), RelatedID: V.ID.String(), Relation: "test-relationship-UV"},
		{ConfigID: U.ID.String(), RelatedID: W.ID.String(), Relation: "test-relationship-UW"},
		{ConfigID: V.ID.String(), RelatedID: X.ID.String(), Relation: "test-relationship-VX"},
		{ConfigID: V.ID.String(), RelatedID: Y.ID.String(), Relation: "test-relationship-VY"},
		{ConfigID: X.ID.String(), RelatedID: Z.ID.String(), Relation: "test-relationship-XZ"},
	}

	ginkgo.BeforeAll(func() {
		err := DefaultContext.DB().Create(&configItems).Error
		Expect(err).To(BeNil())

		var foundConfigs []models.ConfigItem
		err = DefaultContext.DB().Select("id").Where("tags->>'namespace' = 'test-relationship'").Find(&foundConfigs).Error
		Expect(err).To(BeNil())
		Expect(len(foundConfigs)).To(Equal(len(configItems)))

		err = DefaultContext.DB().Create(&relationships).Error
		Expect(err).To(BeNil())

		var foundRelationships []models.ConfigRelationship
		err = DefaultContext.DB().Where("relation LIKE 'test-relationship%'").Find(&foundRelationships).Error
		Expect(err).To(BeNil())
		Expect(len(foundRelationships)).To(Equal(len(relationships)))
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Where("relation LIKE 'test-relationship%'").Delete(&models.ConfigRelationship{}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Where("tags->>'namespace' = 'test-relationship'").Delete(&models.ConfigItem{}).Error
		Expect(err).To(BeNil())
	})

	ginkgo.Context("Multiple parent graph", func() {
		ginkgo.It("should not return duplicate parents", func() {
			var relatedConfigs []RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", P.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			Expect(len(relatedConfigs)).To(Equal(4))
			relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{L.ID, M.ID, N.ID, O.ID}))
		})

		ginkgo.It("should not return duplicate children", func() {
			var relatedConfigs []RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false)", L.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			Expect(len(relatedConfigs)).To(Equal(4))
			relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{P.ID, M.ID, N.ID, O.ID}))
		})

		ginkgo.It("recursive both ways", func() {
			var relatedConfigs []RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'all')", G.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) string { return rc.Name })
			Expect(relatedIDs).To(ConsistOf([]string{*D.Name, *B.Name, *H.Name, *A.Name}))
		})
	})

	ginkgo.Context("Cyclic Graph", func() {
		ginkgo.Context("Outgoing", func() {
			ginkgo.It("should correctly return children in an acyclic path", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", C.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(1))

				Expect(relatedConfigs[0].ID.String()).To(Equal(F.ID.String()))
			})

			ginkgo.It("should correctly return zero relationships for leaf nodes", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", G.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(0))
			})

			ginkgo.It("should correctly handle cycles", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", A.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(7))

				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{B.ID, C.ID, D.ID, E.ID, F.ID, G.ID, H.ID}))
			})
		})

		ginkgo.Context("Incoming", func() {
			ginkgo.It("should return parents of a leaf node in a cyclic path", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", F.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				Expect(len(relatedConfigs)).To(Equal(5))
				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{C.ID, A.ID, H.ID, D.ID, B.ID}))
			})

			ginkgo.It("should return parents of a non-leaf node in a cyclic path", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", G.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{D.ID, B.ID, A.ID, H.ID}))
			})
		})

		ginkgo.Context("Both", func() {
			ginkgo.It("should return parents of a leaf node in a cyclic path", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'all')", F.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) string { return rc.Name })
				Expect(relatedIDs).To(ConsistOf([]string{*A.Name, *C.Name, *H.Name, *D.Name, *B.Name}))
			})
		})
	})

	ginkgo.Context("Acyclic Graph", func() {
		ginkgo.Context("Outgoing", func() {
			ginkgo.It("should correctly return children", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", U.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(5))

				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{V.ID, W.ID, X.ID, Y.ID, Z.ID}))
			})
		})

		ginkgo.Context("Incoming", func() {
			ginkgo.It("should return 0 parents for a root node", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", U.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(0))
			})

			ginkgo.It("should return parents of a leaf node", func() {
				var relatedConfigs []RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", Z.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(3))

				relatedIDs := lo.Map(relatedConfigs, func(rc RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{X.ID, V.ID, U.ID}))
			})
		})
	})
})

var _ = ginkgo.Describe("Config relationship", ginkgo.Ordered, func() {
	ginkgo.It("should return OUTGOING relationships", func() {
		var relatedConfigs []RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs(?, 'outgoing')", dummy.KubernetesCluster.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(2))
		for _, rc := range relatedConfigs {
			Expect(rc.Direction).To(Equal(RelatedConfigTypeOutgoing))
			Expect(rc.ID.String()).To(BeElementOf([]string{dummy.KubernetesNodeA.ID.String(), dummy.KubernetesNodeB.ID.String()}))
		}
	})

	ginkgo.It("should return INCOMING relationships", func() {
		var relatedConfigs []RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs(?, 'incoming', false)", dummy.KubernetesNodeA.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(1))
		Expect(relatedConfigs[0].Direction).To(Equal(RelatedConfigTypeIncoming))
		Expect(relatedConfigs[0].ID.String()).To(Equal(dummy.KubernetesCluster.ID.String()))
	})

	ginkgo.It("should return HARD OUTGOING relationships", func() {
		var relatedConfigs []RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 10, 'hard')", dummy.LogisticsAPIDeployment.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(2))
		for _, rc := range relatedConfigs {
			Expect(rc.Direction).To(Equal(RelatedConfigTypeOutgoing))
			Expect(rc.ID.String()).To(BeElementOf([]string{dummy.LogisticsAPIReplicaSet.ID.String(), dummy.LogisticsAPIPodConfig.ID.String()}))
		}
	})

	ginkgo.It("should return HARD incoming relationships", func() {
		var relatedConfigs []RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false, 10, 'hard')", dummy.LogisticsAPIReplicaSet.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(1))
		for _, rc := range relatedConfigs {
			Expect(rc.Direction).To(Equal(RelatedConfigTypeIncoming))
			Expect(rc.ID.String()).To(BeElementOf([]string{dummy.LogisticsAPIDeployment.ID.String()}))
		}
	})

	ginkgo.It("should return HARD incoming/outgoing relationships", func() {
		var relatedConfigs []RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'all', false, 10, 'hard')", dummy.LogisticsAPIReplicaSet.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(2))
		for _, rc := range relatedConfigs {
			Expect(rc.ID.String()).To(BeElementOf([]string{dummy.LogisticsAPIDeployment.ID.String(), dummy.LogisticsAPIPodConfig.ID.String()}))
			if rc.ID == dummy.LogisticsAPIDeployment.ID {
				Expect(rc.Direction).To(Equal(RelatedConfigTypeIncoming))
			} else {
				Expect(rc.Direction).To(Equal(RelatedConfigTypeOutgoing))
			}
		}
	})
})
