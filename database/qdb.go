// /home/krylon/go/src/github.com/blicero/jobq/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-07-03 18:20:25 krylon>

package database

import "github.com/blicero/jobq/database/query"

var qDB = map[query.ID]string{
	query.JobSubmit: `
INSERT INTO job (submitted, cmd) VALUES (?, ?) RETURNING id
`,
}
