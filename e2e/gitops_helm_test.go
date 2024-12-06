package e2e

import (
	kluctlv1 "github.com/kluctl/kluctl/v2/api/v1beta1"
	test_utils "github.com/kluctl/kluctl/v2/e2e/test-utils"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

type GitOpsHelmSuite struct {
	GitopsTestSuite
}

func TestGitOpsHelm(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GitOpsHelmSuite))
}

func (suite *GitOpsHelmSuite) testHelmPull(tc helmTestCase, prePull bool) {
	g := NewWithT(suite.T())

	p, repo, err := prepareHelmTestCase(suite.T(), suite.k, tc, prePull, false, noLibrary)
	if err != nil {
		if tc.expectedPrepareError == "" {
			assert.Fail(suite.T(), "did not expect error")
		}
		return
	}

	var legacyHelmCreds []kluctlv1.HelmCredentials
	var projectCreds kluctlv1.ProjectCredentials

	if tc.argCredsId != "" {
		name := suite.createGitopsSecret(map[string]string{
			"credentialsId": tc.argCredsId,
			"username":      tc.argUsername,
			"password":      tc.argPassword,
		})
		legacyHelmCreds = append(legacyHelmCreds, kluctlv1.HelmCredentials{
			SecretRef: kluctlv1.LocalObjectReference{Name: name},
		})
	} else if tc.argCredsHost != "" {
		host := strings.ReplaceAll(tc.argCredsHost, "<host>", repo.URL.Host)
		if tc.helmType == test_utils.TestHelmRepo_Oci {
			m := map[string]string{
				"username": tc.argUsername,
				"password": tc.argPassword,
			}
			if !repo.HttpServer.TLSEnabled {
				m["plainHttp"] = "true"
			}
			if tc.argPassCA {
				m["ca"] = string(repo.HttpServer.ServerCAs)
			}
			if tc.argPassClientCert {
				m["cert"] = string(repo.HttpServer.ClientCert)
			}
			if tc.argPassClientCert {
				m["key"] = string(repo.HttpServer.ClientKey)
			}
			name := suite.createGitopsSecret(m)
			projectCreds.Oci = append(projectCreds.Oci, kluctlv1.ProjectCredentialsOci{
				Registry:   host,
				Repository: tc.argCredsPath,
				SecretRef:  kluctlv1.LocalObjectReference{Name: name},
			})
		} else if tc.helmType == test_utils.TestHelmRepo_Helm {
			m := map[string]string{
				"username": tc.argUsername,
				"password": tc.argPassword,
			}
			if tc.argPassCA {
				m["ca"] = string(repo.HttpServer.ServerCAs)
			}
			if tc.argPassClientCert {
				m["cert"] = string(repo.HttpServer.ClientCert)
			}
			if tc.argPassClientCert {
				m["key"] = string(repo.HttpServer.ClientKey)
			}
			name := suite.createGitopsSecret(m)
			projectCreds.Helm = append(projectCreds.Helm, kluctlv1.ProjectCredentialsHelm{
				Host:      host,
				Path:      tc.argCredsPath,
				SecretRef: kluctlv1.LocalObjectReference{Name: name},
			})
		} else if tc.helmType == test_utils.TestHelmRepo_Git {
			m := map[string]string{
				"username": tc.argUsername,
				"password": tc.argPassword,
			}
			name := suite.createGitopsSecret(m)
			projectCreds.Git = append(projectCreds.Git, kluctlv1.ProjectCredentialsGit{
				Host:      host,
				Path:      tc.argCredsPath,
				SecretRef: kluctlv1.LocalObjectReference{Name: name},
			})
		}
	}

	// add a fallback secret that enables plainHttp in case we have no matching creds
	if tc.helmType == test_utils.TestHelmRepo_Oci && !repo.HttpServer.TLSEnabled {
		m := map[string]string{
			"plainHttp": "true",
		}
		name := suite.createGitopsSecret(m)
		projectCreds.Oci = append(projectCreds.Oci, kluctlv1.ProjectCredentialsOci{
			Registry:  repo.URL.Host,
			SecretRef: kluctlv1.LocalObjectReference{Name: name},
		})
	}

	key := suite.createKluctlDeployment2(p, "", map[string]any{
		"namespace": p.TestSlug(),
	}, func(kd *kluctlv1.KluctlDeployment) {
		kd.Spec.Source = kluctlv1.ProjectSource{
			Git: &kluctlv1.ProjectSourceGit{
				URL: p.GitUrl(),
			},
		}
		kd.Spec.HelmCredentials = legacyHelmCreds
		kd.Spec.Credentials = projectCreds
	})

	kd := suite.waitForCommit(key, getHeadRevision(suite.T(), p))

	readinessCondition := suite.getReadiness(kd)
	g.Expect(readinessCondition).ToNot(BeNil())

	if tc.expectedReadyError == "" {
		g.Expect(readinessCondition.Status).ToNot(Equal(metav1.ConditionFalse))
	} else {
		g.Expect(readinessCondition.Status).To(Equal(metav1.ConditionFalse))
		g.Expect(readinessCondition.Message).To(ContainSubstring(tc.expectedReadyError))
	}

	if tc.expectedPrepareError == "" {
		g.Expect(kd.Status.LastDeployResult).ToNot(BeNil())
		g.Expect(readinessCondition.Status).ToNot(Equal(metav1.ConditionFalse))
		assertConfigMapExists(suite.T(), suite.k, p.TestSlug(), "test-helm1-test-chart1")
	} else {
		g.Expect(kd.Status.LastDeployResult).To(BeNil())

		g.Expect(readinessCondition.Status).To(Equal(metav1.ConditionFalse))
		g.Expect(readinessCondition.Reason).To(Equal(kluctlv1.PrepareFailedReason))
		g.Expect(kd.Status.LastPrepareError).To(ContainSubstring(tc.expectedPrepareError))
	}
}

func (suite *GitOpsHelmSuite) TestHelm() {
	for _, tc := range helmTests {
		tc := tc
		if tc.name == "dep-oci-creds-fail" {
			continue
		}
		suite.Run(tc.name, func() {
			suite.testHelmPull(tc, false)
		})
	}
}

func (suite *GitOpsHelmSuite) TestHelmPrePull() {
	for _, tc := range helmTests {
		tc := tc
		if tc.name == "dep-oci-creds-fail" {
			continue
		}
		suite.Run(tc.name, func() {
			suite.testHelmPull(tc, true)
		})
	}
}
