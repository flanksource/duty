package tests

import (
	"encoding/json"
	"sync"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("update_config_item_properties_for_creator", func() {
	It("preserves user, other scraper, and legacy properties", func() {
		configID := uuid.New()
		scraperA := uuid.New()
		scraperB := uuid.New()
		person := uuid.New()
		seedConfigItemWithProperties(configID, models.ConfigItemProperties{
			{Property: types.Property{Name: "Owner", Text: "Team"}, CreatorType: models.PropertyCreatorTypePerson, CreatedBy: person.String()},
			{Property: types.Property{Name: "URL", Text: "old"}, CreatorType: models.PropertyCreatorTypeScraper, CreatedBy: scraperA.String()},
			{Property: types.Property{Name: "Runbook", Text: "rb"}, CreatorType: models.PropertyCreatorTypeScraper, CreatedBy: scraperB.String()},
			{Property: types.Property{Name: "Legacy", Text: "keep"}},
		})

		result := callUpdateConfigItemPropertiesForCreator(configID, models.PropertyCreatorTypeScraper, scraperA, types.Properties{
			{Name: "URL", Text: "new"},
			{Name: "Region", Text: "us-east-1"},
		})

		Expect(result.Changed).To(BeTrue())
		props := propertyMaps(result.Properties)
		Expect(props).To(HaveLen(5))
		Expect(findProperty(props, "Owner")).To(HaveKeyWithValue("created_by", person.String()))
		Expect(findProperty(props, "Runbook")).To(HaveKeyWithValue("created_by", scraperB.String()))
		Expect(findProperty(props, "Legacy")).To(HaveKeyWithValue("text", "keep"))
		Expect(findProperty(props, "URL")).To(SatisfyAll(HaveKeyWithValue("text", "new"), HaveKeyWithValue("created_by", scraperA.String())))
		Expect(findProperty(props, "Region")).To(SatisfyAll(HaveKeyWithValue("text", "us-east-1"), HaveKeyWithValue("created_by", scraperA.String())))
	})

	It("returns changed=false with merged properties when no update is needed", func() {
		configID := uuid.New()
		scraper := uuid.New()
		seedConfigItemWithProperties(configID, models.ConfigItemProperties{
			{Property: types.Property{Name: "URL", Text: "new"}, CreatorType: models.PropertyCreatorTypeScraper, CreatedBy: scraper.String()},
		})

		result := callUpdateConfigItemPropertiesForCreator(configID, models.PropertyCreatorTypeScraper, scraper, types.Properties{{Name: "URL", Text: "new"}})

		Expect(result.Changed).To(BeFalse())
		Expect(findProperty(propertyMaps(result.Properties), "URL")).To(HaveKeyWithValue("created_by", scraper.String()))
	})

	It("removes creator-owned properties when incoming properties are empty", func() {
		configID := uuid.New()
		scraperA := uuid.New()
		scraperB := uuid.New()
		seedConfigItemWithProperties(configID, models.ConfigItemProperties{
			{Property: types.Property{Name: "URL", Text: "old"}, CreatorType: models.PropertyCreatorTypeScraper, CreatedBy: scraperA.String()},
			{Property: types.Property{Name: "Runbook", Text: "rb"}, CreatorType: models.PropertyCreatorTypeScraper, CreatedBy: scraperB.String()},
			{Property: types.Property{Name: "Legacy", Text: "keep"}},
		})

		result := callUpdateConfigItemPropertiesForCreator(configID, models.PropertyCreatorTypeScraper, scraperA, types.Properties{})

		Expect(result.Changed).To(BeTrue())
		props := propertyMaps(result.Properties)
		Expect(findProperty(props, "URL")).To(BeNil())
		Expect(findProperty(props, "Runbook")).To(HaveKeyWithValue("created_by", scraperB.String()))
		Expect(findProperty(props, "Legacy")).To(HaveKeyWithValue("text", "keep"))
	})

	It("does not clobber concurrent updates from different scrapers", func() {
		configID := uuid.New()
		scraperA := uuid.New()
		scraperB := uuid.New()
		seedConfigItemWithProperties(configID, nil)

		var wg sync.WaitGroup
		errs := make(chan error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- callUpdateConfigItemPropertiesForCreatorErr(configID, models.PropertyCreatorTypeScraper, scraperA, types.Properties{{Name: "A", Text: "a"}})
		}()
		go func() {
			defer wg.Done()
			errs <- callUpdateConfigItemPropertiesForCreatorErr(configID, models.PropertyCreatorTypeScraper, scraperB, types.Properties{{Name: "B", Text: "b"}})
		}()
		wg.Wait()
		close(errs)
		for err := range errs {
			Expect(err).ToNot(HaveOccurred())
		}

		maps := propertyMaps(getConfigItemProperties(configID))
		Expect(findProperty(maps, "A")).To(HaveKeyWithValue("created_by", scraperA.String()))
		Expect(findProperty(maps, "B")).To(HaveKeyWithValue("created_by", scraperB.String()))
	})
})

func seedConfigItemWithProperties(id uuid.UUID, properties models.ConfigItemProperties) {
	configType := "test"
	config := "{}"
	Expect(DefaultContext.DB().Create(&models.ConfigItem{
		ID:         id,
		Type:       &configType,
		ExternalID: pq.StringArray{id.String()},
		Config:     &config,
		Properties: &properties,
	}).Error).To(Succeed())
}

func getConfigItemProperties(configID uuid.UUID) models.ConfigItemProperties {
	var propertiesJSON string
	Expect(DefaultContext.DB().Raw(`SELECT COALESCE(properties, '[]'::jsonb)::text FROM config_items WHERE id = ?`, configID).Scan(&propertiesJSON).Error).To(Succeed())

	var props models.ConfigItemProperties
	Expect(json.Unmarshal([]byte(propertiesJSON), &props)).To(Succeed())
	return props
}

func callUpdateConfigItemPropertiesForCreator(configID uuid.UUID, creatorType string, createdBy uuid.UUID, incoming types.Properties) models.UpdateConfigItemPropertiesResult {
	result, err := models.UpdateConfigItemPropertiesForCreator(DefaultContext.DB(), configID, creatorType, createdBy, incoming)
	Expect(err).ToNot(HaveOccurred())
	return result
}

func callUpdateConfigItemPropertiesForCreatorErr(configID uuid.UUID, creatorType string, createdBy uuid.UUID, incoming types.Properties) error {
	_, err := models.UpdateConfigItemPropertiesForCreator(DefaultContext.DB(), configID, creatorType, createdBy, incoming)
	return err
}

func propertyMaps(props models.ConfigItemProperties) []map[string]any {
	data, err := json.Marshal(props)
	Expect(err).ToNot(HaveOccurred())
	var result []map[string]any
	Expect(json.Unmarshal(data, &result)).To(Succeed())
	return result
}

func findProperty(props []map[string]any, name string) map[string]any {
	for _, prop := range props {
		if prop["name"] == name {
			return prop
		}
	}
	return nil
}
