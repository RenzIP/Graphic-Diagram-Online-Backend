//go:build ignore
// +build ignore

// Package gcf previously hosted the Google Cloud Functions (Gen 2) entry point
// for the GraDiOl API. Deployment has moved to Render via ./cmd/api (see
// render.yaml), so this file is now excluded from all builds via the "ignore"
// build tag and carries no compileable code (and no functions-framework import).
//
// To restore GCF support, remove the build tags above, re-add the original
// handleRequest/once/app.New() body, and reinstall the
// `github.com/GoogleCloudPlatform/functions-framework-go` dependency.
package gcf
