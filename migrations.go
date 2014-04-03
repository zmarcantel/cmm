package main

import (
    "os"
    "fmt"
    "time"
    "regexp"
    "strings"
    "io/ioutil"
    "path/filepath"

    "github.com/tux21b/gocql"

    "github.com/zmarcantel/cmm/db"
)

//-------------------------------------------------------
// Migration Type
//-------------------------------------------------------

type Migration struct {
    Name        string
    Path        string
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
//  GetDelay
//      Parse the comments to see if a delay has been set
//      If not, the SettleTime resolved from the cli is used
//
//      comment form: '-- delay: 500' with or without spaces
//
func (self Migration) GetDelay() time.Duration {
    var result = SettleTime

    // matches format `-- delay: ###` with spaces irrelevant and # being a count in ms
    var delayRegex = regexp.MustCompile(`--\s?delay:\s?[0-9]+`)
    var migrationDelay = delayRegex.Find([]byte(self.Query))

    if (len(migrationDelay) > 0) {
        var delayString = string(migrationDelay)
        var msString = delayString[ (strings.LastIndex(delayString, ":") + 1) : ] + "ms"

        var err error
        result, err = time.ParseDuration(strings.TrimSpace(msString))
        if (err != nil) {
            fmt.Printf("ERROR: could not parse migration-level delay\n%s\n\n", err)
            os.Exit(1)
        }
    }

    return result
}


//
//  String -- returns query as string representation
//
func (self Migration) String() string {
    return self.Query
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

func (self MigrationCollection) Print() {
    //  format the migrations slice
    for _, mig := range self {
        fmt.Printf("Name: %s\nQuery:\n%s\n\n", mig.Name, mig.Query)
    }
}

func (self MigrationCollection) Save(path string) {
    for _, mig := range self {
        var err = ioutil.WriteFile(filepath.Join(path, mig.Name), []byte(mig.Query), 0777)
        if (err != nil) {
            fmt.Printf("ERROR: could not save migration to directory [%s]\n%s\n\n", Opts.Output, err)
        }
    }
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
    time.Sleep(2000 * time.Millisecond)

    if (Verbosity > SOFT) {
        fmt.Println("Creating migration table")
    }

    var tableErr = session.Query(`
    CREATE TABLE migrations.completed (
        name      TEXT PRIMARY KEY,
        date      TIMESTAMP
    )`).Exec()
    if tableErr != nil && strings.Index(tableErr.Error(), "Cannot add already existing") < 0 {
        fmt.Printf("Error placing migrations.completed table: %s\n", tableErr)
    }

    // wait for that to settle
    time.Sleep(2000 * time.Millisecond)
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

        // delay
        var delay = m.GetDelay()
        if (delay > 0 && Verbosity >= SOFT) {
            fmt.Printf("\tWaiting %s\n", delay)
        }
        time.Sleep(delay)
    }
}


//
//  CreationMigration
//    Create a migration for adding a field
//
func CreationMigration(table db.TableDescriptor, colName string, colType string) Migration {
    var result = "ALTER TABLE " + table.Keyspace + "." + table.Name + " ADD " + colName + " " + colType + ";"

    var currDate = time.Now().UTC()
    return Migration{
        Name:     currDate.Format(time.RFC3339Nano) + "_add_" + colName + "_to_" + table.Name + ".cql",
        Query:    result,
    }
}


//
//  RemovalMigration
//    Cretes a migration for removing a field
//
func RemovalMigration(table db.TableDescriptor, colName string) Migration {
    var result = "ALTER TABLE " + table.Keyspace + "." + table.Name + " DROP " + colName + ";"

    var currDate = time.Now().UTC()
    return Migration{
        Name:     currDate.Format(time.RFC3339Nano) + "_remove_" + colName + "_from_" + table.Name + ".cql",
        Query:    result,
    }
}


//
//  ChangeTypeMigration
//    Creates a migration for changing a column's type
//
func ChangeTypeMigration(table db.TableDescriptor, colName, newType string) Migration {
    var result = "ALTER TABLE " + table.Keyspace + "." + table.Name + " ALTER " + colName + " TYPE " + newType + ";"

    var currDate = time.Now().UTC()
    var typeString = strings.Replace(newType, "<", "_of_", -1)
    typeString = strings.Replace(typeString, ">", "_", -1)
    typeString = strings.Replace(typeString, " ", "_", -1)
    typeString = strings.Replace(typeString, ",", "_to_", -1)


    typeString = strings.TrimSuffix(typeString, "_")
    typeString = strings.ToLower(typeString)

    return Migration{
        Name:     currDate.Format(time.RFC3339Nano) + "_change_" + table.Keyspace + "_" + table.Name + "_" + colName + "_to_" + typeString + ".cql",
        Query:    result,
    }
}


//
//  CreateTableMigration
//      Creates a migrations that will create a table from a target schema
//
func CreateTableMigration(keyspace, table string, target map[string]interface{}) []Migration {
    var result = "CREATE TABLE " + keyspace + "." + table + " (\n"
    for key, value := range target {
        result += fmt.Sprintf("\t%10s\t%-20s\n", key, value.(string))
    }
    result += "\n);"

    var currDate = time.Now()
    return []Migration{
        Migration{
            Name:       currDate.Format(time.RFC3339Nano) + "_create_table_" + keyspace + "_" + table + ".cql",
            Query:      result,
        },
    }
}