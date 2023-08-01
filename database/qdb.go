// /home/krylon/go/src/github.com/blicero/jobq/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-31 22:38:38 krylon>

package database

import "github.com/blicero/jobq/database/query"

var qDB = map[query.ID]string{
	query.JobSubmit: `
INSERT INTO job (submitted, cmd) VALUES (?, ?) RETURNING id
`,
	query.JobStart:  "UPDATE job SET started = ?, pid = ?, spoolout = ?, spoolerr = ? WHERE id = ?",
	query.JobFinish: "UPDATE job SET ended = ?, exitcode = ? WHERE id = ?",
	query.JobGetByID: `
SELECT
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
        started,
	cmd,
        pid,
	spoolout,
	spoolerr
FROM job
WHERE started IS NOT NULL AND ended IS NULL
ORDER BY submitted
`,
	query.JobGetUnfinished: `
SELECT
	id,
	submitted,
	started,
	cmd,
        pid,
	spoolout,
        spoolerr
FROM job
WHERE ended IS NULL
ORDER BY submitted
`,
	query.JobGetFinished: `
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
WHERE ended IS NOT NULL
ORDER BY ended DESC
LIMIT ?
`,
	query.JobGetAll: `
SELECT
	id,
	submitted,
	started,
	ended,
	exitcode,
	cmd,
	spoolout,
	spoolerr,
	pid
FROM job
ORDER BY submitted
`,
	query.JobDelete: "DELETE FROM job WHERE id = ?",
}
