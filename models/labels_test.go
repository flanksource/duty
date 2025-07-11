package models

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestTrimmedLabels(t *testing.T) {
	testdata := []struct {
		whitelist string
		labels    map[string]string
		expected  map[string]string
	}{
		{
			whitelist: "app|batch.kubernetes.io/jobname|app.kubernetes.io/name;app.kubernetes.io/version|",
			labels: map[string]string{
				"app":                         "my-app",
				"batch.kubernetes.io/jobname": "my-job",
				"app.kubernetes.io/name":      "my-name",
				"app.kubernetes.io/version":   "1.0.0",
			},
			expected: map[string]string{
				"app":                       "my-app",
				"app.kubernetes.io/version": "1.0.0",
			},
		},
		{
			whitelist: "",
			labels: map[string]string{
				"app":     "my-app",
				"version": "1.0.0",
			},
			expected: map[string]string{},
		},
		{
			whitelist: "nonexistent|missing;another|notfound",
			labels: map[string]string{
				"app":     "my-app",
				"version": "1.0.0",
			},
			expected: map[string]string{},
		},
		{
			whitelist: "app;version;environment",
			labels: map[string]string{
				"app":         "my-app",
				"version":     "1.0.0",
				"environment": "prod",
				"team":        "backend",
			},
			expected: map[string]string{
				"app":         "my-app",
				"version":     "1.0.0",
				"environment": "prod",
			},
		},
		{
			whitelist: "app|batch.kubernetes.io/jobname|app.kubernetes.io/name;app.kubernetes.io/version|",
			labels:    nil,
			expected:  map[string]string{},
		},
	}

	for _, test := range testdata {
		t.Run(test.whitelist, func(t *testing.T) {
			g := gomega.NewWithT(t)

			trimmed := TrimLabels(test.whitelist, test.labels)
			g.Expect(trimmed).To(gomega.Equal(test.expected))
		})
	}
}

func TestSortedTrimmedLabels(t *testing.T) {
	testdata := []struct {
		name      string
		whitelist string
		order     string
		tags      map[string]string
		labels    map[string]string
		expected  []Label
	}{
		{
			name:      "basic",
			whitelist: "app|batch.kubernetes.io/jobname|app.kubernetes.io/name;app.kubernetes.io/version|",
			order:     `account;project;cluster;namespace;name;service;deployment;statefulset;daemonset;cronjob;pod;job;container;release;chart`,
			tags: map[string]string{
				"account": "my-account",
				"project": "my-project",
				"cluster": "my-cluster",
			},
			labels: map[string]string{
				"account":                     "my-account",
				"app":                         "my-app",
				"batch.kubernetes.io/jobname": "my-job",
				"app.kubernetes.io/name":      "my-name",
				"app.kubernetes.io/version":   "1.0.0",
				"project":                     "my-project",
				"cluster":                     "my-cluster",
			},
			expected: []Label{
				{Key: "account", Value: "my-account"},
				{Key: "project", Value: "my-project"},
				{Key: "cluster", Value: "my-cluster"},
				{Key: "app", Value: "my-app"},
				{Key: "app.kubernetes.io/version", Value: "1.0.0"},
			},
		},
		{
			name:      "with duplicates in tags and labels",
			whitelist: "app;environment;service",
			order:     `account;project;cluster;namespace;name;service;deployment;statefulset;daemonset;cronjob;pod;job;container;release;chart`,
			tags: map[string]string{
				"environment": "staging",
				"team":        "frontend",
			},
			labels: map[string]string{
				"app":         "web-app",
				"environment": "staging",
				"service":     "api",
				"version":     "2.0.0",
				"owner":       "team-alpha",
			},
			expected: []Label{
				{Key: "service", Value: "api"},
				{Key: "app", Value: "web-app"},
				{Key: "environment", Value: "staging"},
				{Key: "team", Value: "frontend"},
			},
		},
		{
			name:      "with nil tags and labels",
			whitelist: "",
			order:     `account;project;cluster;namespace;name;service;deployment;statefulset;daemonset;cronjob;pod;job;container;release;chart`,
			tags:      nil,
			labels:    nil,
			expected:  []Label{},
		},
		{
			name:      "prefix contains",
			whitelist: "",
			order:     "",
			tags: map[string]string{
				"zone":      "eu-west-1a",
				"region":    "eu-west-1",
				"zone-info": "eu-west-1",
				"env":       "prod",
				"type":      "production",
			},
			expected: []Label{
				{Key: "type", Value: "production"},
				{Key: "zone", Value: "eu-west-1a"},
			},
		},
		{
			name:      "prefix contains with more than 2 labels",
			whitelist: "",
			order:     "",
			tags: map[string]string{
				"zone":      "eu-west",
				"region":    "eu-west-1",
				"regional":  "eu-west-1a",
				"my-region": "eu-west-1a",
				"env":       "prod",
				"type":      "production",
			},
			expected: []Label{
				{Key: "my-region", Value: "eu-west-1a"},
				{Key: "type", Value: "production"},
			},
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			trimmed := sortedTrimmedLabels(test.whitelist, test.order, test.tags, test.labels)
			g.Expect(trimmed).To(gomega.Equal(test.expected))
		})
	}
}
