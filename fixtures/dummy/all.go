package dummy

import (
	"github.com/flanksource/duty/models"
	"gorm.io/gorm"
)

func PopulateDBWithDummyModels(gormDB *gorm.DB) error {
	var err error
	createTime := DummyCreatedAt
	for _, c := range AllDummyPeople {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyComponents {
		c.UpdatedAt = models.LocalTime(createTime)
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyComponentRelationships {
		c.UpdatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigs {
		c.CreatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigAnalysis {
		c.FirstObserved = &createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigComponentRelationships {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyIncidents {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyHypotheses {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyEvidences {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteDummyModelsFromDB(gormDB *gorm.DB) error {
	var err error

	if gormDB == nil {
		panic("yaa")
	}
	if err = gormDB.Exec(`DELETE FROM incident_histories`).Error; err != nil {
		return err
	}

	for _, c := range AllDummyEvidences {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyHypotheses {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyIncidents {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigAnalysis {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyConfigs {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for i := range AllDummyComponents {
		// We need to delete in reverse order
		elem := AllDummyComponents[len(AllDummyComponents)-1-i]
		err = gormDB.Delete(&elem).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyPeople {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}
