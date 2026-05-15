package models

import (
	"testing"
	"time"

	"github.com/samber/lo"
)

func TestDedupConfigChanges(t *testing.T) {
	configID := "dae6b3f5-bc26-48ac-8ad4-06e5efbb2a7d"
	abcKey := ChangeFingerprintCacheKey(configID, "abc")
	ChangeCacheByFingerprint.Set(abcKey, "8b9d2659-7a11-46ff-bdff-1c4e8964c437", time.Hour)
	defer func() {
		ChangeCacheByFingerprint.Delete(abcKey)
		ChangeCacheByFingerprint.Delete(ChangeFingerprintCacheKey(configID, "xyz"))
	}()

	changes := []*ConfigChange{
		{ID: "8b9d2659-7a11-46ff-bdff-1c4e8964c437", Fingerprint: lo.ToPtr("abc"), ConfigID: configID, Summary: "first", Count: 1},
		{ID: "new-1", Fingerprint: lo.ToPtr("abc"), ConfigID: configID, Summary: "second", Count: 1},
		{ID: "new-2", Fingerprint: lo.ToPtr("abc"), ConfigID: configID, Summary: "third", Count: 1},
		{ID: "01eda583-3f5e-4c44-851f-93ac73272b92", Fingerprint: lo.ToPtr("xyz"), ConfigID: configID, Summary: "different", Count: 1},
		{ID: "new-3", Fingerprint: lo.ToPtr("xyz"), ConfigID: configID, Summary: "different two", Count: 1},
	}

	nonDuped, deduped := DedupConfigChanges(time.Hour, changes)

	if len(nonDuped) != 1 || nonDuped[0].ID != "01eda583-3f5e-4c44-851f-93ac73272b92" {
		t.Fatalf("expected one non-duplicate xyz change, got %#v", nonDuped)
	}

	if len(deduped) != 1 {
		t.Fatalf("expected one deduped change, got %#v", deduped)
	}
	if deduped[0].Change.ID != "8b9d2659-7a11-46ff-bdff-1c4e8964c437" {
		t.Fatalf("expected existing change id, got %s", deduped[0].Change.ID)
	}
	if deduped[0].CountIncrement != 3 {
		t.Fatalf("expected count increment 3, got %d", deduped[0].CountIncrement)
	}
}
