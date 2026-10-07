//go:build !linux

package main

func dropPrivileges(string) error { return nil }
