//go:build tools

// Package tools imports tool dependencies to keep them in go.mod.
package tools

import (
	_ "github.com/crossplane/crossplane-runtime/apis/common/v1"
	_ "github.com/crossplane/crossplane-tools/cmd/angryjet"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "pgregory.net/rapid"
	_ "sigs.k8s.io/controller-runtime/pkg/client"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
