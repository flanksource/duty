package tests

import (
	"fmt"

	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
	"github.com/flanksource/duty/types"
	"github.com/flanksource/duty/upstream"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

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
		err := DefaultContext.DB().Where("relation LIKE 'test-relationship%'").Delete(&models.ConfigRelationship{}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Where("tags->>'namespace' = 'test-relationship'").Delete(&models.ConfigItem{}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Create(&configItems).Error
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
			err := job.RefreshConfigItemSummary7d(DefaultContext)
			Expect(err).To(BeNil())

			var relatedConfigs []query.RelatedConfig
			err = DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", P.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			Expect(len(relatedConfigs)).To(Equal(5))
			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{L.ID, P.ID, M.ID, N.ID, O.ID}))
		})

		ginkgo.It("should not return duplicate children", func() {
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false)", L.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			Expect(len(relatedConfigs)).To(Equal(5))
			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{L.ID, P.ID, M.ID, N.ID, O.ID}))
		})

		ginkgo.It("recursive both ways", func() {
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'all')", G.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) string { return rc.Name })
			Expect(relatedIDs).To(ConsistOf([]string{*G.Name, *D.Name, *B.Name, *H.Name, *A.Name}))
		})
	})

	ginkgo.Context("Cyclic Graph", func() {
		ginkgo.Context("Outgoing", func() {
			ginkgo.It("should correctly return children in an acyclic path", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", C.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(2))

				relatedConfigNames := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) string { return rc.Name })
				Expect(relatedConfigNames).To(ConsistOf([]string{*C.Name, *F.Name}))
			})

			ginkgo.It("should only return itself for leaf nodes", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", G.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(1))
				Expect(relatedConfigs[0].ID).To(Equal(G.ID))
			})

			ginkgo.It("should correctly handle cycles", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", A.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(8))

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{A.ID, B.ID, C.ID, D.ID, E.ID, F.ID, G.ID, H.ID}))
			})
		})

		ginkgo.Context("Incoming", func() {
			ginkgo.It("should return parents of a leaf node in a cyclic path", func() {
				relatedConfigs, err := query.GetRelatedConfigs(DefaultContext, query.RelationQuery{ID: F.ID})
				Expect(err).To(BeNil())

				Expect(relatedConfigs).To(HaveLen(5))
				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{C.ID, A.ID, H.ID, D.ID, B.ID}))
			})

			ginkgo.It("should return parents of a non-leaf node in a cyclic path", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", G.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{D.ID, B.ID, A.ID, H.ID, G.ID}))
			})
		})

		ginkgo.Context("Both", func() {
			ginkgo.It("should return parents of a leaf node in a cyclic path", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'all')", F.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) string { return rc.Name })
				Expect(relatedIDs).To(ConsistOf([]string{*A.Name, *C.Name, *H.Name, *D.Name, *B.Name, *F.Name}))
			})
		})
	})

	ginkgo.Context("Acyclic Graph", func() {
		ginkgo.Context("Outgoing", func() {
			ginkgo.It("should correctly return children", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", U.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(6))

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{U.ID, V.ID, W.ID, X.ID, Y.ID, Z.ID}))
			})
		})

		ginkgo.Context("Incoming", func() {
			ginkgo.It("should return only the root node and no parents for a root node", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", U.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())
				Expect(len(relatedConfigs)).To(Equal(1))
				Expect(relatedConfigs[0].ID).To(Equal(U.ID))
			})

			ginkgo.It("should return parents of a leaf node", func() {
				var relatedConfigs []query.RelatedConfig
				err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", Z.ID).Find(&relatedConfigs).Error
				Expect(err).To(BeNil())

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{X.ID, V.ID, Z.ID, U.ID}))
			})
		})
	})
})

var _ = ginkgo.Describe("Config relationship Kubernetes", ginkgo.Ordered, func() {
	// Graph #1 (cyclic)
	//                              Cluster
	//                  ________________|________________
	//                 /                                 \
	//        Namespace(dev)                    Namespace(flux-system)
	//         _____________
	//        /                                          |
	// Deployment(config-db)                  Kustomization(aws-sandbox)
	//        _____________
	//       /
	// ReplicaSet
	//  _________\__
	// /                \
	// PodA              PodB

	// Create a list of ConfigItems
	var (
		cluster       = models.ConfigItem{ID: uuid.MustParse("1dc69673-7994-4e19-b630-19dffa30d2ec"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("Cluster"), Type: lo.ToPtr("Cluster"), ConfigClass: "Cluster"}
		namespacedev  = models.ConfigItem{ID: uuid.MustParse("5088dfde-badb-4208-a1bd-6e66c08e1d4a"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("dev"), Type: lo.ToPtr("Namespace"), ConfigClass: "Namespace"}
		deployment    = models.ConfigItem{ID: uuid.MustParse("beddeaa0-1948-494b-9e37-dcd93b572338"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("config-db"), Type: lo.ToPtr("Deployment"), ConfigClass: "Deployment"}
		replicaset    = models.ConfigItem{ID: uuid.MustParse("25422c1e-af71-4d63-aa38-ab16a5e6846b"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("ReplicaSet"), Type: lo.ToPtr("ReplicaSet"), ConfigClass: "ReplicaSet"}
		podA          = models.ConfigItem{ID: uuid.MustParse("d1cb6c2a-678c-4fdb-8451-017d216f6c0d"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("PodA"), Type: lo.ToPtr("Pod"), ConfigClass: "Pod"}
		podB          = models.ConfigItem{ID: uuid.MustParse("d6135916-c8e8-4982-a1a9-3777251aea10"), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("PodB"), Type: lo.ToPtr("Pod"), ConfigClass: "Pod"}
		fluxsystem    = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("flux-system"), Type: lo.ToPtr("Namespace"), ConfigClass: "Namespace"}
		kustomization = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-related-ids"}, Name: lo.ToPtr("aws-sandbox"), Type: lo.ToPtr("Kustomization"), ConfigClass: "Kustomization"}
	)
	namespacedev.ParentID = &cluster.ID
	fluxsystem.ParentID = &cluster.ID
	deployment.ParentID = &namespacedev.ID
	replicaset.ParentID = &deployment.ID
	podA.ParentID = &replicaset.ID
	podB.ParentID = &replicaset.ID
	kustomization.ParentID = &fluxsystem.ID
	configItems := []models.ConfigItem{
		cluster, namespacedev, fluxsystem,
		kustomization,
		deployment, replicaset, podA, podB,
	}

	// Create relationships between ConfigItems
	relationships := []models.ConfigRelationship{
		// hard relationships
		{ConfigID: cluster.ID.String(), RelatedID: namespacedev.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: cluster.ID.String(), RelatedID: fluxsystem.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: fluxsystem.ID.String(), RelatedID: kustomization.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: namespacedev.ID.String(), RelatedID: deployment.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: deployment.ID.String(), RelatedID: replicaset.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: replicaset.ID.String(), RelatedID: podA.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},
		{ConfigID: replicaset.ID.String(), RelatedID: podB.ID.String(), SelectorID: "test-related-ids", Relation: "hard"},

		// Soft relationship
		{ConfigID: kustomization.ID.String(), RelatedID: namespacedev.ID.String(), SelectorID: "test-related-ids", Relation: "kustomizationNamespace"},
	}

	ginkgo.BeforeAll(func() {

		Expect(models.DeleteAllConfigs(DefaultContext.DB(), configItems...)).To(BeNil())

		Expect(DefaultContext.DB().Create(&configItems).Error).To(BeNil())

		Expect(DefaultContext.DB().Exec("UPDATE config_items set path = config_path(id)").Error).To(BeNil())

		var foundConfigs []models.ConfigItem
		err := DefaultContext.DB().Select("id").Where("tags->>'namespace' = 'test-related-ids'").Find(&foundConfigs).Error
		Expect(err).To(BeNil())
		Expect(len(foundConfigs)).To(Equal(len(configItems)))

		err = DefaultContext.DB().Model(models.ConfigRelationship{}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "related_id"}, {Name: "config_id"}, {Name: "relation"}},
			DoNothing: true,
		}).Create(&relationships).Error
		Expect(err).To(BeNil())

	})

	ginkgo.AfterAll(func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM config_items WHERE tags->>'namespace' = ?", "test-related-ids").Error).To(BeNil())
	})

	ginkgo.It("should return deployment outgoing", func() {
		err := job.RefreshConfigItemSummary7d(DefaultContext)
		Expect(err).To(BeNil())

		relatedConfigs, err := query.GetRelatedConfigs(DefaultContext, query.RelationQuery{ID: deployment.ID, Relation: query.Outgoing})

		Expect(err).To(BeNil())

		relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
		Expect(relatedIDs).To(ConsistOf([]uuid.UUID{namespacedev.ID, cluster.ID, replicaset.ID, podA.ID, podB.ID}))

		outgoingRelatedIDsMap := map[string][]string{
			cluster.ID.String():      {namespacedev.ID.String()},
			namespacedev.ID.String(): {deployment.ID.String()},
			replicaset.ID.String():   {podA.ID.String(), podB.ID.String()},
		}
		for i := range relatedConfigs {
			Expect(outgoingRelatedIDsMap[relatedConfigs[i].ID.String()]).To(ConsistOf([]string(relatedConfigs[i].RelatedIDs)),
				fmt.Sprintf("name: %s, type: %s", relatedConfigs[i].Name, relatedConfigs[i].Type))
		}
	})

	ginkgo.Context("deployment incoming", func() {
		ginkgo.It("should return hard incoming related ids for deployment", func() {

			relatedConfigs, err := query.GetRelatedConfigs(DefaultContext, query.RelationQuery{
				ID:       deployment.ID,
				Relation: query.Incoming,
				Incoming: query.Hard,
			})

			Expect(err).To(BeNil())
			Expect(relatedConfigs).To(HaveLen(2))

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{namespacedev.ID, cluster.ID}))

			outgoingRelatedIDsMap := map[string][]string{
				cluster.ID.String():      {namespacedev.ID.String()},
				namespacedev.ID.String(): {deployment.ID.String()},
			}
			for i := range relatedConfigs {
				Expect(outgoingRelatedIDsMap[relatedConfigs[i].ID.String()]).To(ConsistOf([]string(relatedConfigs[i].RelatedIDs)),
					fmt.Sprintf("name: %s, type: %s", relatedConfigs[i].Name, relatedConfigs[i].Type))
			}
		})

		ginkgo.Context("deployment incoming", func() {
			ginkgo.It("should return soft incoming related ids for deployment", func() {

				relatedConfigs, err := query.GetRelatedConfigs(DefaultContext, query.RelationQuery{
					ID:       deployment.ID,
					Relation: query.Incoming,
					Incoming: query.Soft,
				})

				Expect(err).To(BeNil())
				Expect(relatedConfigs).To(HaveLen(4))

				relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
				Expect(relatedIDs).To(ConsistOf([]uuid.UUID{namespacedev.ID, cluster.ID, kustomization.ID, *kustomization.ParentID}))
			})
		})

		ginkgo.It("should return deployment incoming", func() {
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false, 10, 'both', 'both')", deployment.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]uuid.UUID{deployment.ID, cluster.ID, namespacedev.ID, fluxsystem.ID, kustomization.ID}))

			incomingRelatedIDsMap := map[string][]string{
				cluster.ID.String():       {namespacedev.ID.String(), fluxsystem.ID.String()},
				namespacedev.ID.String():  {deployment.ID.String()},
				fluxsystem.ID.String():    {kustomization.ID.String()},
				kustomization.ID.String(): {namespacedev.ID.String()},
			}
			for i := range relatedConfigs {
				Expect(incomingRelatedIDsMap[relatedConfigs[i].ID.String()]).To(ConsistOf([]string(relatedConfigs[i].RelatedIDs)))
			}
		})
	})
})

var _ = ginkgo.Describe("config relationship deletion test", func() {
	var tempRelationships = []models.ConfigRelationship{
		{
			ConfigID:  dummy.KubernetesCluster.ID.String(),
			RelatedID: dummy.KubernetesNodeA.ID.String(),
			Relation:  "TempRelationship",
		},
		{
			ConfigID:  dummy.KubernetesCluster.ID.String(),
			RelatedID: dummy.KubernetesNodeB.ID.String(),
			Relation:  "TempRelationship",
		},
	}

	ginkgo.It("should insert temp config relationships", func() {
		err := DefaultContext.DB().Create(&tempRelationships).Error
		Expect(err).To(BeNil())
	})

	ginkgo.It("should delete", func() {
		err := upstream.DeleteOnUpstream(DefaultContext, &upstream.PushData{
			ConfigRelationships: tempRelationships,
		})
		Expect(err).To(BeNil())

		var foundRelationships []models.ConfigRelationship
		err = DefaultContext.DB().Where("relation = 'TempRelationship'").Find(&foundRelationships).Error
		Expect(err).To(BeNil())
		Expect(len(foundRelationships)).To(Equal(0))
	})
})

var _ = ginkgo.Describe("config relationship depth", ginkgo.Ordered, func() {
	generator := pkgGenerator.ConfigGenerator{
		Nodes:                   pkgGenerator.ConfigTypeRequirements{Count: 3},
		Namespaces:              pkgGenerator.ConfigTypeRequirements{Count: 2},
		DeploymentPerNamespace:  pkgGenerator.ConfigTypeRequirements{Count: 2},
		ReplicaSetPerDeployment: pkgGenerator.ConfigTypeRequirements{Count: 4, Deleted: 3},
		PodsPerReplicaSet:       pkgGenerator.ConfigTypeRequirements{Count: 1, NumChangesPerConfig: 5, NumInsightsPerConfig: 2},
		Tags: map[string]string{
			"test": "true",
		},
	}

	ginkgo.BeforeAll(func() {
		generator.GenerateKubernetes()
		Expect(generator.Save(DefaultContext.DB())).To(BeNil())
	})

	ginkgo.AfterAll(func() {
		err := generator.Destroy(DefaultContext.DB())
		Expect(err).To(BeNil())

		assertConfigCount("", DeleteFilterNone, 0)
	})

	ginkgo.Context("cluster relationship", func() {
		ginkgo.It("should fetch level 1", func() {
			err := job.RefreshConfigItemSummary7d(DefaultContext)
			Expect(err).To(BeNil())

			cluster := generator.Generated.ConfigByTypes("Kubernetes::Cluster")[0]
			var relatedConfigs []query.RelatedConfig
			err = DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 1, 'hard', 'hard')", cluster.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			expected := lo.Map(generator.Generated.ConfigByTypes("Kubernetes::Cluster", "Kubernetes::Node", "Kubernetes::Namespace"), func(c models.ConfigItem, _ int) uuid.UUID { return c.ID })
			Expect(relatedIDs).To(ConsistOf(expected))
		})

		ginkgo.It("should fetch level 2", func() {
			cluster := generator.Generated.ConfigByTypes("Kubernetes::Cluster")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 2, 'hard', 'hard')", cluster.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			expected := lo.Map(generator.Generated.ConfigByTypes("Kubernetes::Cluster", "Kubernetes::Node", "Kubernetes::Namespace", "Kubernetes::Deployment"), func(c models.ConfigItem, _ int) uuid.UUID { return c.ID })
			Expect(relatedIDs).To(ConsistOf(expected))
		})

		ginkgo.It("should fetch level 3", func() {
			cluster := generator.Generated.ConfigByTypes("Kubernetes::Cluster")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 3, 'hard', 'hard')", cluster.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			expected := lo.Map(generator.Generated.ConfigByTypes("Kubernetes::Cluster", "Kubernetes::Node", "Kubernetes::Namespace", "Kubernetes::Deployment", "Kubernetes::ReplicaSet"), func(c models.ConfigItem, _ int) uuid.UUID { return c.ID })
			Expect(relatedIDs).To(ConsistOf(expected))
		})

		ginkgo.It("should fetch level 4", func() {
			cluster := generator.Generated.ConfigByTypes("Kubernetes::Cluster")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 4, 'hard', 'hard')", cluster.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedIDs := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) uuid.UUID { return rc.ID })
			expected := lo.Map(generator.Generated.ConfigByTypes("Kubernetes::Cluster", "Kubernetes::Node", "Kubernetes::Namespace", "Kubernetes::Deployment", "Kubernetes::ReplicaSet", "Kubernetes::Pod"), func(c models.ConfigItem, _ int) uuid.UUID { return c.ID })
			Expect(relatedIDs).To(ConsistOf(expected))
		})
	})

	ginkgo.Context("deployment relationship", func() {
		ginkgo.It("should fetch level 0", func() {
			deployment := generator.Generated.ConfigByTypes("Kubernetes::Deployment")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 0, 'hard', 'hard')", deployment.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			Expect(len(relatedConfigs)).To(Equal(1))
			Expect(relatedConfigs[0].ID.String()).To(Equal(deployment.ID.String()))
		})

		ginkgo.It("should fetch level 1", func() {
			dep := generator.Generated.ConfigByTypes("Kubernetes::Deployment")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 1, 'hard', 'hard')", dep.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedTypes := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) string { return rc.Type })
			Expect(lo.Uniq(relatedTypes)).To(ConsistOf([]string{"Kubernetes::Deployment", "Kubernetes::ReplicaSet"}))
		})

		ginkgo.It("should fetch level 2", func() {
			dep := generator.Generated.ConfigByTypes("Kubernetes::Deployment")[0]
			var relatedConfigs []query.RelatedConfig
			err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'outgoing', false, 2, 'hard', 'hard')", dep.ID).Find(&relatedConfigs).Error
			Expect(err).To(BeNil())

			relatedTypes := lo.Map(relatedConfigs, func(rc query.RelatedConfig, _ int) string { return rc.Type })
			Expect(lo.Uniq(relatedTypes)).To(ConsistOf([]string{"Kubernetes::Deployment", "Kubernetes::ReplicaSet", "Kubernetes::Pod"}))
		})
	})
})
