package tests

import (
	"fmt"

	"github.com/flanksource/duty/models"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/samber/lo"
)

type EqualsConfigItems struct {
	Expected []models.ConfigItem
}

func (matcher *EqualsConfigItems) Match(actual interface{}) (bool, error) {
	to, ok := actual.([]models.ConfigItem)
	if !ok {
		return false, fmt.Errorf("EqualsConfigItems must be passed []models.ConfigItem. Got\n%+s", actual)
	}

	got := lo.Map(to,
		func(i models.ConfigItem, _ int) string { return i.ID.String() })

	expected := lo.Map(matcher.Expected,
		func(i models.ConfigItem, _ int) string { return i.ID.String() })
	Expect(len(got)).To(Equal(len(expected)))
	if lo.Every(got, expected) {
		return true, nil
	}
	return false, nil
}

func (matcher *EqualsConfigItems) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %s to equal %s", actual, matcher.Expected)
}

func (matcher *EqualsConfigItems) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected %s to not equal %s", actual, matcher.Expected)
}
func EqualConfigs(expected ...models.ConfigItem) types.GomegaMatcher {
	return &EqualsConfigItems{
		Expected: expected,
	}
}
