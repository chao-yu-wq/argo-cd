package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//"github.com/argoproj/argo-cd/v3/common"
	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-cd/v3/test/e2e/fixture"
	"github.com/argoproj/argo-cd/v3/util/errors"
)

type ExternalNamespace string

const (
	// ArgoCDNamespace is the namespace into which Argo CD and ApplicationSet controller are deployed,
	// and in which Application resources should be created.
	ArgoCDNamespace = "argocd-e2e"

	// ArgoCDExternalNamespace is an external namespace to test additional namespaces
	ArgoCDExternalNamespace ExternalNamespace = "argocd-e2e-external"

	// ArgoCDExternalNamespace2 is an external namespace to test additional namespaces
	ArgoCDExternalNamespace2 ExternalNamespace = "argocd-e2e-external-2"

	// ApplicationsResourcesNamespace is the namespace into which temporary resources (such as Deployments/Pods/etc)
	// can be deployed, such as using it as the target namespace in an Application resource.
	// Note: this is NOT the namespace the ApplicationSet controller is deployed to; see ArgoCDNamespace.
	ApplicationsResourcesNamespace = "applicationset-e2e"

	TmpDir       = "/tmp/applicationset-e2e"
	TestingLabel = "e2e.argoproj.io"
)

var (
	id string

	// call GetClientVars() to retrieve the Kubernetes client data for E2E test fixtures
	clientInitialized  sync.Once
	internalClientVars *E2EFixtureK8sClient
)

// E2EFixtureK8sClient contains Kubernetes clients initialized from local k8s configuration
type E2EFixtureK8sClient struct {
	KubeClientset            kubernetes.Interface
	DynamicClientset         dynamic.Interface
	AppClientset             appclientset.Interface
	AppSetClientset          dynamic.ResourceInterface
	ExternalAppSetClientsets map[ExternalNamespace]dynamic.ResourceInterface
}

func GetEnvWithDefault(envName, defaultValue string) string {
	r := os.Getenv(envName)
	if r == "" {
		return defaultValue
	}
	return r
}

// TestNamespace returns the namespace where Argo CD E2E test instance will be
// running in.
func TestNamespace() string {
	return GetEnvWithDefault("ARGOCD_E2E_NAMESPACE", ArgoCDNamespace)
}

// GetE2EFixtureK8sClient initializes the Kubernetes clients (if needed), and returns the most recently initialized value.
// Note: this requires a local Kubernetes configuration (for example, while running the E2E tests).
func GetE2EFixtureK8sClient(t *testing.T) *E2EFixtureK8sClient {
	t.Helper()
	// Initialize the Kubernetes clients only on first use
	clientInitialized.Do(func() {
		// set-up variables
		config := getKubeConfig(t, "", clientcmd.ConfigOverrides{})

		internalClientVars = &E2EFixtureK8sClient{
			AppClientset:     appclientset.NewForConfigOrDie(config),
			DynamicClientset: dynamic.NewForConfigOrDie(config),
			KubeClientset:    kubernetes.NewForConfigOrDie(config),
		}

		internalClientVars.AppSetClientset = internalClientVars.DynamicClientset.Resource(v1alpha1.SchemeGroupVersion.WithResource("applicationsets")).Namespace(TestNamespace())
		internalClientVars.ExternalAppSetClientsets = map[ExternalNamespace]dynamic.ResourceInterface{
			ArgoCDExternalNamespace:  internalClientVars.DynamicClientset.Resource(v1alpha1.SchemeGroupVersion.WithResource("applicationsets")).Namespace(string(ArgoCDExternalNamespace)),
			ArgoCDExternalNamespace2: internalClientVars.DynamicClientset.Resource(v1alpha1.SchemeGroupVersion.WithResource("applicationsets")).Namespace(string(ArgoCDExternalNamespace2)),
		}
	})
	return internalClientVars
}

// EnsureCleanSlate ensures that the Kubernetes resources on the cluster are in a 'clean' state, before a test is run.
func EnsureCleanState(t *testing.T) {
	t.Helper()
	start := time.Now()

	fixtureClient := GetE2EFixtureK8sClient(t)

	policy := metav1.DeletePropagationForeground

	fixture.RunFunctionsInParallelAndCheckErrors(t, []func() error{
		/*func() error {
			// kubectl delete secrets -l argocd.argoproj.io/secret-type=repository
			return fixtureClient.KubeClientset.CoreV1().Secrets(TestNamespace()).DeleteCollection(
				t.Context(),
				metav1.DeleteOptions{PropagationPolicy: &policy},
				metav1.ListOptions{LabelSelector: common.LabelKeySecretType + "=" + common.LabelValueSecretTypeRepository})
		},
		func() error {
			// kubectl delete secrets -l argocd.argoproj.io/secret-type=repo-creds
			return fixtureClient.KubeClientset.CoreV1().Secrets(TestNamespace()).DeleteCollection(
				t.Context(),
				metav1.DeleteOptions{PropagationPolicy: &policy},
				metav1.ListOptions{LabelSelector: common.LabelKeySecretType + "=" + common.LabelValueSecretTypeRepoCreds})
		},*/
		/*func() error {
			// Delete the applicationset-e2e namespace, if it exists
			err := fixtureClient.KubeClientset.CoreV1().Namespaces().Delete(t.Context(), ApplicationsResourcesNamespace, metav1.DeleteOptions{PropagationPolicy: &policy})
			if err != nil && !apierrors.IsNotFound(err) { // 'not found' error is expected
				return err
			}
			return nil
		},*/
		/*func() error {
			// Delete the argocd-e2e-external namespace, if it exists
			err := fixtureClient.KubeClientset.CoreV1().Namespaces().Delete(t.Context(), string(ArgoCDExternalNamespace), metav1.DeleteOptions{PropagationPolicy: &policy})
			if err != nil && !apierrors.IsNotFound(err) { // 'not found' error is expected
				return err
			}
			return nil
		},
		func() error {
			// Delete the argocd-e2e-external namespace, if it exists
			err := fixtureClient.KubeClientset.CoreV1().Namespaces().Delete(t.Context(), string(ArgoCDExternalNamespace2), metav1.DeleteOptions{PropagationPolicy: &policy})
			if err != nil && !apierrors.IsNotFound(err) { // 'not found' error is expected
				return err
			}
			return nil
		},*/
		// delete resources
		func() error {
			// kubectl delete applicationsets --all
			return fixtureClient.AppSetClientset.DeleteCollection(t.Context(), metav1.DeleteOptions{PropagationPolicy: &policy}, metav1.ListOptions{})
		},
		func() error {
			// kubectl delete apps --all
			return fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(TestNamespace()).DeleteCollection(t.Context(), metav1.DeleteOptions{PropagationPolicy: &policy}, metav1.ListOptions{})
		},
		func() error {
			// kubectl delete secrets -l e2e.argoproj.io=true
			return fixtureClient.KubeClientset.CoreV1().Secrets(TestNamespace()).DeleteCollection(
				t.Context(),
				metav1.DeleteOptions{PropagationPolicy: &policy},
				metav1.ListOptions{LabelSelector: TestingLabel + "=true"})
		},
	})

	// First we wait up to 30 seconds for all the ApplicationSets to delete, but we don't fail if they don't.
	// Why? We want to give Argo CD time to delete the Application's child resources, before we remove the finalizers below.
	_ = waitForSuccess(func() error {
		list, err := fixtureClient.AppSetClientset.List(t.Context(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if list != nil && len(list.Items) > 0 {
			// Fail
			return fmt.Errorf("waiting for list of ApplicationSets to be size zero: %d", len(list.Items))
		}

		return nil // Pass
	}, time.Now().Add(30*time.Second))

	// Remove finalizers from Argo CD Application resources in the namespace
	err := waitForSuccess(func() error {
		appList, err := fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(TestNamespace()).List(t.Context(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, app := range appList.Items {
			t.Log("Removing finalizer for: ", app.Name)
			app.Finalizers = []string{}
			_, err := fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(TestNamespace()).Update(t.Context(), &app, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	}, time.Now().Add(120*time.Second))
	require.NoError(t, err)

	require.NoError(t, waitForExpectedClusterState(t))

	// remove tmp dir
	require.NoError(t, os.RemoveAll(TmpDir))

	// create tmp dir
	errors.NewHandler(t).FailOnErr(Run("", "mkdir", "-p", TmpDir))

	// We can switch user and as result in previous state we will have non-admin user, this case should be reset
	require.NoError(t, fixture.LoginAs("admin"))

	log.WithFields(log.Fields{"duration": time.Since(start), "name": t.Name(), "id": id, "username": "admin", "password": "password"}).Info("clean state")
}

func waitForExpectedClusterState(t *testing.T) error {
	t.Helper()
	fixtureClient := GetE2EFixtureK8sClient(t)

	SetProjectSpec(t, fixtureClient, "default", v1alpha1.AppProjectSpec{
		OrphanedResources:        nil,
		SourceRepos:              []string{"*"},
		Destinations:             []v1alpha1.ApplicationDestination{{Namespace: "*", Server: "*"}},
		ClusterResourceWhitelist: []metav1.GroupKind{{Group: "*", Kind: "*"}},
		SourceNamespaces:         []string{string(ArgoCDExternalNamespace), string(ArgoCDExternalNamespace2)},
	})

	// Wait up to 60 seconds for all the ApplicationSets to delete
	if err := waitForSuccess(func() error {
		list, err := fixtureClient.AppSetClientset.List(t.Context(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if list != nil && len(list.Items) > 0 {
			// Fail
			return fmt.Errorf("waiting for list of ApplicationSets to be size zero: %d", len(list.Items))
		}

		return nil // Pass
	}, time.Now().Add(60*time.Second)); err != nil {
		return err
	}

	// Wait up to 60 seconds for all the Applications to delete
	if err := waitForSuccess(func() error {
		appList, err := fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(TestNamespace()).List(t.Context(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if appList != nil && len(appList.Items) > 0 {
			// Fail
			return fmt.Errorf("waiting for list of Applications to be size zero: %d", len(appList.Items))
		}
		return nil // Pass
	}, time.Now().Add(60*time.Second)); err != nil {
		return err
	}

	// Wait up to 120 seconds for namespace to not exist
	/*for _, namespace := range []string{string(ApplicationsResourcesNamespace), string(ArgoCDExternalNamespace), string(ArgoCDExternalNamespace2)} {
		// Wait up to 120 seconds for namespace to not exist
		if err := waitForSuccess(func() error {
			return cleanUpNamespace(fixtureClient, namespace)
		}, time.Now().Add(120*time.Second)); err != nil {
			return err
		}
	}*/

	return nil
}

func SetProjectSpec(t *testing.T, fixtureClient *E2EFixtureK8sClient, project string, spec v1alpha1.AppProjectSpec) {
	t.Helper()
	proj, err := fixtureClient.AppClientset.ArgoprojV1alpha1().AppProjects(TestNamespace()).Get(t.Context(), project, metav1.GetOptions{})
	require.NoError(t, err)
	proj.Spec = spec
	_, err = fixtureClient.AppClientset.ArgoprojV1alpha1().AppProjects(TestNamespace()).Update(t.Context(), proj, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func cleanUpNamespace(fixtureClient *E2EFixtureK8sClient, namespace string) error {
	_, err := fixtureClient.KubeClientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})

	msg := ""

	if err == nil {
		msg = fmt.Sprintf("namespace '%s' still exists, after delete", namespace)
	}

	if msg == "" && err != nil && apierrors.IsNotFound(err) {
		// Success is an error containing 'applicationset-e2e' not found.
		return nil
	}

	if msg == "" {
		msg = err.Error()
	}

	return fmt.Errorf("%s", msg)
}

// waitForSuccess waits for the condition to return a non-error value.
// Returns if condition returns nil, or the expireTime has elapsed (in which
// case the last error will be returned)
func waitForSuccess(condition func() error, expireTime time.Time) error {
	var mostRecentError error

	sleepIntervals := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}
	sleepIntervalsIdx := -1

	for !time.Now().After(expireTime) {
		conditionErr := condition()
		if conditionErr == nil {
			// Pass!
			mostRecentError = nil
			break
		}
		// Fail!
		mostRecentError = conditionErr

		// Wait on fail
		if sleepIntervalsIdx < len(sleepIntervals)-1 {
			sleepIntervalsIdx++
		}
		time.Sleep(sleepIntervals[sleepIntervalsIdx])
	}
	return mostRecentError
}

// getKubeConfig creates new kubernetes client config using specified config path and config overrides variables
func getKubeConfig(t *testing.T, configPath string, overrides clientcmd.ConfigOverrides) *rest.Config {
	t.Helper()
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = configPath
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)

	restConfig, err := clientConfig.ClientConfig()
	require.NoError(t, err)
	return restConfig
}

// creates e2e tests fixture: ensures that Application CRD is installed, creates temporal namespace, starts repo and api server,
// configure currently available cluster.
func init() {
	// ensure we log all shell execs
	log.SetLevel(log.DebugLevel)
}

// PrettyPrintJson is a utility function for debugging purposes
func PrettyPrintJson(obj any) string { //nolint:revive //FIXME(var-naming)
	bytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

// returns dns friends string which is no longer than 63 characters and has specified postfix at the end
func DnsFriendly(str string, postfix string) string { //nolint:revive //FIXME(var-naming)
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	str = matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	str = matchAllCap.ReplaceAllString(str, "${1}-${2}")
	str = strings.ToLower(str)

	if diff := len(str) + len(postfix) - 63; diff > 0 {
		str = str[:len(str)-diff]
	}
	return str + postfix
}

func MustToUnstructured(obj any) *unstructured.Unstructured {
	uObj, err := ToUnstructured(obj)
	if err != nil {
		panic(err)
	}
	return uObj
}

// ToUnstructured converts a concrete K8s API type to an unstructured object
func ToUnstructured(obj any) (*unstructured.Unstructured, error) {
	uObj, err := runtime.NewTestUnstructuredConverter(equality.Semantic).ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: uObj}, nil
}

// IsGitHubSkippedTest returns true if the test should be skipped because it requires a GitHub API Token
// and one has not been provided.
// Unfortunately, GitHub Actions cannot use repository secrets, so we need to skip these tests for PRs.
//
// Tests that call this function require a GITHUB_TOKEN to be present, otherwise they will fail, due to
// GitHub's rate limiting on anonymous API requests.
//
// Note: This only applies to tests that use the GitHub API (different from GitHub's Git service)
func IsGitHubAPISkippedTest(t *testing.T) bool {
	t.Helper()
	if strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) == "" {
		t.Skip("Skipping this test, as the GITHUB_TOKEN is not set. Please ensure this test passes locally, with your own GITHUB_TOKEN.")
		return true
	}

	return false
}
