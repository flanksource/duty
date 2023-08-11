package dummy

import (
	"fmt"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DummyData struct {
	People                       []models.Person
	Agents                       []models.Agent
	Topologies                   []models.Topology
	Components                   []models.Component
	ComponentRelationships       []models.ComponentRelationship
	Configs                      []models.ConfigItem
	ConfigChanges                []models.ConfigChange
	ConfigAnalyses               []models.ConfigAnalysis
	ConfigComponentRelationships []models.ConfigComponentRelationship
	Teams                        []models.Team
	Incidents                    []models.Incident
	Hypotheses                   []models.Hypothesis
	Evidences                    []models.Evidence
	Canaries                     []models.Canary
	Checks                       []models.Check
	CheckStatuses                []models.CheckStatus
	Responders                   []models.Responder
	Comments                     []models.Comment
	CheckComponentRelationships  []models.CheckComponentRelationship
}

// GenerateDummyData generates a set of dummy data.
// If randomize is true, the IDs of the data will be randomly generated.
func GenerateDummyData(randomize bool) DummyData {
	// we're appending here so we do not mutate the original slice.
	d := DummyData{
		People:                       append([]models.Person(nil), AllDummyPeople...),
		Agents:                       append([]models.Agent(nil), AllDummyAgents...),
		Topologies:                   append([]models.Topology(nil), AllDummyTopologies...),
		Components:                   append([]models.Component(nil), AllDummyComponents...),
		ComponentRelationships:       append([]models.ComponentRelationship(nil), AllDummyComponentRelationships...),
		Configs:                      append([]models.ConfigItem(nil), AllDummyConfigs...),
		ConfigChanges:                append([]models.ConfigChange(nil), AllDummyConfigChanges...),
		ConfigAnalyses:               append([]models.ConfigAnalysis(nil), AllDummyConfigAnalysis...),
		ConfigComponentRelationships: append([]models.ConfigComponentRelationship(nil), AllDummyConfigComponentRelationships...),
		Teams:                        append([]models.Team(nil), AllDummyTeams...),
		Incidents:                    append([]models.Incident(nil), AllDummyIncidents...),
		Hypotheses:                   append([]models.Hypothesis(nil), AllDummyHypotheses...),
		Evidences:                    append([]models.Evidence(nil), AllDummyEvidences...),
		Canaries:                     append([]models.Canary(nil), AllDummyCanaries...),
		Checks:                       append([]models.Check(nil), AllDummyChecks...),
		CheckStatuses:                append([]models.CheckStatus(nil), AllDummyCheckStatuses...),
		Responders:                   append([]models.Responder(nil), AllDummyResponders...),
		Comments:                     append([]models.Comment(nil), AllDummyComments...),
		CheckComponentRelationships:  append([]models.CheckComponentRelationship(nil), AllDummyCheckComponentRelationships...),
	}

	if !randomize {
		return d
	}

	for i := range d.People {
		d.People[i].ID = uuid.New()
	}

	for i := range d.Agents {
		d.Agents[i].ID = uuid.New()
	}

	for i := range d.Topologies {
		d.Topologies[i].ID = uuid.New()
	}

	for i := range d.Components {
		d.Components[i].ID = uuid.New()

		if d.Components[i].ParentId != nil {
			d.Components[i].ParentId = &d.Components[0].ID
		}
	}

	for i := range d.Configs {
		d.Configs[i].ID = uuid.New()
	}

	for i := range d.ConfigChanges {
		d.ConfigChanges[i].ID = uuid.New().String()
	}

	for i := range d.ConfigAnalyses {
		d.ConfigAnalyses[i].ID = uuid.New()
	}

	for i := range d.Teams {
		d.Teams[i].ID = uuid.New()
		d.Teams[i].CreatedBy = d.People[0].ID
	}

	for i := range d.Incidents {
		d.Incidents[i].ID = uuid.New()
		d.Incidents[i].CommanderID = &d.People[0].ID
		d.Incidents[i].CreatedBy = d.People[0].ID
	}

	for i := range d.Hypotheses {
		d.Hypotheses[i].ID = uuid.New()
		d.Hypotheses[i].IncidentID = d.Incidents[0].ID
		d.Hypotheses[i].CreatedBy = d.People[0].ID
	}

	for i := range d.Evidences {
		d.Evidences[i].ID = uuid.New()
		d.Evidences[i].CreatedBy = d.People[0].ID
		d.Evidences[i].HypothesisID = d.Hypotheses[0].ID

		if d.Evidences[i].ComponentID != nil {
			d.Evidences[i].ComponentID = &d.Components[0].ID
		}
	}

	for i := range d.Canaries {
		d.Canaries[i].ID = uuid.New()
	}

	for i := range d.Checks {
		d.Checks[i].ID = uuid.New()
		d.Checks[i].CanaryID = d.Canaries[0].ID
	}

	for i := range d.Responders {
		d.Responders[i].ID = uuid.New()
		d.Responders[i].CreatedBy = d.People[0].ID
		d.Responders[i].IncidentID = d.Incidents[0].ID

		if d.Responders[i].PersonID != nil {
			d.Responders[i].PersonID = &d.People[0].ID
		}

		if d.Responders[i].TeamID != nil {
			d.Responders[i].TeamID = &d.Teams[0].ID
		}
	}

	for i := range d.Comments {
		d.Comments[i].ID = uuid.New()
		d.Comments[i].CreatedBy = d.People[0].ID
		d.Comments[i].IncidentID = d.Incidents[0].ID
	}

	d.ComponentRelationships = nil
	d.CheckComponentRelationships = nil
	d.ConfigComponentRelationships = nil
	d.ConfigAnalyses = nil
	d.CheckStatuses = nil
	d.ConfigChanges = nil

	// for i := range d.ComponentRelationships {
	// 	d.ComponentRelationships[i].ComponentID = d.Components[0].ID
	// 	d.ComponentRelationships[i].RelationshipID = d.Components[1].ID
	// }

	return d
}

func (t *DummyData) Populate(gormDB *gorm.DB) error {
	var err error
	createTime := DummyCreatedAt
	for _, c := range t.People {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Agents {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Topologies {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Components {
		c.UpdatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ComponentRelationships {
		c.UpdatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Configs {
		c.CreatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigChanges {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigAnalyses {
		c.FirstObserved = &createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigComponentRelationships {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Teams {
		if err := gormDB.Create(&c).Error; err != nil {
			return err
		}
	}
	for _, c := range t.Incidents {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Hypotheses {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Evidences {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Canaries {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Checks {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Responders {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy responder: %w", err)
		}
	}
	for _, c := range t.Comments {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy comment: %w", err)
		}
	}
	for _, c := range t.CheckStatuses {
		// TODO: Figure out why it panics without Table
		err = gormDB.Table("check_statuses").Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.CheckComponentRelationships {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *DummyData) Delete(gormDB *gorm.DB) error {
	var err error
	if err = gormDB.Exec(`DELETE FROM incident_histories`).Error; err != nil {
		return err
	}

	for _, c := range t.Evidences {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Hypotheses {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Comments {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Responders {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Incidents {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.CheckComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigAnalyses {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Teams {
		if err := gormDB.Delete(&c).Error; err != nil {
			return err
		}
	}
	for _, c := range t.ConfigChanges {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Configs {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for i := range t.Components {
		// We need to delete in reverse order
		elem := t.Components[len(t.Components)-1-i]
		err = gormDB.Delete(&elem).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Canaries {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Topologies {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.People {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Agents {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}
