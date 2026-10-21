package main

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultAPIURL is the enum public API endpoint used when the solver config
// does not override it.
const defaultAPIURL = "api.enum.co:443"

// enumConfig is the per-Issuer solver configuration carried in the ACME solver's
// `config` block (opaque JSON to cert-manager).
type enumConfig struct {
	// APIURL is the enum public API endpoint. Defaults to api.enum.co:443.
	APIURL string `json:"apiUrl,omitempty"`
	// ProjectID is the enum project that owns the target zone.
	ProjectID string `json:"projectId"`
	// APITokenSecretRef references the Secret holding a project-scoped API token.
	APITokenSecretRef corev1.SecretKeySelector `json:"apiTokenSecretRef"`
}

func loadConfig(raw *extapi.JSON) (enumConfig, error) {
	cfg := enumConfig{}
	if raw == nil {
		return cfg, fmt.Errorf("solver config is empty")
	}
	if err := json.Unmarshal(raw.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("decode solver config: %w", err)
	}

	if cfg.ProjectID == "" {
		return cfg, fmt.Errorf("solver config: projectId is required")
	}

	if cfg.APITokenSecretRef.Name == "" || cfg.APITokenSecretRef.Key == "" {
		return cfg, fmt.Errorf("solver config: apiTokenSecretRef.name and .key are required")
	}

	return cfg, nil
}

// token reads the enum API token from the referenced Secret in the challenge's
// resource namespace.
func (s *enumSolver) token(ctx context.Context, namespace string, ref corev1.SecretKeySelector) (string, error) {
	secret, err := s.kube.CoreV1().Secrets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get secret %s/%s: %w", namespace, ref.Name, err)
	}

	data, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s has no key %q", namespace, ref.Name, ref.Key)
	}

	return string(data), nil
}
