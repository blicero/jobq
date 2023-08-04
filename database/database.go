// /home/krylon/go/src/github.com/blicero/jobq/database/database.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2023 by Benjamin Walkenhorst
// (c) 2023 Benjamin Walkenhorst
// Time-stamp: <2023-08-03 17:52:19 krylon>

// Package database provides the persistence layer for jobs.
// It is a wrapper around an SQLite database, exposing the operations required
// by the application.
package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/jobq/common"
	"github.com/blicero/jobq/database/query"
	"github.com/blicero/jobq/job"
	"github.com/blicero/jobq/logdomain"
	"github.com/blicero/krylib"
	_ "github.com/mattn/go-sqlite3" // Import the database driver
)

var (
	openLock sync.Mutex
	idCnt    int64
)

// ErrTxInProgress indicates that an attempt to initiate a transaction failed
// because there is already one in progress.
var ErrTxInProgress = errors.New("A Transaction is already in progress")

// ErrNoTxInProgress indicates that an attempt was made to finish a
// transaction when none was active.
var ErrNoTxInProgress = errors.New("There is no transaction in progress")

// ErrEmptyUpdate indicates that an update operation would not change any
// values.
var ErrEmptyUpdate = errors.New("Update operation does not change any values")

// ErrInvalidValue indicates that one or more parameters passed to a method
// had values that are invalid for that operation.
var ErrInvalidValue = errors.New("Invalid value for parameter")

// ErrObjectNotFound indicates that an Object was not found in the database.
var ErrObjectNotFound = errors.New("object was not found in database")

// ErrInvalidSavepoint is returned when a user of the Database uses an unkown
// (or expired) savepoint name.
var ErrInvalidSavepoint = errors.New("that save point does not exist")

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)database is (?:locked|busy)")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const retryDelay = 25 * time.Millisecond

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// Database wraps the connection to the underlying data store and
// associated state.
type Database struct {
	id      int64
	db      *sql.DB
	tx      *sql.Tx
	log     *log.Logger
	path    string
	queries map[query.ID]*sql.Stmt
}

// Open opens a Database. If the database specified by the path does not exist,
// yet, it is created and initialized.
func Open(path string) (*Database, error) {
	var (
		err      error
		dbExists bool
		db       = &Database{
			path:    path,
			queries: make(map[query.ID]*sql.Stmt, len(qDB)),
		}
	)

	openLock.Lock()
	defer openLock.Unlock()
	idCnt++
	db.id = idCnt

	if db.log, err = common.GetLogger(logdomain.Database); err != nil {
		return nil, err
	} else if common.Debug {
		db.log.Printf("[DEBUG] Open database %s\n", path)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	if dbExists, err = krylib.Fexists(path); err != nil {
		db.log.Printf("[ERROR] Failed to check if %s already exists: %s\n",
			path,
			err.Error())
		return nil, err
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		db.log.Printf("[ERROR] Failed to open %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !dbExists {
		if err = db.initialize(); err != nil {
			var e2 error
			if e2 = db.db.Close(); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to close database: %s\n",
					e2.Error())
				return nil, e2
			} else if e2 = os.Remove(path); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to remove database file %s: %s\n",
					db.path,
					e2.Error())
			}
			return nil, err
		}
		db.log.Printf("[INFO] Database at %s has been initialized\n",
			path)
	}

	return db, nil
} // func Open(path string) (*Database, error)

func (db *Database) initialize() error {
	var err error
	var tx *sql.Tx

	if common.Debug {
		db.log.Printf("[DEBUG] Initialize fresh database at %s\n",
			db.path)
	}

	if tx, err = db.db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	for _, q := range qInit {
		db.log.Printf("[TRACE] Execute init query:\n%s\n",
			q)
		if _, err = tx.Exec(q); err != nil {
			db.log.Printf("[ERROR] Cannot execute init query: %s\n%s\n",
				err.Error(),
				q)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Printf("[CANTHAPPEN] Cannot rollback transaction: %s\n",
					rbErr.Error())
				return rbErr
			}
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		db.log.Printf("[CANTHAPPEN] Failed to commit init transaction: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) initialize() error

// Close closes the database.
// If there is a pending transaction, it is rolled back.
func (db *Database) Close() error {
	// I wonder if would make more snese to panic() if something goes wrong

	var err error

	for key, stmt := range db.queries {
		if err = stmt.Close(); err != nil {
			db.log.Printf("[CRITICAL] Cannot close statement handle %s: %s\n",
				key,
				err.Error())
			return err
		}
		delete(db.queries, key)
	}

	if err = db.db.Close(); err != nil {
		db.log.Printf("[CRITICAL] Cannot close database: %s\n",
			err.Error())
	}

	db.db = nil
	return nil
} // func (db *Database) Close() error

func (db *Database) getQuery(id query.ID) (*sql.Stmt, error) {
	var (
		stmt  *sql.Stmt
		found bool
		err   error
	)

	if stmt, found = db.queries[id]; found {
		return stmt, nil
	} else if _, found = qDB[id]; !found {
		return nil, fmt.Errorf("Unknown Query %d",
			id)
	}

	db.log.Printf("[TRACE] Prepare query %s\n", id)

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(qDB[id]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto PREPARE_QUERY
		}

		db.log.Printf("[ERROR] Cannor parse query %s: %s\n%s\n",
			id,
			err.Error(),
			qDB[id])
		return nil, err
	}

	db.queries[id] = stmt
	return stmt, nil
} // func (db *Database) getQuery(query.ID) (*sql.Stmt, error)

// Begin begins an explicit database transaction.
// Only one transaction can be in progress at once, attempting to start one,
// while another transaction is already in progress will yield ErrTxInProgress.
func (db *Database) Begin() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Begin Transaction\n",
		db.id)

	if db.tx != nil {
		return ErrTxInProgress
	}

BEGIN_TX:
	for db.tx == nil {
		if db.tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				continue BEGIN_TX
			} else {
				db.log.Printf("[ERROR] Failed to start transaction: %s\n",
					err.Error())
				return err
			}
		}
	}

	return nil
} // func (db *Database) Begin() error

// Rollback terminates a pending transaction, undoing any changes to the
// database made during that transaction.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Rollback() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Roll back Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Rollback(); err != nil {
		return fmt.Errorf("Cannot roll back database transaction: %s",
			err.Error())
	}

	db.tx = nil

	return nil
} // func (db *Database) Rollback() error

// Commit ends the active transaction, making any changes made during that
// transaction permanent and visible to other connections.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Commit() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Commit Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Commit(); err != nil {
		return fmt.Errorf("Cannot commit transaction: %s",
			err.Error())
	}

	db.tx = nil
	return nil
} // func (db *Database) Commit() error

// JobSubmit adds a new Job to the database.
func (db *Database) JobSubmit(j *job.Job) error {
	const qid query.ID = query.JobSubmit
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(j.TimeSubmitted.Unix(), j.CmdString()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to persist new Job to database: %s\n",
			err.Error())
		return err
	}

	defer rows.Close() // nolint: errcheck,gosec

	rows.Next()

	if err = rows.Scan(&j.ID); err != nil {
		db.log.Printf("[ERROR] Cannot extract Job ID from row: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) JobSubmit(j *job.Job) error

// JobStart marks a Job as having started.
func (db *Database) JobStart(j *job.Job) error {
	const qid query.ID = query.JobStart
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var stamp = time.Now()

EXEC_QUERY:
	if _, err = stmt.Exec(stamp.Unix(), j.PID, j.SpoolOut, j.SpoolErr, j.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to mark Job as started: %s\n",
			err.Error())
		return err
	}

	j.TimeStarted = stamp
	return nil
} // func (db *Database) JobStart(j *job.Job) error

// JobFinish marks a Job as finished.
func (db *Database) JobFinish(j *job.Job) error {
	const qid query.ID = query.JobFinish
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var stamp = time.Now()

EXEC_QUERY:
	if _, err = stmt.Exec(stamp.Unix(), j.ExitCode, j.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to mark Job as finished: %s\n",
			err.Error())
		return err
	}

	j.TimeEnded = stamp
	return nil
} // func (db *Database) JobFinish(j *job.Job) error

// JobGetByID looks up a Job by its ID. If no Job with the given ID exists, it
// is not considered an error, in that case (nil, nil) is returned.
func (db *Database) JobGetByID(id int64) (*job.Job, error) {
	const qid query.ID = query.JobGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for Job %d: %s\n",
			id,
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck

	if rows.Next() {
		var (
			submit             int64
			start, end, exit   *int64
			cmd                string
			spoolout, spoolerr *string
			j                  = &job.Job{ID: id}
		)

		if err = rows.Scan(&submit, &start, &end, &exit, &cmd, &spoolout, &spoolerr); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		}

		j.TimeSubmitted = time.Unix(submit, 0)
		if start != nil {
			j.TimeStarted = time.Unix(*start, 0)
		}
		if end != nil {
			j.TimeEnded = time.Unix(*end, 0)
		}
		if exit != nil {
			j.ExitCode = int(*exit)
		}

		if spoolout != nil {
			j.SpoolOut = *spoolout
		}

		if spoolerr != nil {
			j.SpoolErr = *spoolerr
		}

		if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot parse JSON into Cmd: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		return j, nil
	}

	return nil, nil
} // func (db *Database) JobGetByID(id int64) (*job.Job, error)

// JobGetPending returns up to <max> Jobs that have been submitted but not yet started.
func (db *Database) JobGetPending(max int64) ([]job.Job, error) {
	const qid query.ID = query.JobGetPending
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(max); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for pending Jobs: %s\n",
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck
	var jobs = make([]job.Job, 0)

	for rows.Next() {
		var (
			submit     int64
			cmd        string
			jout, jerr *string
			j          job.Job
		)

		if err = rows.Scan(&j.ID, &submit, &cmd, &jout, &jerr); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		} else if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot restore Job command arguments from JSON: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		if jout != nil {
			j.SpoolOut = *jout
		}
		if jerr != nil {
			j.SpoolErr = *jerr
		}

		j.TimeSubmitted = time.Unix(submit, 0)
		jobs = append(jobs, j)
	}

	return jobs, nil
} // func (db *Database) JobGetPending(max int64) ([]job.Job, error)

// JobGetRunning returns the list of Jobs (possibly empty) that are currently being executed.
func (db *Database) JobGetRunning() ([]job.Job, error) {
	const qid query.ID = query.JobGetRunning
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for running Jobs: %s\n",
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck
	var jobs = make([]job.Job, 0)

	for rows.Next() {
		var (
			submit, start int64
			pid           *int64
			cmd           string
			jerr, jout    *string
			j             job.Job
		)

		if err = rows.Scan(&submit, &start, &cmd, &pid, &jout, &jerr); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		}

		if pid != nil {
			j.PID = *pid
		}

		if jout != nil {
			j.SpoolOut = *jout
		}
		if jerr != nil {
			j.SpoolErr = *jerr
		}

		j.TimeSubmitted = time.Unix(submit, 0)
		j.TimeStarted = time.Unix(start, 0)

		if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot parse JSON into Cmd: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
} // func (db *Database) JobGetRunning() ([]job.Job, error)

// JobGetUnfinished returns a slice of jobs that are currently running or
// enqueued to be run.
func (db *Database) JobGetUnfinished() ([]job.Job, error) {
	const qid query.ID = query.JobGetRunning
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for running Jobs: %s\n",
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck
	var jobs = make([]job.Job, 0)

	for rows.Next() {
		var (
			submit, start int64
			pid           *int64
			cmd           string
			jerr, jout    *string
			j             job.Job
		)

		if err = rows.Scan(&submit, &start, &cmd, &jout, &jerr, &pid); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		}

		if pid != nil {
			j.PID = *pid
		}

		if jout != nil {
			j.SpoolOut = *jout
		}
		if jerr != nil {
			j.SpoolErr = *jerr
		}

		j.TimeSubmitted = time.Unix(submit, 0)
		j.TimeStarted = time.Unix(start, 0)

		if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot parse JSON into Cmd: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
} // func (db *Database) JobGetUnfinished() ([]*job.Job, error)

// JobGetFinished returns the <max> most recently finished Jobs.
// Passing -1 for max means all of them.
func (db *Database) JobGetFinished(max int64) ([]job.Job, error) {
	const qid query.ID = query.JobGetFinished
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(max); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for running Jobs: %s\n",
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck
	var jobs = make([]job.Job, 0)

	for rows.Next() {
		var (
			submit, start, end, exitcode int64
			cmd                          string
			jout, jerr                   *string
			j                            job.Job
		)

		if err = rows.Scan(&j.ID, &submit, &start, &end, &exitcode, &cmd, &jout, &jerr); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		}

		j.ExitCode = int(exitcode)
		j.TimeSubmitted = time.Unix(submit, 0)
		j.TimeStarted = time.Unix(start, 0)
		if jout != nil {
			j.SpoolOut = *jout
		}
		if jerr != nil {
			j.SpoolErr = *jerr
		}

		if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot parse JSON into Cmd: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
} // func (db *Database) JobGetFinished(max int64) ([]*job.Job, error)

// JobGetAll loads *all* Jobs from the database, regardless of age or status.
// Beware that this might be a lot.
func (db *Database) JobGetAll() ([]job.Job, error) {
	const qid query.ID = query.JobGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to query database for running Jobs: %s\n",
			err.Error())
		return nil, err
	}

	defer rows.Close() // nolint: errcheck
	var jobs = make([]job.Job, 0)

	for rows.Next() {
		var (
			submit, start, end, exitcode int64
			pid                          *int64
			cmd                          string
			jout, jerr                   *string
			j                            job.Job
		)

		if err = rows.Scan(
			&j.ID,
			&submit,
			&start,
			&end,
			&exitcode,
			&cmd,
			&jout,
			&jerr,
			&pid); err != nil {
			db.log.Printf("[ERROR] Cannot extract values from cursor: %s\n",
				err.Error())
			return nil, err
		}

		j.ExitCode = int(exitcode)
		j.TimeSubmitted = time.Unix(submit, 0)
		j.TimeStarted = time.Unix(start, 0)
		if jout != nil {
			j.SpoolOut = *jout
		}
		if jerr != nil {
			j.SpoolErr = *jerr
		}
		if pid != nil {
			j.PID = *pid
		}

		if err = json.Unmarshal([]byte(cmd), &j.Cmd); err != nil {
			db.log.Printf("[ERROR] Cannot parse JSON into Cmd: %s\nRaw: %s\n",
				err.Error(),
				cmd)
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
} // func (db *Database) JobGetAll() ([]job.Job, error)

// JobDelete removes a Job from the database.
func (db *Database) JobDelete(j *job.Job) error {
	const qid query.ID = query.JobDelete
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if _, err = stmt.Exec(j.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to delete Job %d from database: %s\n",
			j.ID,
			err.Error())
		return err
	}

	return nil
} // func (db *Database) JobDelete(j *job.Job) error

// JobCleanFinished removes all finished Jobs from the database.
func (db *Database) JobCleanFinished() (int64, error) {
	const qid query.ID = query.JobCleanFinished
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return 0, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		db.log.Printf("[ERROR] Failed to delete finished Jobs from database: %s\n",
			err.Error())
		return 0, err
	}

	var cnt int64

	if cnt, err = res.RowsAffected(); err != nil {
		db.log.Printf("[ERROR] Cannot query number of rows deleted: %s\n",
			err.Error())
		return 0, err
	}

	return cnt, nil
} // func (db *Database) JobCleanFinished() (int64, error)
