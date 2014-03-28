package main

import (
    "os"
    "fmt"
    "strings"
    "io/ioutil"
    "encoding/json"

    "./db"

    "github.com/aybabtme/color/brush"
)

//
//  Describe
//      Describe the given keyspace(s) or tables
//      Returns a JSON string representation of the item
//      Can be "", "all", "none", "keyspace", "keyspace.table"
//
func Describe(target string) string {
    var result []byte
    var err error

    if (target == "" || target == "all") {    // "all", or just --describe
        var keyspaces, err = db.AllKeyspaces()
        if (err != nil) {
            if (err.Error() == "not found") {
                fmt.Println("That kayspace does not exist.")
            } else {
                fmt.Printf("ERROR: could not get keyspace:\n%s\n", err)
            }
            os.Exit(1)
        }
        result, err = json.MarshalIndent(keyspaces, "", "    ")

    } else if (strings.Index(target, ".") > 0) {     // keyspace.table pair
        var parts = strings.Split(target, ".")
        var table, err = db.Table(parts[0], parts[1])
        if (err != nil) {
            if (err.Error() == "not found") {
                fmt.Println("That columnfamily does not exist.")
            } else {
                fmt.Printf("ERROR: could not get columnfamily:\n%s\n", err)
            }
            os.Exit(1)
        }
        result, err = json.MarshalIndent(table, "", "    ")

    } else {                                                // default to keyspace
        var keyspace, err = db.Keyspace(target)
        if (err != nil) {
            if (err.Error() == "not found") {
                fmt.Println("That columnfamily does not exist.")
            } else {
                fmt.Printf("ERROR: could not get columnfamily:\n%s\n", err)
            }
            os.Exit(1)
        }
        result, err = json.MarshalIndent(keyspace, "", "    ")
    }

    if err != nil {
        fmt.Printf("ERROR: invalid internal json representation\n%s\n", err)
        os.Exit(1)
    }

    var formatted = strings.Replace(string(result), "\\u003c", "<", -1)
    formatted = strings.Replace(formatted, "\\u003e", ">", -1)

    return formatted
}


//
//  Backfill
//      Generation of migrations that bring the current table/keyspace format to equal a JSON descriptor
//
func Backfill(collection, target string) MigrationCollection {
    if (len(target) <= 0) {
        fmt.Println("ERROR: must supply (-f, --file) flag to backfill")
        os.Exit(1)
    }

    if (strings.Index(collection, ".") < 1) {
        fmt.Println("ERROR: backfill can only be used on {keyspace}.{table} items")
        os.Exit(1)
    }

    // read target JSON
    var contents, err = ioutil.ReadFile(target)
    if (err != nil) {
        fmt.Printf("ERROR: could not read descriptor JSON:\n%s\n\n", err)
        os.Exit(1)
    }

    // unmarshal target JSON
    var targetJSON map[string]interface{}
    var jsonErr = json.Unmarshal(contents, &targetJSON)
    if (jsonErr != nil) {
        fmt.Printf("ERROR: could not parse descriptor JSON:\n%s\n\n", jsonErr)
        os.Exit(1)
    }

    // remove any comments of the suggested form
    delete(targetJSON, "_")

    // create the placeholder for the result migrations
    var migrations MigrationCollection

    // get existing table
    var parts = strings.Split(collection, ".")
    var table, tblErr = db.Table(parts[0], parts[1])
    if (tblErr != nil) {
        if (tblErr.Error() == "not found") {
            migrations = CreateTableMigration(parts[0], parts[1], targetJSON)
        } else {
            os.Exit(1)
        }
    } else {
        migrations = BackfillTable(table, targetJSON)
    }

    return migrations
}


//
//  List
//      Return lists of completed and remaining migrations
//      JSON flag determines if output is JSON
//
func List(isJson bool) (complete MigrationCollection, remaining MigrationCollection) { // should explicitly be passed Opts.JsonList
    GetMigrationFiles(Opts.Migrations)

    for _, mig := range Migrations {
        var isComplete, err = mig.IsComplete()
        if (err != nil) {
            fmt.Printf("ERROR: error checking completion status of [%s]\n%s\n\n", mig.Name, err)
            return
        }

        if (isComplete == false) {
            remaining = append(remaining, mig)
        } else {
            complete = append(complete, mig)
        }
    }

    for _, mig := range complete {
        fmt.Printf("%5s  %s\n", brush.Green("+"), brush.Green(mig.Name))
    }
    for _, mig := range remaining {
        fmt.Printf("%5s  %2s\n", brush.Red("-"), brush.Red(mig.Name))
    }

    return complete, remaining
}


//  ListToJSON
//      Return the JSON string representation of the List functions
//      Simply marshals the structure into a JSON map of 'Complete' and 'Remaining' arrays
//
func ListToJSON(complete, remaining MigrationCollection) string {
    var formatted, err = json.MarshalIndent(map[string][]Migration{
        "Complete":       complete,
        "Remaining":      remaining,
    }, "", "    ")
    if (err != nil) {
        fmt.Printf("ERROR: could not marshal JSON of --list\n%s\n\n", err)
        os.Exit(1)
    }
    return string(formatted)
}


//
//  BackfillTable
//    Generates a series of queries that equate to the diff of the current table, and a given JSON
//
func BackfillTable(table db.TableDescriptor, target map[string]interface{}) []Migration {
    var result []Migration

    // check for additions
    for key, value := range target {
        var found = false
        var col db.ColumnDescriptor
        for _, col = range table.Columns {
            if (col.Name == key) {
                found = true

                // multitask and look for changed type
                // convert types to upper case for ease
                var upperKnown = strings.ToUpper(col.Type)
                var upperTesting = strings.ToUpper(value.(string))
                if (upperKnown != upperTesting) {
                    // handle the PRIMARY KEY part a little differently
                    if (col.Primary) {
                        var trimmedTesting = upperTesting[:len(upperTesting) - len(" PRIMARY KEY")]
                        if (upperKnown == trimmedTesting) { continue }
                    }
                    result = append(result, ChangeTypeMigration(table, col.Name, value.(string)))
                }

                // escape for loop
                break
            }
        }

        if (!found) {
            result = append(result, CreationMigration(table, key, value.(string)))
        }
    }


    // check for removals
    for _, col := range table.Columns {
        var found = false
        for key, _ := range target {
            if (col.Name == key) {
                found = true
                break
            }
        }

        if (!found) {
            result = append(result, RemovalMigration(table, col.Name))
        }
    }


    return result
}