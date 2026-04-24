package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

// GenerateKubeconfig requests a bound ServiceAccount token via the TokenRequest
// API, then writes a self-contained kubeconfig file to outPath.
//
// The generated kubeconfig embeds:
//   - the cluster's CA certificate (base64)
//   - the cluster's API server URL
//   - the ServiceAccount bearer token (valid for ttl)
func (c *Client) GenerateKubeconfig(ctx context.Context, cfg RBACConfig, ttl time.Duration, outPath string) error {
	// ---- Request a bound token ------------------------------------------
	ttlSeconds := int64(ttl.Seconds())
	treq := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &ttlSeconds,
		},
	}

	tokenResponse, err := c.clientset.CoreV1().
		ServiceAccounts(cfg.Namespace).
		CreateToken(ctx, cfg.SAName, treq, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("TokenRequest for %s/%s: %w", cfg.Namespace, cfg.SAName, err)
	}

	token := tokenResponse.Status.Token
	expiry := tokenResponse.Status.ExpirationTimestamp.Time
	fmt.Printf("  Token issued, expires at %s\n", expiry.UTC().Format(time.RFC3339))

	// ---- Extract cluster info from the admin kubeconfig -----------------
	rawConfig, err := c.kubeconfig.RawConfig()
	if err != nil {
		return fmt.Errorf("reading raw kubeconfig: %w", err)
	}

	server, caData, err := clusterInfoFromRawConfig(rawConfig)
	if err != nil {
		return fmt.Errorf("extracting cluster info: %w", err)
	}

	// ---- Build the kubeconfig struct ------------------------------------
	contextName := fmt.Sprintf("%s-readonly", cfg.SAName)
	clusterName := "target-cluster"
	userName := cfg.SAName

	kubeConfig := &api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   server,
				CertificateAuthorityData: caData,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userName: {
				Token: token,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		CurrentContext: contextName,
	}

	// ---- Serialise to YAML ---------------------------------------------
	kubeConfigBytes, err := marshalKubeconfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("serialising kubeconfig: %w", err)
	}

	// ---- Write to disk with restricted permissions ---------------------
	if err := os.WriteFile(outPath, kubeConfigBytes, 0600); err != nil {
		return fmt.Errorf("writing kubeconfig to %s: %w", outPath, err)
	}

	fmt.Printf("  Kubeconfig written to %s (mode 0600)\n", outPath)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// clusterInfoFromRawConfig extracts the API server URL and CA certificate from
// the current context of the admin kubeconfig.
func clusterInfoFromRawConfig(rawConfig api.Config) (server string, caData []byte, err error) {
	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return "", nil, fmt.Errorf("kubeconfig has no current-context set")
	}

	ctx, ok := rawConfig.Contexts[currentContext]
	if !ok {
		return "", nil, fmt.Errorf("context %q not found in kubeconfig", currentContext)
	}

	cluster, ok := rawConfig.Clusters[ctx.Cluster]
	if !ok {
		return "", nil, fmt.Errorf("cluster %q not found in kubeconfig", ctx.Cluster)
	}

	if cluster.Server == "" {
		return "", nil, fmt.Errorf("cluster %q has no server URL", ctx.Cluster)
	}

	ca := cluster.CertificateAuthorityData

	// Some kubeconfigs store the CA as a file path instead of inline data.
	if len(ca) == 0 && cluster.CertificateAuthority != "" {
		ca, err = os.ReadFile(cluster.CertificateAuthority)
		if err != nil {
			return "", nil, fmt.Errorf("reading CA file %s: %w", cluster.CertificateAuthority, err)
		}
	}

	if len(ca) == 0 {
		return "", nil, fmt.Errorf("cluster %q has no certificate authority data", ctx.Cluster)
	}

	return cluster.Server, ca, nil
}

// marshalKubeconfig converts an api.Config to YAML bytes.
// We marshal a plain map so that the output is a clean kubeconfig YAML file
// without runtime-only fields.
func marshalKubeconfig(cfg *api.Config) ([]byte, error) {
	// Build a plain map that mirrors the kubeconfig YAML schema.
	clusters := make([]map[string]interface{}, 0, len(cfg.Clusters))
	for name, c := range cfg.Clusters {
		clusters = append(clusters, map[string]interface{}{
			"name": name,
			"cluster": map[string]interface{}{
				"server":                     c.Server,
				"certificate-authority-data": base64.StdEncoding.EncodeToString(c.CertificateAuthorityData),
			},
		})
	}

	users := make([]map[string]interface{}, 0, len(cfg.AuthInfos))
	for name, u := range cfg.AuthInfos {
		users = append(users, map[string]interface{}{
			"name": name,
			"user": map[string]interface{}{
				"token": u.Token,
			},
		})
	}

	contexts := make([]map[string]interface{}, 0, len(cfg.Contexts))
	for name, ctx := range cfg.Contexts {
		contexts = append(contexts, map[string]interface{}{
			"name": name,
			"context": map[string]interface{}{
				"cluster": ctx.Cluster,
				"user":    ctx.AuthInfo,
			},
		})
	}

	raw := map[string]interface{}{
		"apiVersion":      "v1",
		"kind":            "Config",
		"current-context": cfg.CurrentContext,
		"clusters":        clusters,
		"users":           users,
		"contexts":        contexts,
	}

	return yaml.Marshal(raw)
}
