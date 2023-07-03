// /home/krylon/go/src/github.com/blicero/jobq/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-03 18:15:05 krylon>

package database

var qInit = []string{
	`
CREATE TABLE job (
    id		INTEGER PRIMARY KEY,
    submitted	INTEGER NOT NULL,
    started	INTEGER,
    ended	INTEGER,
    exitcode    INTEGER,
    cmd         TEXT NOT NULL,
    spoolout    TEXT UNIQUE,
    spoolerr    TEXT UNIQUE,
) STRICT
`,
	"CREATE INDEX job_submit_idx ON job (submitted)",
}
