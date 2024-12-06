package e2e

import (
	"github.com/getsops/sops/v3/age"
	"github.com/kluctl/kluctl/v2/e2e/test-utils"
	"github.com/kluctl/kluctl/v2/e2e/test_project"
	"github.com/kluctl/kluctl/v2/pkg/utils/uo"
	"github.com/kluctl/kluctl/v2/pkg/vars/sops_test_resources"
	"github.com/stretchr/testify/assert"
	"testing"
)

func setSopsKey(p *test_project.TestProject) {
	key, _ := sops_test_resources.TestResources.ReadFile("test-key.txt")
	p.SetEnv(age.SopsAgeKeyEnv, string(key))
}

func TestSopsVars(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := test_project.NewTestProject(t, test_project.WithUseProcess(true))
	setSopsKey(p)

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", nil)

	addConfigMapDeployment(p, "cm", map[string]string{
		"v1": "{{ test1.test2 }}",
	}, resourceOpts{
		name:      "cm",
		namespace: p.TestSlug(),
	})
	p.UpdateDeploymentYaml("", func(o *uo.UnstructuredObject) error {
		_ = o.SetNestedField([]map[string]any{
			{
				"file": "encrypted-vars.yaml",
			},
		}, "vars")
		return nil
	})

	p.UpdateFile("encrypted-vars.yaml", func(f string) (string, error) {
		b, _ := sops_test_resources.TestResources.ReadFile("test.yaml")
		return string(b), nil
	}, "")

	p.KluctlMust(t, "deploy", "--yes", "-t", "test")

	cm := assertConfigMapExists(t, k, p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, map[string]any{
		"v1": "42",
	}, "data")
}

func TestSopsResources(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := test_project.NewTestProject(t, test_project.WithUseProcess(true))
	setSopsKey(p)

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", nil)
	p.UpdateDeploymentYaml("", func(o *uo.UnstructuredObject) error {
		_ = o.SetNestedField(p.TestSlug(), "overrideNamespace")
		return nil
	})

	p.AddKustomizeDeployment("cm", []test_project.KustomizeResource{
		{Name: "encrypted-cm.yaml"},
	}, nil)

	p.UpdateFile("cm/encrypted-cm.yaml", func(f string) (string, error) {
		b, _ := sops_test_resources.TestResources.ReadFile("test-configmap.yaml")
		return string(b), nil
	}, "")

	p.KluctlMust(t, "deploy", "--yes", "-t", "test")

	cm := assertConfigMapExists(t, k, p.TestSlug(), "encrypted-cm")
	assertNestedFieldEquals(t, cm, map[string]any{
		"a": "b",
	}, "data")
}

func TestSopsHelmValues(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := test_project.NewTestProject(t, test_project.WithUseProcess(true))
	setSopsKey(p)

	createNamespace(t, k, p.TestSlug())

	charts := []test_utils.RepoChart{
		{ChartName: "test-chart1", Version: "0.1.0"},
	}
	repo := test_utils.NewHelmTestRepo(test_utils.TestHelmRepo_Oci, "", charts)

	repo.Start(t)

	valuesBytes, err := sops_test_resources.TestResources.ReadFile("helm-values.yaml")
	assert.NoError(t, err)
	values1, err := uo.FromString(string(valuesBytes))
	assert.NoError(t, err)

	p.UpdateTarget("test", nil)
	p.AddHelmDeployment("helm1", repo, "test-chart1", "0.1.0", "test-helm1", p.TestSlug(), values1.Object)
	p.UpdateYaml("helm1/helm-chart.yaml", func(o *uo.UnstructuredObject) error {
		_ = o.SetNestedField(true, "helmChart", "skipPrePull")
		return nil
	}, "")

	p.KluctlMust(t, "deploy", "--yes", "-t", "test")

	cm1 := assertConfigMapExists(t, k, p.TestSlug(), "test-helm1-test-chart1")

	assert.Equal(t, map[string]any{
		"a":           "secret1",
		"b":           "secret2",
		"version":     "0.1.0",
		"kubeVersion": k.ServerVersion.String(),
	}, cm1.Object["data"])
}
