// /home/krylon/go/src/github.com/blicero/jobq/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-03 18:55:25 krylon>

package database

import "github.com/blicero/jobq/database/query"

var qDB = map[query.ID]string{
	query.JobSubmit: `
INSERT INTO job (submitted, cmd) VALUES (?, ?) RETURNING id
`,
	query.JobStart:  "UPDATE job SET started = ? WHERE id = ?",
	query.JobFinish: "UPDATE job SET ended = ? WHERE id = ?",
	query.JobGetByID: `
SELECT
	id,
	submitted,
	started,
	ended,
	exitcode,
	cmd,
	spoolout,
	spoolerr
FROM job
WHERE id = ?
`,
	query.JobGetPending: `
SELECT
	id,
	submitted,
	cmd,
	spoolout,
	spoolerr
FROM job
WHERE started IS NULL
ORDER BY submitted
LIMIT ?
`,
	query.JobGetRunning: `
SELECT
	id,
	submitted,
	cmd,
	spoolout,
	spoolerr
FROM job
WHERE started IS NOT NULL AND ended IS NULL
ORDER BY submitted
`,
	query.JobGetFinished: `
SELECT
	id,
	submitted,
	cmd,
	spoolout,
	spoolerr
FROM job
WHERE ended IS NOT NULL
ORDER BY submitted
LIMIT ?
`,
}
