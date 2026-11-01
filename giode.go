// Package giode is a UI library built on Gio that brings the ease of
// retained-mode APIs to immediate mode: interfaces are described with
// simple composable functions, styled with CSS-like properties and
// wired up with plain Go callbacks, while raw Gio remains fully
// available for low-level customization.
//
// The package re-exports the public API of its subpackages
// (styles/properties, elements, components, router) for single-import
// ergonomics; the subpackages remain importable directly.
//
// The re-exports live in package ui, which a custom host (a GLFW
// window, see docs/low-level.md) imports instead to get the same names
// without App and Gio's window layer; reexport.go is generated from it.
package giode

//go:generate go run ./internal/cmd/reexport
//go:generate go run ./internal/cmd/skilldocs
