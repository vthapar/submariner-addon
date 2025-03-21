package util

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/stolostron/submariner-addon/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	workv1 "open-cluster-management.io/api/work/v1"
)

const (
	expectedBrokerRole    = "submariner-k8s-broker-cluster"
	expectedIPSECSecret   = "submariner-ipsec-psk"
	InstallationNamespace = "submariner-operator"
)

// on prow env, the /var/run/secrets/kubernetes.io/serviceaccount/namespace can be found.
func GetCurrentNamespace(kubeClient kubernetes.Interface, defaultNamespace string) (string, error) {
	namespace := defaultNamespace

	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		namespace = string(nsBytes)
	}

	if _, err = kubeClient.CoreV1().Namespaces().Create(context.Background(), NewNamespace(namespace),
		metav1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return "", err
	}

	return namespace, nil
}

func UpdateManagedClusterLabels(clusterClient clusterclientset.Interface, managedClusterName string, labels map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		managedCluster.Labels = labels

		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.Background(), managedCluster, metav1.UpdateOptions{})

		return err
	})
}

func CheckBrokerResources(kubeClient kubernetes.Interface, brokerNamespace string, expPresent bool) bool {
	ns, err := kubeClient.CoreV1().Namespaces().Get(context.Background(), brokerNamespace, metav1.GetOptions{})

	// The controller-runtime does not have a gc controller, so if the namespace is in terminating state we consider it deleted.
	if err == nil && ns.Status.Phase == corev1.NamespaceTerminating {
		err = errors.NewNotFound(schema.GroupResource{}, brokerNamespace)
	}

	if !checkPresence(err, expPresent) {
		return false
	}

	if !expPresent {
		return true // short circuit since the other resources are in the brokerNamespace
	}

	_, err = kubeClient.RbacV1().Roles(brokerNamespace).Get(context.Background(), expectedBrokerRole, metav1.GetOptions{})
	if !checkPresence(err, expPresent) {
		return false
	}

	_, err = kubeClient.CoreV1().Secrets(brokerNamespace).Get(context.Background(), expectedIPSECSecret, metav1.GetOptions{})

	return checkPresence(err, expPresent)
}

func CheckManifestWorks(workClient workclientset.Interface, managedClusterName string, expPresent bool, works ...string,
) (bool, []*workv1.ManifestWork) {
	actual := make([]*workv1.ManifestWork, len(works))

	for i, work := range works {
		w, err := workClient.WorkV1().ManifestWorks(managedClusterName).Get(context.Background(), work, metav1.GetOptions{})
		if !checkPresence(err, expPresent) {
			return false, nil
		}

		actual[i] = w
	}

	return true, actual
}

func checkPresence(err error, expPresent bool) bool {
	if err == nil {
		return expPresent
	} else if errors.IsNotFound(err) {
		return !expPresent
	}

	Expect(err).To(Succeed())

	return false
}

func SetupServiceAccount(kubeClient kubernetes.Interface, namespace, name string) error {
	return wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 30*time.Second, false,
		func(ctx context.Context) (bool, error) {
			// add a token secret to serviceaccount
			sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			}

			secretName := fmt.Sprintf("%s-token-%s", name, rand.String(5))

			// create a serviceaccount token secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      secretName,
					Annotations: map[string]string{
						corev1.ServiceAccountNameKey: sa.Name,
					},
				},
				Data: map[string][]byte{
					"ca.crt": []byte("test-ca"),
					"token":  []byte("test-token"),
				},
				Type: corev1.SecretTypeServiceAccountToken,
			}
			if _, err := kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
				return false, err
			}

			return true, nil
		})
}

func NewNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
}

func NewManagedCluster(name string, labels map[string]string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func NewManagedClusterSet(name string) *clusterv1beta2.ManagedClusterSet {
	return &clusterv1beta2.ManagedClusterSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewManagedClusterAddOn(namespace string) *addonv1alpha1.ManagedClusterAddOn {
	return &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.SubmarinerAddOnName,
			Namespace: namespace,
		},
		Spec: addonv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: InstallationNamespace,
		},
	}
}

func NewSubmariner(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "submariner.io/v1alpha1",
			"kind":       "Submariner",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"broker":                   "k8s",
				"brokerK8sApiServer":       "api:6443",
				"brokerK8sApiServerToken":  "token",
				"brokerK8sCA":              "ca",
				"brokerK8sRemoteNamespace": "subm-broker",
				"cableDriver":              "libreswan",
				"ceIPSecDebug":             false,
				"ceIPSecPSK":               "psk",
				"clusterCIDR":              "",
				"clusterID":                "test",
				"debug":                    false,
				"namespace":                InstallationNamespace,
				"natEnabled":               true,
				"serviceCIDR":              "",
			},
		},
	}
}

func SetSubmarinerDeployedStatus(submariner *unstructured.Unstructured) {
	submariner.Object["status"] = map[string]interface{}{
		"clusterID":  "test",
		"natEnabled": true,
		"gatewayDaemonSetStatus": map[string]interface{}{
			"mismatchedContainerImages": false,
			"status": map[string]interface{}{
				"currentNumberScheduled": int64(1),
				"desiredNumberScheduled": int64(1),
				"numberMisscheduled":     int64(0),
				"numberReady":            int64(1),
			},
		},
		"routeAgentDaemonSetStatus": map[string]interface{}{
			"mismatchedContainerImages": false,
			"status": map[string]interface{}{
				"currentNumberScheduled": int64(6),
				"desiredNumberScheduled": int64(6),
				"numberMisscheduled":     int64(0),
				"numberReady":            int64(6),
			},
		},
	}
}

func NewIntegrationTestEventRecorder(comp string) events.Recorder {
	return &IntegrationTestEventRecorder{component: comp}
}

type IntegrationTestEventRecorder struct {
	component string
}

func (r *IntegrationTestEventRecorder) ComponentName() string {
	return r.component
}

func (r *IntegrationTestEventRecorder) ForComponent(c string) events.Recorder {
	return &IntegrationTestEventRecorder{component: c}
}

func (r *IntegrationTestEventRecorder) WithContext(_ context.Context) events.Recorder {
	return r
}

func (r *IntegrationTestEventRecorder) WithComponentSuffix(suffix string) events.Recorder {
	return r.ForComponent(fmt.Sprintf("%s-%s", r.ComponentName(), suffix))
}

func (r *IntegrationTestEventRecorder) Event(reason, message string) {
	fmt.Fprintf(GinkgoWriter, "Event: [%s] %v: %v \n", r.component, reason, message)
}

func (r *IntegrationTestEventRecorder) Eventf(reason, messageFmt string, args ...interface{}) {
	r.Event(reason, fmt.Sprintf(messageFmt, args...))
}

func (r *IntegrationTestEventRecorder) Warning(reason, message string) {
	fmt.Fprintf(GinkgoWriter, "Warning: [%s] %v: %v \n", r.component, reason, message)
}

func (r *IntegrationTestEventRecorder) Warningf(reason, messageFmt string, args ...interface{}) {
	r.Warning(reason, fmt.Sprintf(messageFmt, args...))
}

func (r *IntegrationTestEventRecorder) Shutdown() {
}
