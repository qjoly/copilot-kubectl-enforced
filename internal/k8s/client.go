package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and the raw REST config so that
// other packages can build derived clients (e.g. for TokenRequest).
type Client struct {
	clientset  *kubernetes.Clientset
	kubeconfig clientcmd.ClientConfig
}

// NewClient builds a Client from the given kubeconfig file path.
// The kubeconfig must have cluster-admin (or equivalent) privileges because
// it is used to create ClusterRoles and ClusterRoleBindings.
func NewClient(kubeconfigPath string) (*Client, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig %q: %w", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes clientset: %w", err)
	}

	return &Client{
		clientset:  clientset,
		kubeconfig: kubeconfig,
	}, nil
}
