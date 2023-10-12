package upstream

import (
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/postq"
)

const EventPushQueueCreate = "push_queue.create"

// NewPushQueueConsumer creates a new push queue consumer
func NewPushQueueConsumer(config UpstreamConfig) postq.AsyncEventConsumer {
	return postq.AsyncEventConsumer{
		WatchEvents: []string{EventPushQueueCreate},
		Consumer:    getPushUpstreamConsumer(config),
		BatchSize:   50,
		ConsumerOption: &postq.ConsumerOption{
			NumConsumers: 5,
			ErrorHandler: func(err error) bool {
				logger.Errorf("error consuming upstream push_queue.create events: %v", err)
				time.Sleep(time.Second)
				return true
			},
		},
	}
}

// getPushUpstreamConsumer acts as an adapter to supply PushToUpstream event consumer.
func getPushUpstreamConsumer(config UpstreamConfig) func(ctx postq.Context, events postq.Events) postq.Events {
	return func(ctx postq.Context, events postq.Events) postq.Events {
		dbCtx, ok := ctx.(duty.DBContext)
		if !ok {
			return addErrorToFailedEvents(events, fmt.Errorf("invalid context type: %T. Need duty.DBContext", ctx))
		}

		return PushToUpstream(dbCtx, config, events)
	}
}

// PushToUpstream fetches records specified in events from this instance and sends them to the upstream instance.
func PushToUpstream(ctx duty.DBContext, config UpstreamConfig, events []postq.Event) []postq.Event {
	upstreamMsg := &PushData{
		AgentName: config.AgentName,
	}

	var failedEvents []postq.Event
	for _, cl := range GroupChangelogsByTables(events) {
		switch cl.TableName {
		case "topologies":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.Topologies).Error; err != nil {
				errMsg := fmt.Errorf("error fetching topologies: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "components":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.Components).Error; err != nil {
				errMsg := fmt.Errorf("error fetching components: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "canaries":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.Canaries).Error; err != nil {
				errMsg := fmt.Errorf("error fetching canaries: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "checks":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.Checks).Error; err != nil {
				errMsg := fmt.Errorf("error fetching checks: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_scrapers":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigScrapers).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_scrapers: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_analysis":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigAnalysis).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_analysis: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_changes":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigChanges).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_changes: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_items":
			if err := ctx.DB().Where("id IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigItems).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_items: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "check_statuses":
			if err := ctx.DB().Where(`(check_id, "time") IN ?`, cl.ItemIDs).Find(&upstreamMsg.CheckStatuses).Error; err != nil {
				errMsg := fmt.Errorf("error fetching check_statuses: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_component_relationships":
			if err := ctx.DB().Where("(component_id, config_id) IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigComponentRelationships).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_component_relationships: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "component_relationships":
			if err := ctx.DB().Where("(component_id, relationship_id, selector_id) IN ?", cl.ItemIDs).Find(&upstreamMsg.ComponentRelationships).Error; err != nil {
				errMsg := fmt.Errorf("error fetching component_relationships: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}

		case "config_relationships":
			if err := ctx.DB().Where("(related_id, config_id, selector_id) IN ?", cl.ItemIDs).Find(&upstreamMsg.ConfigRelationships).Error; err != nil {
				errMsg := fmt.Errorf("error fetching config_relationships: %w", err)
				failedEvents = append(failedEvents, addErrorToFailedEvents(cl.Events, errMsg)...)
			}
		}
	}

	upstreamMsg.ApplyLabels(config.LabelsMap())

	upstreamClient := NewUpstreamClient(config)
	err := upstreamClient.Push(ctx, upstreamMsg)
	if err == nil {
		return failedEvents
	}

	if len(events) == 1 {
		errMsg := fmt.Errorf("failed to push to upstream: %w", err)
		failedEvents = append(failedEvents, addErrorToFailedEvents(events, errMsg)...)
		return failedEvents
	} else {
		// Error encountered while pushing could be an SQL or Application error
		// Since we do not know which event in the bulk is failing
		// Process each event individually since upsteam.Push is idempotent
		var failedIndividualEvents []postq.Event
		for _, e := range events {
			failedIndividualEvents = append(failedIndividualEvents, PushToUpstream(ctx, config, []postq.Event{e})...)
		}

		return failedIndividualEvents
	}
}

func addErrorToFailedEvents(events []postq.Event, err error) []postq.Event {
	var failedEvents []postq.Event
	for _, e := range events {
		e.SetError(err.Error())
		failedEvents = append(failedEvents, e)
	}

	return failedEvents
}

// GroupChangelogsByTables groups the given events by the table they belong to.
func GroupChangelogsByTables(events []postq.Event) []GroupedPushEvents {
	type pushEvent struct {
		TableName string
		ItemIDs   []string
		Event     postq.Event
	}

	var pushEvents []pushEvent
	for _, cl := range events {
		tableName := cl.Properties["table"]
		var itemIDs []string
		switch tableName {
		case "component_relationships":
			itemIDs = []string{cl.Properties["component_id"], cl.Properties["relationship_id"], cl.Properties["selector_id"]}
		case "config_component_relationships":
			itemIDs = []string{cl.Properties["component_id"], cl.Properties["config_id"]}
		case "config_relationships":
			itemIDs = []string{cl.Properties["related_id"], cl.Properties["config_id"], cl.Properties["selector_id"]}
		case "check_statuses":
			itemIDs = []string{cl.Properties["check_id"], cl.Properties["time"]}
		default:
			itemIDs = []string{cl.Properties["id"]}
		}
		pe := pushEvent{
			TableName: tableName,
			ItemIDs:   itemIDs,
			Event:     cl,
		}
		pushEvents = append(pushEvents, pe)
	}

	tblPushMap := make(map[string]*GroupedPushEvents)
	var group []GroupedPushEvents
	for _, p := range pushEvents {
		if k, exists := tblPushMap[p.TableName]; exists {
			k.ItemIDs = append(k.ItemIDs, p.ItemIDs)
			k.Events = append(k.Events, p.Event)
		} else {
			gp := &GroupedPushEvents{
				TableName: p.TableName,
				ItemIDs:   [][]string{p.ItemIDs},
				Events:    []postq.Event{p.Event},
			}
			tblPushMap[p.TableName] = gp
		}
	}

	for _, v := range tblPushMap {
		group = append(group, *v)
	}
	return group
}

type GroupedPushEvents struct {
	TableName string
	ItemIDs   [][]string
	Events    postq.Events
}
