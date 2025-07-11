package models

import (
	"sort"
	"strings"

	"github.com/flanksource/commons/properties"
	"github.com/samber/lo"
)

func init() {
	properties.RegisterListener(func(p *properties.Properties) {
		if v := p.String("notifications.labels.whitelist", ""); v != "" {
			defaultLabelsWhitelist = v
		}

		if v := p.String("notifications.labels.order", ""); v != "" {
			defaultLabelsOrder = v
		}
	})
}

var (
	defaultLabelsOrder     = `account;project;cluster;namespace;name;service;deployment;statefulset;daemonset;cronjob;pod;job;container;release;chart`
	defaultLabelsWhitelist = `app|batch.kubernetes.io/jobname|app.kubernetes.io/name|kustomize.toolkit.fluxcd.io/name;app.kubernetes.io/version`
)

// TrimLabels returns a subset of labels that match the whitelist.
// The whitelist contains a set of groups separated by semicolons
// with group members separated by pipes.
func TrimLabels(whitelist string, labels map[string]string) map[string]string {
	groups := strings.Split(whitelist, ";")
	matchedLabels := make(map[string]string)

	for _, group := range groups {
		for key := range strings.SplitSeq(group, "|") {
			key = strings.TrimSpace(key)
			if val, ok := labels[key]; ok {
				matchedLabels[key] = val
				break // move to the next group
			}
		}
	}

	return matchedLabels
}

type Label struct {
	Key   string
	Value string
}

func deduplicateByValuePrefix(labels []Label) []Label {
	if len(labels) <= 1 {
		return labels
	}

	// Sort in descending order of value length
	sort.Slice(labels, func(i, j int) bool {
		if len(labels[i].Value) == len(labels[j].Value) {
			return labels[i].Key < labels[j].Key
		}

		return len(labels[i].Value) > len(labels[j].Value)
	})

	labelsToSkip := make(map[int]struct{})
	for i, label := range labels {
		if _, ok := labelsToSkip[i]; ok {
			continue
		}

		for j := i + 1; j < len(labels); j++ {
			if strings.HasPrefix(label.Value, labels[j].Value) {
				labelsToSkip[j] = struct{}{}
			}
		}
	}

	result := lo.Filter(labels, func(_ Label, i int) bool {
		_, ok := labelsToSkip[i]
		return !ok
	})

	return result
}

func sortedTrimmedLabels(whitelist, order string, tags, labels map[string]string) []Label {
	output := make([]Label, 0, len(labels)+len(tags))
	for k, v := range tags {
		output = append(output, Label{Key: k, Value: v})
	}
	output = deduplicateByValuePrefix(output)

	trimmed := TrimLabels(whitelist, labels)
	for k, v := range trimmed {
		output = append(output, Label{Key: k, Value: v})
	}

	output = lo.UniqBy(output, func(l Label) string {
		return l.Key
	})

	// Must sort by the given order and then alphabetically
	orderList := strings.Split(order, ";")
	sort.Slice(output, func(i, j int) bool {
		orderI := lo.IndexOf(orderList, output[i].Key)
		orderJ := lo.IndexOf(orderList, output[j].Key)

		if orderI == -1 && orderJ == -1 {
			// If the order isn't specified for both, sort alphabetically
			return output[i].Key < output[j].Key
		} else if orderI == -1 {
			return false
		} else if orderJ == -1 {
			return true
		}

		return orderI < orderJ
	})

	return output
}
