package types

import (
	"encoding/json"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type kinded interface {
	Kind() string
}

func expectKind(v kinded) map[string]any {
	data, err := json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())

	var m map[string]any
	Expect(json.Unmarshal(data, &m)).To(Succeed())
	Expect(m["kind"]).To(Equal(v.Kind()))
	return m
}

var _ = Describe("ConfigChangeDetails", func() {
	DescribeTable("current detail payloads implement ConfigChangeDetail",
		func(d ConfigChangeDetail, expectedFields map[string]any) {
			m := expectKind(d)
			for key, val := range expectedFields {
				Expect(m[key]).To(Equal(val))
			}
		},
		Entry("UserChange", UserChangeDetails{UserName: "alice"}, map[string]any{
			"user_name": "alice",
		}),
		Entry("Screenshot", ScreenshotDetails{URL: "https://example.com"}, map[string]any{
			"url": "https://example.com",
		}),
		Entry("PermissionChange", PermissionChangeDetails{RoleName: "admin"}, map[string]any{
			"role_name": "admin",
		}),
		Entry("GroupMembership", GroupMembership{
			Group:  Identity{Name: "platform-admins", Type: IdentityTypeGroup},
			Member: Identity{Name: "alice", Type: IdentityTypeUser},
			Action: GroupMembershipActionAdded,
		}, map[string]any{
			"action": string(GroupMembershipActionAdded),
		}),
		Entry("Backup", Backup{Status: StatusCompleted, Size: "4.2GB"}, map[string]any{
			"status": string(StatusCompleted),
			"size":   "4.2GB",
		}),
		Entry("Scale", Scale{
			Dimension:     ScalingDimensionReplicas,
			PreviousValue: Dimension{Desired: "2"},
			Value:         Dimension{Desired: "5"},
		}, map[string]any{
			"dimension": string(ScalingDimensionReplicas),
		}),
	)

	DescribeTable("exported structs inject kind",
		func(v kinded, expectedFields map[string]any) {
			m := expectKind(v)
			for key, val := range expectedFields {
				Expect(m[key]).To(Equal(val))
			}
		},
		Entry("Identity", Identity{Name: "alice"}, map[string]any{
			"name": "alice",
		}),
		Entry("Approval", Approval{
			Event:       Event{ID: "evt-1"},
			SubmittedBy: &Identity{Name: "submitter"},
			Approver:    &Identity{Name: "approver"},
			Stage:       ApprovalStageManual,
			Status:      ApprovalStatusApproved,
		}, map[string]any{
			"id":     "evt-1",
			"stage":  string(ApprovalStageManual),
			"status": string(ApprovalStatusApproved),
		}),
		Entry("GitSource", GitSource{URL: "https://example.com/repo.git"}, map[string]any{
			"url": "https://example.com/repo.git",
		}),
		Entry("HelmSource", HelmSource{ChartName: "api"}, map[string]any{
			"chart_name": "api",
		}),
		Entry("ImageSource", ImageSource{ImageName: "backend"}, map[string]any{
			"image": "backend",
		}),
		Entry("DatabaseSource", DatabaseSource{Name: "appdb"}, map[string]any{
			"name": "appdb",
		}),
		Entry("Source", Source{
			Git:  &GitSource{URL: "https://example.com/repo.git"},
			Path: "deploy/app",
		}, map[string]any{
			"path": "deploy/app",
		}),
		Entry("Environment", Environment{Name: "prod"}, map[string]any{
			"name": "prod",
		}),
		Entry("Event", Event{ID: "evt-2"}, map[string]any{
			"id": "evt-2",
		}),
		Entry("Test", Test{
			Event:  Event{ID: "evt-3"},
			Name:   "smoke",
			Type:   TestingTypeE2E,
			Status: TestingStatusPassed,
			Result: TestingResultPassed,
		}, map[string]any{
			"id":     "evt-3",
			"name":   "smoke",
			"type":   string(TestingTypeE2E),
			"status": string(TestingStatusPassed),
			"result": string(TestingResultPassed),
		}),
		Entry("Promotion", Promotion{
			Event:   Event{ID: "evt-4"},
			From:    Environment{Name: "staging"},
			To:      Environment{Name: "prod"},
			Source:  Source{Git: &GitSource{URL: "https://example.com/repo.git"}},
			Version: "v1.2.3",
		}, map[string]any{
			"id":      "evt-4",
			"version": "v1.2.3",
		}),
		Entry("PipelineRun", PipelineRun{
			Event:       Event{ID: "evt-5"},
			Environment: Environment{Name: "prod"},
			Status:      StatusRunning,
		}, map[string]any{
			"id":     "evt-5",
			"status": string(StatusRunning),
		}),
		Entry("Change", Change{
			Path: ".spec.replicas",
			From: map[string]any{"desired": "2"},
			To:   map[string]any{"desired": "3"},
			Type: "update",
		}, map[string]any{
			"path": ".spec.replicas",
			"type": "update",
		}),
		Entry("ConfigChange", ConfigChange{
			Event:       Event{ID: "evt-6"},
			Author:      Identity{Name: "alice"},
			Changes:     []Change{{Path: ".spec.replicas", Type: "update"}},
			Environment: Environment{Name: "prod"},
			Source:      Source{Git: &GitSource{URL: "https://example.com/repo.git"}},
		}, map[string]any{
			"id": "evt-6",
		}),
		Entry("Restore", Restore{
			Event:  Event{ID: "evt-7"},
			From:   Environment{Name: "backup"},
			To:     Environment{Name: "prod"},
			Source: Source{Database: &DatabaseSource{Name: "appdb"}},
			Status: StatusCompleted,
		}, map[string]any{
			"id":     "evt-7",
			"status": string(StatusCompleted),
		}),
		Entry("Dimension", Dimension{Desired: "3"}, map[string]any{
			"desired": "3",
		}),
	)

	It("preserves embedded event fields and nested kinds", func() {
		data, err := json.Marshal(ConfigChange{
			Event:  Event{ID: "evt-8", Timestamp: "2026-04-10T12:00:00Z"},
			Author: Identity{Name: "alice"},
			Changes: []Change{{
				Path: ".spec.replicas",
				Type: "update",
			}},
			Environment: Environment{Name: "prod", Stage: EnvironmentStageProduction},
			Source: Source{
				Git: &GitSource{
					URL: "https://example.com/repo.git",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("ConfigChange/v1"))
		Expect(m["id"]).To(Equal("evt-8"))
		Expect(m["timestamp"]).To(Equal("2026-04-10T12:00:00Z"))

		author := m["author"].(map[string]any)
		Expect(author["kind"]).To(Equal("Identity/v1"))
		Expect(author["name"]).To(Equal("alice"))

		environment := m["environment"].(map[string]any)
		Expect(environment["kind"]).To(Equal("Environment/v1"))
		Expect(environment["name"]).To(Equal("prod"))

		source := m["source"].(map[string]any)
		Expect(source["kind"]).To(Equal("Source/v1"))
		git := source["git"].(map[string]any)
		Expect(git["kind"]).To(Equal("GitSource/v1"))

		changes := m["changes"].([]any)
		Expect(changes).To(HaveLen(1))
		change := changes[0].(map[string]any)
		Expect(change["kind"]).To(Equal("Change/v1"))
		Expect(change["path"]).To(Equal(".spec.replicas"))
	})

	It("emits nested identity kinds for GroupMembership", func() {
		data, err := json.Marshal(GroupMembership{
			Group:  Identity{ID: "g-1", Name: "platform-admins", Type: IdentityTypeGroup},
			Member: Identity{ID: "u-1", Name: "alice", Type: IdentityTypeUser},
			Action: GroupMembershipActionAdded,
			Tenant: "acme",
		})
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("GroupMembership/v1"))
		Expect(m["action"]).To(Equal(string(GroupMembershipActionAdded)))
		Expect(m["tenant"]).To(Equal("acme"))

		group := m["group"].(map[string]any)
		Expect(group["kind"]).To(Equal("Identity/v1"))
		Expect(group["name"]).To(Equal("platform-admins"))
		Expect(group["type"]).To(Equal(string(IdentityTypeGroup)))

		member := m["member"].(map[string]any)
		Expect(member["kind"]).To(Equal("Identity/v1"))
		Expect(member["name"]).To(Equal("alice"))
		Expect(member["type"]).To(Equal(string(IdentityTypeUser)))
	})

	It("omits zero-value nested struct fields from outer payloads", func() {
		data, err := json.Marshal(PipelineRun{
			Event:  Event{ID: "evt-9"},
			Status: StatusPending,
		})
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("PipelineRun/v1"))
		Expect(m["id"]).To(Equal("evt-9"))
		Expect(m).ToNot(HaveKey("environment"))
	})

	It("registers all exported structs for kind lookup", func() {
		kinds := make([]string, 0, len(configChangeDetailTypes))
		for _, candidate := range configChangeDetailTypes {
			kinds = append(kinds, candidate.Kind())
		}
		slices.Sort(kinds)

		Expect(kinds).To(Equal([]string{
			"Approval/v1",
			"Backup/v1",
			"Change/v1",
			"ConfigChange/v1",
			"DatabaseSource/v1",
			"Dimension/v1",
			"Environment/v1",
			"Event/v1",
			"GitSource/v1",
			"GroupMembership/v1",
			"HelmSource/v1",
			"Identity/v1",
			"ImageSource/v1",
			"PermissionChange/v1",
			"PipelineRun/v1",
			"Promotion/v1",
			"Restore/v1",
			"Scale/v1",
			"Screenshot/v1",
			"Source/v1",
			"Test/v1",
			"UserChange/v1",
		}))
	})

	DescribeTable("UnmarshalChangeDetails returns the matching registered type",
		func(in ConfigChangeDetail, expected any) {
			raw, err := json.Marshal(in)
			Expect(err).ToNot(HaveOccurred())

			got, err := UnmarshalChangeDetails(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(BeAssignableToTypeOf(expected))

			reraw, err := json.Marshal(got)
			Expect(err).ToNot(HaveOccurred())
			Expect(reraw).To(MatchJSON(raw))
		},
		Entry("UserChange", UserChangeDetails{UserName: "alice"}, UserChangeDetails{}),
		Entry("Screenshot", ScreenshotDetails{URL: "https://example.com"}, ScreenshotDetails{}),
		Entry("PermissionChange", PermissionChangeDetails{RoleName: "admin"}, PermissionChangeDetails{}),
		Entry("GroupMembership", GroupMembership{
			Group:  Identity{ID: "g-1", Type: IdentityTypeGroup},
			Member: Identity{ID: "u-1", Type: IdentityTypeUser},
			Action: GroupMembershipActionRemoved,
		}, GroupMembership{}),
		Entry("Identity", Identity{Name: "alice"}, Identity{}),
		Entry("Approval", Approval{
			Event:       Event{ID: "evt-10"},
			SubmittedBy: &Identity{Name: "alice"},
			Status:      ApprovalStatusApproved,
		}, Approval{}),
		Entry("GitSource", GitSource{URL: "https://example.com/repo.git"}, GitSource{}),
		Entry("HelmSource", HelmSource{ChartName: "api"}, HelmSource{}),
		Entry("ImageSource", ImageSource{ImageName: "backend"}, ImageSource{}),
		Entry("DatabaseSource", DatabaseSource{Name: "appdb"}, DatabaseSource{}),
		Entry("Source", Source{
			Git:  &GitSource{URL: "https://example.com/repo.git"},
			Path: "deploy/app",
		}, Source{}),
		Entry("Environment", Environment{Name: "prod"}, Environment{}),
		Entry("Event", Event{ID: "evt-11"}, Event{}),
		Entry("Test", Test{
			Event: Event{ID: "evt-12"},
			Name:  "smoke",
		}, Test{}),
		Entry("Promotion", Promotion{
			Event:   Event{ID: "evt-13"},
			Version: "v1.2.3",
		}, Promotion{}),
		Entry("PipelineRun", PipelineRun{
			Event:  Event{ID: "evt-14"},
			Status: StatusRunning,
		}, PipelineRun{}),
		Entry("Change", Change{Path: ".spec.replicas", Type: "update"}, Change{}),
		Entry("ConfigChange", ConfigChange{
			Event:  Event{ID: "evt-15"},
			Author: Identity{Name: "alice"},
		}, ConfigChange{}),
		Entry("Restore", Restore{
			Event:  Event{ID: "evt-16"},
			Status: StatusCompleted,
		}, Restore{}),
		Entry("Backup", Backup{Status: StatusCompleted, Size: "4.2GB"}, Backup{}),
		Entry("Dimension", Dimension{Desired: "3"}, Dimension{}),
		Entry("Scale", Scale{Dimension: ScalingDimensionReplicas, Value: Dimension{Desired: "3"}}, Scale{}),
	)

	It("returns nil for empty and null details payloads", func() {
		got, err := UnmarshalChangeDetails(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil())

		got, err = UnmarshalChangeDetails([]byte("null"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("returns an error for an unknown kind", func() {
		_, err := UnmarshalChangeDetails([]byte(`{"kind":"Nope/v1"}`))
		Expect(err).To(MatchError(ContainSubstring("unknown config change detail kind")))
	})

	It("returns an error for invalid JSON", func() {
		_, err := UnmarshalChangeDetails([]byte(`{`))
		Expect(err).To(MatchError(ContainSubstring("decode config change details envelope")))
	})
})
