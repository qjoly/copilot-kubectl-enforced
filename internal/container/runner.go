package container

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VerifyImage checks the cosign keyless signature of the given image against
// the official GitHub Actions OIDC signatures produced by this project's CI
// and release workflows.
//
// Verification requires cosign to be present in PATH. If it is not installed
// the function returns an actionable error that includes the install URL and a
// hint to use --insecure-image.
//
// The certificate identity regexp matches any workflow inside this repository,
// while the OIDC issuer is pinned to GitHub Actions — ensuring only images
// signed during a legitimate CI/CD run are accepted.
func VerifyImage(image string) error {
	if _, err := exec.LookPath("cosign"); err != nil {
		return fmt.Errorf(
			"cosign is not installed — cannot verify image signature\n" +
				"  Install cosign: https://docs.sigstore.dev/cosign/system_config/installation/\n" +
				"  Or skip verification with: --insecure-image (not recommended)",
		)
	}

	fmt.Printf("  Verifying cosign signature for %s…\n", image)

	var stderr bytes.Buffer
	cmd := exec.Command("cosign", "verify",
		"--certificate-identity-regexp", "https://github.com/qjoly/kpil/",
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		image,
	)
	// Suppress verbose JSON stdout; capture stderr so we can include it in the error.
	cmd.Stdout = nil
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"image signature verification failed for %s: %w\n"+
				"  cosign output: %s\n"+
				"  Use --insecure-image to skip verification (not recommended)",
			image, err, strings.TrimSpace(stderr.String()),
		)
	}

	fmt.Println("  Signature verified.")
	return nil
}

// findDockerfile searches for a Dockerfile next to the running binary, then
// in the current working directory.
func findDockerfile() (string, error) {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "Dockerfile")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "Dockerfile")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Dockerfile not found next to the binary or in the current directory; use --build from the project root")
}
