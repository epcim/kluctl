package e2e

import (
	"fmt"
	"github.com/kluctl/kluctl/v2/e2e/test-utils"
	"github.com/kluctl/kluctl/v2/pkg/utils/uo"
	"os"
	"testing"
)

func TestArgs(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := test_utils.NewTestProject(t)

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", func(target *uo.UnstructuredObject) {
	})

	args := []any{
		map[string]any{
			"name": "a",
		},
		map[string]any{
			"name":    "b",
			"default": "default",
		},
		map[string]any{
			"name": "d",
			"default": map[string]any{
				"nested": "default",
			},
		},
		map[string]any{
			"name":    "e",
			"default": 42,
		},
	}

	p.UpdateKluctlYaml(func(o *uo.UnstructuredObject) error {
		_ = o.SetNestedField(args, "args")
		return nil
	})

	addConfigMapDeployment(p, "cm", map[string]string{
		"a": `{{ args.a | default("na") }}`,
		"b": `{{ args.b | default("na") }}`,
		"c": `{{ args.c | default("na") }}`,
		"d": "{{ args.d | to_json }}",
		"e": "{{ args.e + 1 }}",
	}, resourceOpts{
		name:      "cm",
		namespace: p.TestSlug(),
	})

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a")
	cm := k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "default", "data", "b")
	assertNestedFieldEquals(t, cm, "na", "data", "c")
	assertNestedFieldEquals(t, cm, `{"nested": "default"}`, "data", "d")
	assertNestedFieldEquals(t, cm, `43`, "data", "e")

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a", "-ab=b")
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "b", "data", "b")
	assertNestedFieldEquals(t, cm, "na", "data", "c")
	assertNestedFieldEquals(t, cm, `{"nested": "default"}`, "data", "d")

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a", "-ab=b", "-ac=c")
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "b", "data", "b")
	assertNestedFieldEquals(t, cm, "c", "data", "c")
	assertNestedFieldEquals(t, cm, `{"nested": "default"}`, "data", "d")

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a", "-ab=b", "-ac=c", "-ad.nested=d")
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, `{"nested": "d"}`, "data", "d")

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a", "-ab=b", "-ac=c", `-ad={"nested": "d2"}`)
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, `{"nested": "d2"}`, "data", "d")

	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString(`
nested:
  nested2: d3
`)

	p.KluctlMust("deploy", "--yes", "-t", "test", "-aa=a", "-ab=b", "-ac=c", fmt.Sprintf(`-ad=@%s`, tmpFile.Name()))
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, `{"nested": {"nested2": "d3"}}`, "data", "d")

	_ = tmpFile.Truncate(0)
	_, _ = tmpFile.Seek(0, 0)
	_, _ = tmpFile.WriteString(`
a: a2
c: c2
d:
  nested:
    nested2: d4
`)

	p.KluctlMust("deploy", "--yes", "-t", "test", fmt.Sprintf(`--args-from-file=%s`, tmpFile.Name()))
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a2", "data", "a")
	assertNestedFieldEquals(t, cm, "default", "data", "b")
	assertNestedFieldEquals(t, cm, "c2", "data", "c")
	assertNestedFieldEquals(t, cm, `{"nested": {"nested2": "d4"}}`, "data", "d")
}

func TestArgsFromEnv(t *testing.T) {
	k := defaultCluster1

	p := test_utils.NewTestProject(t, test_utils.WithUseProcess(true))
	p.AddExtraEnv("KLUCTL_ARG=a=a")
	p.AddExtraEnv("KLUCTL_ARG_1=b=b")
	p.AddExtraEnv(`KLUCTL_ARG_2=c={"nested":{"nested2":"c"}}`)
	p.AddExtraEnv("KLUCTL_ARG_3=d=true")
	p.AddExtraEnv("KLUCTL_ARG_4=e='true'")

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", func(target *uo.UnstructuredObject) {
	})

	addConfigMapDeployment(p, "cm", map[string]string{
		"a": `{{ args.a }}`,
		"b": `{{ args.b }}`,
		"c": `{{ args.c | to_json }}`,
		"d": `{{ args.d }}`,
		"e": `{{ args.e }}`,
	}, resourceOpts{
		name:      "cm",
		namespace: p.TestSlug(),
	})

	p.KluctlMust("deploy", "--yes", "-t", "test")
	cm := k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "b", "data", "b")
	assertNestedFieldEquals(t, cm, `{"nested": {"nested2": "c"}}`, "data", "c")
	assertNestedFieldEquals(t, cm, "True", "data", "d")
	assertNestedFieldEquals(t, cm, "true", "data", "e")
}

func TestArgsFromEnvAndCli(t *testing.T) {
	k := defaultCluster1

	p := test_utils.NewTestProject(t, test_utils.WithUseProcess(true))
	p.AddExtraEnv("KLUCTL_ARG_1=a=a")
	p.AddExtraEnv("KLUCTL_ARG_2=c=c")

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", func(target *uo.UnstructuredObject) {
	})

	addConfigMapDeployment(p, "cm", map[string]string{
		"a": `{{ args.a }}`,
		"b": `{{ args.b }}`,
		"c": `{{ args.c }}`,
	}, resourceOpts{
		name:      "cm",
		namespace: p.TestSlug(),
	})

	p.KluctlMust("deploy", "--yes", "-t", "test", "-a", "b=b")
	cm := k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "b", "data", "b")
	assertNestedFieldEquals(t, cm, "c", "data", "c")

	// make sure the CLI overrides values from env
	p.KluctlMust("deploy", "--yes", "-t", "test", "-a", "b=b", "-a", "c=c2")
	cm = k.MustGetCoreV1(t, "configmaps", p.TestSlug(), "cm")
	assertNestedFieldEquals(t, cm, "a", "data", "a")
	assertNestedFieldEquals(t, cm, "b", "data", "b")
	assertNestedFieldEquals(t, cm, "c2", "data", "c")
}
