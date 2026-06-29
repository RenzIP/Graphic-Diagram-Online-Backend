//go:build ignore
// +build ignore

// This file is the legacy Google Cloud Functions (Gen 2) entry point.
// The GraDiOl backend now deploys to Render via ./cmd/api (see render.yaml),
// so the GCF entry point is no longer used and is excluded from all builds.
//
// It is kept as a reference stub (build tag "ignore") so that:
//   - it does not participate in `go build ./...` (no functions-framework import),
//   - the deployment target can be switched back to GCF later if needed.
//
// To re-enable GCF support, remove the build tags above and reinstall the
// `github.com/GoogleCloudPlatform/functions-framework-go` dependency.
package p
