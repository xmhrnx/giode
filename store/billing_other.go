//go:build !android && !darwin

package store

func billingRequired() bool { return false }
