package models

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// ConfigChangeUpdate represents an existing config change that should be updated
// instead of inserting a duplicate.
type ConfigChangeUpdate struct {
	Change         *ConfigChange
	CountIncrement int
	FirstInBatch   bool // First occurrence in current batch (not found in cache)
}

var ChangeCacheByFingerprint = cache.New(time.Hour, time.Hour)

func ChangeFingerprintCacheKey(configID, fingerprint string) string {
	return fmt.Sprintf("%s:%s", configID, fingerprint)
}

func InitChangeFingerprintCache(db *gorm.DB, window time.Duration) error {
	type configChangeFingerprint struct {
		ID          string
		ConfigID    string
		Fingerprint string
		CreatedAt   time.Time
	}

	var changes []configChangeFingerprint
	if err := db.Table("config_changes").
		Select("id, config_id, fingerprint, created_at").
		Where("fingerprint IS NOT NULL").
		Where(fmt.Sprintf("created_at >= NOW() - INTERVAL '%d SECOND'", int(window.Seconds()))).
		Find(&changes).Error; err != nil {
		return err
	}

	for _, c := range changes {
		if c.Fingerprint == "" {
			continue
		}

		key := ChangeFingerprintCacheKey(c.ConfigID, c.Fingerprint)
		ChangeCacheByFingerprint.Set(key, c.ID, time.Until(c.CreatedAt.Add(window)))
	}

	return nil
}

// DedupConfigChanges deduplicates config changes by (config_id, fingerprint).
// New fingerprints are returned in nonDuped for insertion; fingerprints already
// present in the cache are returned as updates with CountIncrement.
func DedupConfigChanges(window time.Duration, changes []*ConfigChange) ([]*ConfigChange, []ConfigChangeUpdate) {
	if len(changes) == 0 {
		return nil, nil
	}

	var nonDuped []*ConfigChange
	fingerprinted := map[string]ConfigChangeUpdate{}

	for _, change := range changes {
		if change.Fingerprint == nil || *change.Fingerprint == "" {
			nonDuped = append(nonDuped, change)
			continue
		}

		key := ChangeFingerprintCacheKey(change.ConfigID, *change.Fingerprint)
		if existingChangeID, ok := ChangeCacheByFingerprint.Get(key); !ok {
			ChangeCacheByFingerprint.Set(key, change.ID, window)
			fingerprinted[change.ID] = ConfigChangeUpdate{Change: change, CountIncrement: 0, FirstInBatch: true}
		} else {
			change.ID = existingChangeID.(string)
			ChangeCacheByFingerprint.Set(key, change.ID, window) // Refresh the cache expiry

			if existing, ok := fingerprinted[change.ID]; ok {
				// Preserve the original change, just increment the count
				fingerprinted[change.ID] = ConfigChangeUpdate{
					Change:         existing.Change,
					CountIncrement: existing.CountIncrement + 1,
					FirstInBatch:   existing.FirstInBatch,
				}
			} else {
				fingerprinted[change.ID] = ConfigChangeUpdate{Change: change, CountIncrement: 1, FirstInBatch: false}
			}
		}
	}

	var deduped []ConfigChangeUpdate
	for _, v := range fingerprinted {
		if v.FirstInBatch || v.CountIncrement == 0 {
			// First occurrence in the batch will be inserted
			nonDuped = append(nonDuped, v.Change)
		} else {
			deduped = append(deduped, v)
		}
	}

	return nonDuped, deduped
}
