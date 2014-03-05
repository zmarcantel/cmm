package main

import (
    "os"
    "fmt"
    "time"
    "strings"

    "github.com/tux21b/gocql"
)

//-------------------------------------------------------
// Migration Type
//-------------------------------------------------------

type Migration struct {
    Name        string
    Path        string
    File        *os.FileInfo
    Query       string
}

//
//  Exec
//    Executes the query(ies) described in the migration
//    Upon completion, will mark itself as complete
//
func (self Migration) Exec() error {
    if (Verbosity != QUIET) {
        fmt.Printf("\n\nMigration: %s\n", self.Name)
    }

    // if the migration has already been issued, notify of skip
    // catch error, and completed == true
    if complete, err := self.IsComplete() ; err != nil {
        fmt.Printf("ERROR: could not fetch status of migration\n%s\n\n", err)
        return err
    } else if (complete == true) {
        if (Verbosity != QUIET) {
            fmt.Printf("\tSkipping [%s]\n", self.Name)
        }
        return nil
    }

    // first, split the query into CQL "lines"
    var queries = strings.Split(self.Query, ";")
    if (Verbosity >= LOUD) {
        fmt.Printf("\tSplit into %d queries\n", len(queries))
    }

    // allow for multiple queries to be in the same file
    // split them up and run sequentially
    for i, q := range queries {
        var query = strings.TrimSpace(q)
        if (len(query) == 0) { continue } // if empty line

        if (Verbosity >= MEDIUM) {
            fmt.Printf("\tPart: %d\n", i)
        }

        var err = Session.Query(query).Consistency(Consistency).Exec()
        if err != nil {
            fmt.Printf("Error applying [%s]:\n\tQuery: '%s'\n%s\n", self.Name, query, err)
            os.Exit(1)
        }
    }

    // mark the migration complete
    self.MarkComplete()

    return nil
}


//
//  IsComplete
//    Queries the migrations table to detect if a migration has been run or not
//    This is done by testing for existence only
//
func (self Migration) IsComplete() (bool, error) {
    var name string
    var date time.Time

    if (Verbosity >= LOUD) {
        fmt.Println("\tChecking if complete")
    }

    // try to select the migration from the completed table
    // existence indicates completion
    var err = Session.Query(
        `SELECT * FROM migrations.completed WHERE name = ?`,
        self.Name).Consistency(Consistency).Scan(&name, &date)

    // not found is a passable error -- the scan is a better indicator
    // handle errors here
    if err != nil && err.Error() != "not found" {
        fmt.Printf("Error checking status of migration [%s]:\n%s\n", self.Name, err)
        return false, err
    }

    // if the name is a non-null value (gocql coerces null->"")
    // return it is in fact done
    if len(name) > 0 {
        if (Verbosity >= LOUD) {
            fmt.Println("\tWas completed")
        }

        return true, nil
    }

    if (Verbosity >= LOUD) {
        fmt.Println("\tNeeds to be run")
    }

    // otherwise, it needs to be run
    return false, nil
}


//
//  MarkComplete
//    Mark the given migratiton as complete
//    This consists of inserting it into the migrations table (pure existence test)
//
func (self Migration) MarkComplete() error {
    if (Verbosity >= MEDIUM) {
        fmt.Println("\tMarking complete")
    }

    // insert the filename (migration name) and the date run into the completion table
    var err = Session.Query(
        `INSERT INTO migrations.completed (name, date) VALUES (?, ?)`,
        self.Name, time.Now()).Exec()

    if err != nil {
        fmt.Printf("Error marking migration [%s] complete:\n%s\n", self.Name, err)
        return err
    }

    if (Verbosity >= SOFT) {
        fmt.Printf("Completed: %s\n", self.Name)
    }

    return nil
}



//
// Sorter
//

type MigrationCollection []Migration

// Len is part of sort.Interface.
func (self MigrationCollection) Len() int {
    return len(self)
}

// Swap is part of sort.Interface.
func (self MigrationCollection) Swap(i, j int) {
    if (Verbosity > LOUD) {
        fmt.Printf("\tSwap:\n\t%s\n\t%s\n\n", self[i].Name, self[j].Name)
    }

    self[i], self[j] = self[j], self[i]
}

// Less is part of sort.Interface.
func (self MigrationCollection) Less(i, j int) bool {
    if (Verbosity > LOUD) {
        fmt.Printf("\tCompare:\n\t%s\n\t%s\n\n", self[i].Name, self[j].Name)
    }

    // ISO-8601 prefix allows simple alphabetic sort
    return self[i].Name < self[j].Name
}


//-------------------------------------------------------
// General Functions
//-------------------------------------------------------

//
//  CreateMigrationsTable
//    Creates the table we will use to monitor the status of migrations
//    This table contains a set of (name, date) tuples of the name and completion date
//
func CreateMigrationTable(session *gocql.Session) {
    if (Verbosity > SOFT) {
        fmt.Println("Creating migration keypace")
    }

    // errors are ignored here
    // gocql is expanding error codes
    // these are idempotent anyway

    var keyErr = session.Query(`
        CREATE KEYSPACE migrations
        WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 }
    `).Exec()
    if keyErr != nil && strings.Index(keyErr.Error(), "Cannot add existing") < 0 {
        fmt.Printf("Error placing migrations keyspace: %s\n", keyErr)
    }

    // wait for that to settle
    time.Sleep(1000 * time.Millisecond)

    if (Verbosity > SOFT) {
        fmt.Println("Creating migration table")
    }

    var tableErr = session.Query(`
    CREATE TABLE migrations.completed (
        name      TEXT PRIMARY KEY,
        date      TIMESTAMP
    )`).Exec()
    if tableErr != nil && strings.Index(tableErr.Error(), "Cannot add already existing") < 0 {
        fmt.Printf("Error placing migrations keyspace: %s\n", tableErr)
    }
}


//
//  DoMigrations
//    Once a set of migrations has been loaded/sorted, we need to run them
//    This function iterates over the sorted slice calling .Exec(session) on all migrations
//    Logic as far as completion and marking are done by the migration's .Exec(session)
//
func DoMigrations(migrations []Migration, delay time.Duration) {
    // iterate over the migrations we loaded
    for _, m := range migrations {
        m.Exec()
        time.Sleep(delay)
    }
}
