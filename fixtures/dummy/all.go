package dummy

import (
	"fmt"

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
		c.UpdatedAt = createTime
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
	for _, c := range AllDummyConfigChanges {
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
	for _, c := range AllDummyCanaries {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyChecks {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyResponders {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy responder: %w", err)
		}
	}
	for _, c := range AllDummyComments {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy comment: %w", err)
		}
	}
	for _, c := range AllDummyCheckStatuses {
		// TODO: Figure out why it panics without Table
		err = gormDB.Table("check_statuses").Create(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteDummyModelsFromDB(gormDB *gorm.DB) error {
	var err error
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
	for _, c := range AllDummyComments {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range AllDummyResponders {
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
	for _, c := range AllDummyConfigChanges {
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
