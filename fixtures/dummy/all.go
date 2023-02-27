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
