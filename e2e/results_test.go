package e2e

import (
	"context"
	gittypes "github.com/kluctl/kluctl/lib/git/types"
	test_utils "github.com/kluctl/kluctl/v2/e2e/test_project"
	"github.com/kluctl/kluctl/v2/pkg/results"
	"github.com/kluctl/kluctl/v2/pkg/types/result"
	"github.com/kluctl/kluctl/v2/pkg/utils/uo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func assertSummary(t *testing.T, expected result.CommandResultSummary, actual result.CommandResultSummary) {
	assert.Equal(t, expected.AppliedObjects, actual.AppliedObjects)
	assert.Equal(t, expected.NewObjects, actual.NewObjects)
	assert.Equal(t, expected.ChangedObjects, actual.ChangedObjects)
	assert.Equal(t, expected.OrphanObjects, actual.OrphanObjects)
	assert.Equal(t, expected.DeletedObjects, actual.DeletedObjects)
	assert.Equal(t, expected.Errors, actual.Errors)
	assert.Equal(t, expected.Warnings, actual.Warnings)
	assert.Equal(t, expected.TotalChanges, actual.TotalChanges)
}

func TestWriteResult(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := test_utils.NewTestProject(t)

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", nil)

	addConfigMapDeployment(p, "cm", map[string]string{
		"d1": "v1",
	}, resourceOpts{
		name:      "cm",
		namespace: p.TestSlug(),
	})
	p.KluctlMust(t, "deploy", "--yes", "-t", "test")
	assertConfigMapExists(t, k, p.TestSlug(), "cm")

	// we must ensure that at least a second passes between deployments, as otherwise command result sorting becomes
	// unstable
	b := newSecondPassedBarrier(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs, err := results.NewResultStoreSecrets(ctx, k.RESTConfig(), k.Client, false, "kluctl-results", 0, 0)
	assert.NoError(t, err)

	opts := results.ListResultSummariesOptions{
		ProjectFilter: &gittypes.ProjectKey{
			RepoKey: gittypes.ParseGitUrlMust(p.GitUrl()).RepoKey(),
		},
	}

	summaries, err := rs.ListCommandResultSummaries(opts)
	assert.NoError(t, err)
	assert.Len(t, summaries, 1)
	assertSummary(t, result.CommandResultSummary{
		AppliedObjects: 1,
		NewObjects:     1,
	}, summaries[0])

	addConfigMapDeployment(p, "cm2", nil, resourceOpts{
		name:      "cm2",
		namespace: p.TestSlug(),
	})
	p.UpdateYaml("cm/configmap-cm.yml", func(o *uo.UnstructuredObject) error {
		_ = o.SetNestedField("v2", "data", "d1")
		return nil
	}, "")

	b.Wait()
	p.KluctlMust(t, "deploy", "--yes", "-t", "test")
	assertConfigMapExists(t, k, p.TestSlug(), "cm2")

	summaries, err = rs.ListCommandResultSummaries(opts)
	assert.NoError(t, err)
	assert.Len(t, summaries, 2)
	assertSummary(t, result.CommandResultSummary{
		AppliedObjects: 2,
		NewObjects:     1,
		ChangedObjects: 1,
		TotalChanges:   1,
	}, summaries[0])

	p.UpdateDeploymentYaml("", func(o *uo.UnstructuredObject) error {
		_ = o.RemoveNestedField("deployments", 1)
		return nil
	})
	b.Wait()
	p.KluctlMust(t, "deploy", "--yes", "-t", "test")
	assertConfigMapExists(t, k, p.TestSlug(), "cm2")

	summaries, err = rs.ListCommandResultSummaries(opts)
	assert.NoError(t, err)
	assert.Len(t, summaries, 3)
	assertSummary(t, result.CommandResultSummary{
		AppliedObjects: 1,
		OrphanObjects:  1,
	}, summaries[0])

	b.Wait()
	p.KluctlMust(t, "prune", "--yes", "-t", "test")
	assertConfigMapNotExists(t, k, p.TestSlug(), "cm2")

	summaries, err = rs.ListCommandResultSummaries(opts)
	assert.NoError(t, err)
	assert.Len(t, summaries, 4)
	assertSummary(t, result.CommandResultSummary{
		DeletedObjects: 1,
	}, summaries[0])
}
