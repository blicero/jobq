// /home/krylon/go/src/github.com/blicero/jobq/logdomain/id.go
// -*- mode: go; coding: utf-8; -*-
// Created on 17. 06. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-06-17 18:54:01 krylon>

// Package logdomain provides symbolic constants for the various parts of
// the application that log messages.
package logdomain

//go:generate stringer -type=ID

// ID is an id...
type ID uint8

// These constants represent the pieces of the application that need to log stuff.
const (
	Common ID = iota
)

// AllDomains returns a slice of all the valid values for ID.
func AllDomains() []ID {
	return []ID{
		Common,
	}
} // func AllDomains() []ID
