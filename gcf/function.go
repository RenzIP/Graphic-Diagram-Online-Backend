// Package gcf provides documentation and helpers for deploying the
// GraDiOl API to Google Cloud Functions (Gen 2).
//
// The canonical GCF entry point lives at the module root
// (../function.go — package p, function Handle).
//
// If you need to deploy from this subdirectory instead (e.g. when the
// backend is part of a monorepo), copy the root function.go here, change
// the package to `gcf`, and point --entry-point to the function in this
// package.
//
// Local testing (optional):
//
//	go run github.com/GoogleCloudPlatform/functions-framework-go/cmd/functions-framework@latest \
//	  --target=Handle \
//	  --port=8080
package gcf
