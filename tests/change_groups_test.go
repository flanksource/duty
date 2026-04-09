package tests

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/changegroup"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

// stubProgram wraps a Go closure so a test can build an Evaluator without CEL.
// The Program value stored on GroupingRule is one of these closures.
type stubProgram struct {
	boolFn    func(changegroup.Env) (bool, error)
	stringFn  func(changegroup.Env) (string, error)
	detailsFn func(changegroup.Env) (types.GroupType, error)
}

// stubEvaluator implements changegroup.Evaluator by treating each expression
// string as an opaque key into a map that the test fills out before Validate.
type stubEvaluator struct {
	programs map[string]*stubProgram
}

func newStubEvaluator() *stubEvaluator { return &stubEvaluator{programs: map[string]*stubProgram{}} }

func (s *stubEvaluator) register(expr string, p *stubProgram) { s.programs[expr] = p }

func (s *stubEvaluator) Compile(expr string, kind changegroup.ExprKind) (changegroup.Program, error) {
	p, ok := s.programs[expr]
	if !ok {
		return nil, ginkgoError("stub evaluator: no program registered for expression %q", expr)
	}
	_ = kind
	return p, nil
}

func (s *stubEvaluator) EvalBool(p changegroup.Program, env changegroup.Env) (bool, error) {
	return p.(*stubProgram).boolFn(env)
}
func (s *stubEvaluator) EvalString(p changegroup.Program, env changegroup.Env) (string, error) {
	return p.(*stubProgram).stringFn(env)
}
func (s *stubEvaluator) EvalGroupDetails(p changegroup.Program, env changegroup.Env) (types.GroupType, error) {
	return p.(*stubProgram).detailsFn(env)
}

func ginkgoError(format string, args ...any) error {
	return &stubError{msg: sprintf(format, args...)}
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

func sprintf(format string, args ...any) string {
	// Tiny fmt shim so this file doesn't import "fmt" just for one call.
	b := make([]byte, 0, len(format)+64)
	ai := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) && format[i+1] == 'q' && ai < len(args) {
			b = append(b, '"')
			b = append(b, []byte(args[ai].(string))...)
			b = append(b, '"')
			i++
			ai++
			continue
		}
		b = append(b, format[i])
	}
	return string(b)
}

// insertChange persists a config_change row and returns it with ID populated.
func insertChange(ctx context.Context, c models.ConfigChange) models.ConfigChange {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.CreatedAt == nil {
		c.CreatedAt = lo.ToPtr(time.Now().UTC())
	}
	Expect(ctx.DB().Create(&c).Error).To(Succeed())
	return c
}

var _ = ginkgo.Describe("changegroup engine", ginkgo.Ordered, func() {
	var (
		ev     *stubEvaluator
		engine *changegroup.Engine
	)

	ginkgo.BeforeEach(func() {
		ev = newStubEvaluator()
		engine = nil
	})

	ginkgo.AfterEach(func() {
		// Clean rows we inserted so each spec is independent.
		Expect(DefaultContext.DB().Exec(`DELETE FROM config_changes WHERE source = 'test-changegroup'`).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec(`DELETE FROM change_groups WHERE source LIKE 'rule:test-%' OR source = 'explicit'`).Error).To(Succeed())
	})

	ginkgo.It("pod startup: time-bucket groups many changes on one config into one group", func() {
		// Rule: same_config + 5s bucket + @changed.
		ev.register("startup_key", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) {
				cfgID := env.Change["config_id"].(string)
				return cfgID + ":bucket0", nil
			},
		})
		ev.register("startup_details", &stubProgram{
			detailsFn: func(env changegroup.Env) (types.GroupType, error) {
				cfg, _ := uuid.Parse(env.Change["config_id"].(string))
				return types.StartupGroup{
					ConfigID:     cfg,
					Reason:       "test",
					RestartCount: len(env.Changes),
				}, nil
			},
		})
		ev.register("startup_summary", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) {
				return "pod startup burst", nil
			},
		})

		var err error
		engine, err = changegroup.New(ev, []changegroup.GroupingRule{
			{
				Name:        "test-pod-startup",
				Scope:       changegroup.Scope{Kind: changegroup.ScopeSameConfig},
				Window:      changegroup.Duration(5 * time.Second),
				ChangeTypes: []string{"UPDATE", "diff"},
				Key:         "startup_key",
				Details:     "startup_details",
				Summary:     "startup_summary",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		cfg := dummy.NginxIngressPod.ID.String()
		for i := 0; i < 5; i++ {
			c := insertChange(DefaultContext, models.ConfigChange{
				ConfigID:   cfg,
				ChangeType: "diff",
				Source:     "test-changegroup",
				Summary:    "startup event",
			})
			Expect(engine.Evaluate(DefaultContext, &c)).To(Succeed())
		}

		groups, err := query.FindChangeGroups(DefaultContext, query.ChangeGroupsSearchRequest{
			Type: types.GroupTypeStartup,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Summary).To(Equal("pod startup burst"))
		Expect(groups[0].MemberCount).To(Equal(5))

		members, err := query.GetGroupMembers(DefaultContext, groups[0].ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(members).To(HaveLen(5))
	})

	ginkgo.It("deployment fan-out: same image across different configs shares one group", func() {
		ev.register("deploy_key", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) {
				d, _ := env.Change["details"].(types.JSON)
				var m map[string]any
				if len(d) > 0 {
					_ = json.Unmarshal(d, &m)
				}
				img, _ := m["new_image"].(string)
				return img, nil
			},
		})
		ev.register("deploy_details", &stubProgram{
			detailsFn: func(env changegroup.Env) (types.GroupType, error) {
				// Aggregate config_ids from all members.
				ids := make([]uuid.UUID, 0, len(env.Changes))
				for _, m := range env.Changes {
					if id, err := uuid.Parse(m["config_id"].(string)); err == nil {
						ids = append(ids, id)
					}
				}
				d, _ := env.Change["details"].(types.JSON)
				var details map[string]any
				if len(d) > 0 {
					_ = json.Unmarshal(d, &details)
				}
				img, _ := details["new_image"].(string)
				return types.DeploymentGroup{
					Image:           img,
					Version:         "v1.2.3",
					TargetConfigIDs: ids,
				}, nil
			},
		})

		var err error
		engine, err = changegroup.New(ev, []changegroup.GroupingRule{
			{
				Name:        "test-deployment-fanout",
				Scope:       changegroup.Scope{Kind: changegroup.ScopeAll},
				Window:      changegroup.Duration(2 * time.Minute),
				ChangeTypes: []string{types.ChangeTypeDeployment},
				Key:         "deploy_key",
				Details:     "deploy_details",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		configs := []models.ConfigItem{
			dummy.EC2InstanceA,
			dummy.EC2InstanceB,
			dummy.NginxIngressPod,
		}
		for _, ci := range configs {
			details := types.JSON(`{"new_image":"registry/app:v1.2.3"}`)
			c := insertChange(DefaultContext, models.ConfigChange{
				ConfigID:   ci.ID.String(),
				ChangeType: types.ChangeTypeDeployment,
				Source:     "test-changegroup",
				Summary:    "deployed " + ci.ID.String(),
				Details:    details,
			})
			Expect(engine.Evaluate(DefaultContext, &c)).To(Succeed())
		}

		groups, err := query.FindChangeGroups(DefaultContext, query.ChangeGroupsSearchRequest{
			Type: types.GroupTypeDeployment,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].MemberCount).To(Equal(len(configs)))

		typed, err := groups[0].TypedDetails()
		Expect(err).ToNot(HaveOccurred())
		dep := typed.(types.DeploymentGroup)
		Expect(dep.Image).To(Equal("registry/app:v1.2.3"))
		Expect(dep.TargetConfigIDs).To(HaveLen(len(configs)))
	})

	ginkgo.It("temporary permission: grant then revoke close the group with duration", func() {
		ev.register("tp_key", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) {
				d, _ := env.Change["details"].(types.JSON)
				var m map[string]any
				_ = json.Unmarshal(d, &m)
				return m["user_id"].(string) + "|" + m["role_id"].(string), nil
			},
		})
		ev.register("tp_details", &stubProgram{
			detailsFn: func(env changegroup.Env) (types.GroupType, error) {
				var grant, revoke *uuid.UUID
				for _, m := range env.Changes {
					id, _ := uuid.Parse(m["id"].(string))
					switch m["change_type"].(string) {
					case types.ChangeTypePermissionAdded:
						tmp := id
						if grant == nil {
							grant = &tmp
						}
					case types.ChangeTypePermissionRemoved:
						tmp := id
						if revoke == nil {
							revoke = &tmp
						}
					}
				}
				d, _ := env.Change["details"].(types.JSON)
				var dm map[string]any
				_ = json.Unmarshal(d, &dm)
				return types.TemporaryPermissionGroup{
					UserID:         dm["user_id"].(string),
					RoleID:         dm["role_id"].(string),
					Scope:          dm["scope"].(string),
					GrantChangeID:  grant,
					RevokeChangeID: revoke,
				}, nil
			},
		})

		var err error
		engine, err = changegroup.New(ev, []changegroup.GroupingRule{
			{
				Name:        "test-temporary-permission",
				Scope:       changegroup.Scope{Kind: changegroup.ScopeByDetailsField, Field: "user_id"},
				Window:      changegroup.Duration(720 * time.Hour),
				CloseAfter:  0,
				ChangeTypes: []string{types.ChangeTypePermissionAdded, types.ChangeTypePermissionRemoved},
				Key:         "tp_key",
				Details:     "tp_details",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		cfg := dummy.EKSCluster.ID.String()
		t0 := time.Now().UTC().Add(-time.Hour)
		grant := insertChange(DefaultContext, models.ConfigChange{
			ConfigID:   cfg,
			ChangeType: types.ChangeTypePermissionAdded,
			Source:     "test-changegroup",
			Details:    types.JSON(`{"user_id":"alice","role_id":"admin","scope":"cluster-1"}`),
			CreatedAt:  &t0,
		})
		Expect(engine.Evaluate(DefaultContext, &grant)).To(Succeed())

		t1 := t0.Add(time.Hour)
		revoke := insertChange(DefaultContext, models.ConfigChange{
			ConfigID:   cfg,
			ChangeType: types.ChangeTypePermissionRemoved,
			Source:     "test-changegroup",
			Details:    types.JSON(`{"user_id":"alice","role_id":"admin","scope":"cluster-1"}`),
			CreatedAt:  &t1,
		})
		Expect(engine.Evaluate(DefaultContext, &revoke)).To(Succeed())

		groups, err := query.FindChangeGroups(DefaultContext, query.ChangeGroupsSearchRequest{
			Type: types.GroupTypeTemporaryPermission,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].MemberCount).To(Equal(2))

		typed, err := groups[0].TypedDetails()
		Expect(err).ToNot(HaveOccurred())
		tp := typed.(types.TemporaryPermissionGroup)
		Expect(tp.GrantChangeID).ToNot(BeNil())
		Expect(tp.RevokeChangeID).ToNot(BeNil())
		Expect(*tp.GrantChangeID).To(Equal(uuid.MustParse(grant.ID)))
		Expect(*tp.RevokeChangeID).To(Equal(uuid.MustParse(revoke.ID)))
	})

	ginkgo.It("explicit path: engine leaves producer-assigned group_id alone", func() {
		// Rule that would otherwise match — but change already has GroupID.
		ev.register("noop_key", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) { return "noop", nil },
		})
		ev.register("noop_details", &stubProgram{
			detailsFn: func(env changegroup.Env) (types.GroupType, error) {
				return types.CustomGroup{Fields: map[string]any{"k": "v"}}, nil
			},
		})

		var err error
		engine, err = changegroup.New(ev, []changegroup.GroupingRule{{
			Name:    "test-should-not-run",
			Scope:   changegroup.Scope{Kind: changegroup.ScopeAll},
			Window:  changegroup.Duration(time.Minute),
			Key:     "noop_key",
			Details: "noop_details",
		}})
		Expect(err).ToNot(HaveOccurred())

		gid, err := changegroup.CreateTyped(DefaultContext,
			types.CustomGroup{Fields: map[string]any{"origin": "playbook"}},
			"playbook-driven",
		)
		Expect(err).ToNot(HaveOccurred())

		c := insertChange(DefaultContext, models.ConfigChange{
			ConfigID:   dummy.EC2InstanceA.ID.String(),
			ChangeType: "Pulled",
			Source:     "test-changegroup",
			GroupID:    &gid,
		})
		Expect(engine.Evaluate(DefaultContext, &c)).To(Succeed())

		g, err := query.GetChangeGroup(DefaultContext, gid)
		Expect(err).ToNot(HaveOccurred())
		Expect(g.Source).To(Equal(models.ChangeGroupSourceExplicit))
		Expect(g.MemberCount).To(Equal(1), "explicit assign should have been counted by the 047 trigger")
	})

	ginkgo.It("re-evaluation: each attach sees growing changes list", func() {
		// Record how many members each evaluation saw.
		var observed []int
		ev.register("re_key", &stubProgram{
			stringFn: func(env changegroup.Env) (string, error) { return "single-key", nil },
		})
		ev.register("re_details", &stubProgram{
			detailsFn: func(env changegroup.Env) (types.GroupType, error) {
				observed = append(observed, len(env.Changes))
				ids := make([]uuid.UUID, 0, len(env.Changes))
				for _, m := range env.Changes {
					if id, err := uuid.Parse(m["config_id"].(string)); err == nil {
						ids = append(ids, id)
					}
				}
				return types.DeploymentGroup{Image: "x", TargetConfigIDs: ids}, nil
			},
		})

		var err error
		engine, err = changegroup.New(ev, []changegroup.GroupingRule{{
			Name:        "test-re-eval",
			Scope:       changegroup.Scope{Kind: changegroup.ScopeAll},
			Window:      changegroup.Duration(time.Minute),
			ChangeTypes: []string{types.ChangeTypeDeployment},
			Key:         "re_key",
			Details:     "re_details",
		}})
		Expect(err).ToNot(HaveOccurred())

		for _, ci := range []models.ConfigItem{dummy.EC2InstanceA, dummy.EC2InstanceB, dummy.NginxIngressPod} {
			c := insertChange(DefaultContext, models.ConfigChange{
				ConfigID:   ci.ID.String(),
				ChangeType: types.ChangeTypeDeployment,
				Source:     "test-changegroup",
			})
			Expect(engine.Evaluate(DefaultContext, &c)).To(Succeed())
		}

		Expect(observed).To(Equal([]int{1, 2, 3}))

		groups, err := query.FindChangeGroups(DefaultContext, query.ChangeGroupsSearchRequest{
			Type: types.GroupTypeDeployment,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(groups).To(HaveLen(1))
		typed, err := groups[0].TypedDetails()
		Expect(err).ToNot(HaveOccurred())
		Expect(typed.(types.DeploymentGroup).TargetConfigIDs).To(HaveLen(3))
	})
})
