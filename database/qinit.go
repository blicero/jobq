// /home/krylon/go/src/github.com/blicero/jobq/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-05 20:06:05 krylon>

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
    CHECK (ended IS NULL OR (started IS NOT NULL AND started < ended)),
    CHECK (ended IS NULL OR exitcode IS NOT NULL)
) STRICT
`,
	"CREATE INDEX job_submit_idx ON job (submitted)",
	"CREATE INDEX job_end_null_idx ON job (ended IS NOT NULL)",
}
