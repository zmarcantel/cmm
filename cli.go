package main

import (
    "os"
    "fmt"
    "time"
    "sort"
    "strings"
    "strconv"
    "io/ioutil"
    "path/filepath"
    "encoding/json"

    "github.com/jessevdk/go-flags"
    "github.com/tux21b/gocql"

    "./db"
)

type Options struct {
    Verbose       []bool `short:"v"   long:"verbose"        description:"Show verbose log information. Supports -v[vvv] syntax."`

    Protocol      int    `short:"P"   long:"protocol"       description:"Protocol version to use [1 or 2]"`
    Consistency   string `short:"c"   long:"consistency"    description:"Cassandra consistency to use: one, quorum, all"`

    Hosts         string `short:"p"   long:"peers"          description:"Comma-serparated list of Cassandra hosts (hostname:port)"`
    Migrations    string `short:"m"   long:"migrations"     description:"Directory containing timestamp-prefixed migration files"`

    Delay         int    `short:"d"   long:"delay"          description:"Wait n milliseconds between migrations"`

    // pseudo-commands

    // help
    Describe      string `short:"D"   long:"describe"       description:"Print out the current layout as reported by the DB. [optional: keyspace or keyspace.table]" default:"none" `
}

var Opts Options

//
// Parse the supplied cli arguments
//
func HandleArguments() {
    if _, help := flags.Parse(&Opts) ; help != nil {
        os.Exit(1)
    } else {
        // handle hosts list
        if (len(Opts.Hosts) == 0) {
            if (Verbosity >= SOFT) {
                fmt.Println("Defaulting to single host { localhost:9042 }")
            }
            Hosts = []string{"localhost:9042"}
        }

        // handle migration directory
        if (len(Opts.Migrations) == 0) {
            if (Verbosity >= SOFT) {
                fmt.Println("Defaulting to current directory for migrations")
            }
            Opts.Migrations = "./"
        }

        // handle delay timer
        var delayErr error
        if (Opts.Delay > 0) {
            SettleTime, delayErr = time.ParseDuration(strconv.Itoa(Opts.Delay) + "ms")
            if (Verbosity >= SOFT) {
                fmt.Printf("Adding %dms delay after all queries\n", Opts.Delay)
            }
        } else {
            SettleTime, delayErr = time.ParseDuration("0ms")
        }
        if (delayErr != nil) { panic(delayErr) }

        // handle verbosity
        Verbosity = len(Opts.Verbose)

        // handle consistency
        if (len(Opts.Consistency) == 0) {
            Consistency = gocql.Quorum
        } else {
            switch(strings.ToLower(Opts.Consistency)) {
                case "any":
                    Consistency = gocql.Any
                    break

                case "one":
                    Consistency = gocql.One
                    break

                case "two":
                    Consistency = gocql.Two
                    break

                case "three":
                    Consistency = gocql.Three
                    break

                case "quorum":
                    Consistency = gocql.Quorum
                    break

                case "all":
                    Consistency = gocql.All
                    break

                case "localquorum":
                    Consistency = gocql.LocalQuorum
                    break

                case "eachquorum":
                    Consistency = gocql.EachQuorum
                    break

                case "serial":
                    Consistency = gocql.Serial
                    break

                case "localserial":
                    Consistency = gocql.LocalSerial
                    break

                default:
                    Consistency = gocql.Quorum
                    break
            }

            if (Verbosity > QUIET) {
                fmt.Printf("Using consistency: %s\n", Consistency)
            }
        }
    }
}


//
// Generate hosts slice from comma separated list
//
func BuildHosts(peerList string) {
    // TODO: validate and imprint formatting
    Hosts = strings.Split(peerList, ",") // defined in main.go

    if (Verbosity >= SOFT) {
        fmt.Println("Gathered Cassandra hosts")
    }
}


//
// Recursively walk tree looking for .cql files
// Build migration object when one is found
//
func GetMigrationFiles(dir string) {
    if (Verbosity >= SOFT) {
        fmt.Printf("Loading migration files from: %s\n", dir)
    }

    var topErr = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        // if there was an error, bubble to top
        if (err != nil) { return err }

        // if the file does not have the extension .cql
        // or it is a hidden file (or swap file for many editors)
        // just skip this file
        if (filepath.Ext(path) != ".cql" || info.Name()[:1] == ".") { return nil }

        // attempt to read the contents of the migration file
        if contents, fileErr := ioutil.ReadFile(path) ; fileErr != nil {
            fmt.Printf("Error reading migration file: %s\n", path)
            return fileErr
        } else {
            if (Verbosity >= LOUD) {
                fmt.Printf("\tFile: %s\n", info.Name())
            }

            // we got everything we need, so let's create a migration
            // and append it the list initiated in main.go
            Migrations = append(Migrations, Migration{
                Name:           info.Name(),
                Path:           path,
                File:           &info,
                Query:          string(contents),
            })
        }

        return nil
    })

    // if there was an error walking the path print it and exit
    // a notice that an error occured is printed within the walk func
    if topErr != nil {
        fmt.Print(topErr)
        os.Exit(1)
    }

    if (Verbosity >= SOFT) {
        fmt.Println("Sorting migrations")
    }
    sort.Sort(Migrations)
}


//
//  handlePseudocommands
//      This will be the "catch point" for flags that do not run migrations
//
func handlePseudocommands() {
    if (Opts.Describe != "none") {
        handleDescribe()
        os.Exit(1)
    }
}


//
//  handleDescribe
//      Describe the given object
//      Can be "", "all", "none", "keyspace", "keyspace.table"
//
func handleDescribe() {
    var result []byte
    var err error

    if (Opts.Describe == "" || Opts.Describe == "all") {    // "all", or just --describe
        var keyspaces = db.AllKeyspaces()
        result, err = json.MarshalIndent(keyspaces, "", "    ")

    } else if (strings.Index(Opts.Describe, ".") > 0) {     // keyspace.table pair
        var parts = strings.Split(Opts.Describe, ".")
        var table = db.Table(parts[0], parts[1])
        result, err = json.MarshalIndent(table, "", "    ")

    } else {                                                // default to keyspace
        var keyspace = db.Keyspace(Opts.Describe)
        result, err = json.MarshalIndent(keyspace, "", "    ")
    }

    if err != nil {
        fmt.Printf("ERROR: invalid internal json representation\n%s\n", err)
        os.Exit(1)
    }

    fmt.Print(string(result))
}