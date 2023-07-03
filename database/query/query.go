// /home/krylon/go/src/github.com/blicero/jobq/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-03 18:17:07 krylon>

// Package query provides symbolic constants for the database operations
// we want to perform.
package query

//go:generate stringer -type=ID

// ID is the "enum" type for the symbolic constants representing our SQL queries.
type ID uint8

const (
	JobSubmit ID = iota
	JobStart
	JobFinish
	JobGetByID
	JobGetPending
	JobGetRunning
	JobGetFinished
)
