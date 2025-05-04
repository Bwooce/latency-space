// This file explains how to build with test mode enabled
// The setupTestMode function is defined in calculations_test.go and is only available
// when building with the 'test' tag.

//go:build ignore
// +build ignore

/*
To build with test mode enabled, use:
go build -tags test

To test with test mode enabled:
go test -tags test

Without the test tag, the production version in calculations_prod.go is used.
*/

package main