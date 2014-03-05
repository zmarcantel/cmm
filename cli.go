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

var Opts Options
type Options struct {
    Verbose       []bool `short:"v"   long:"verbose"        description:"Show verbose log information. Supports -v[vvv] syntax."`
    Config        string `short:"C"   long:"config"         description:"Provide a path to a JSON file containing hosts,migrations,version,etc" value-name:"FILE"`

    Protocol      int    `short:"P"   long:"protocol"       description:"Protocol version to use [1 or 2]" value-name:"VERSION"`
    Consistency   string `short:"c"   long:"consistency"    description:"Cassandra consistency to use: one, quorum, all" value-name:"LEVEL"`

    Hosts         string `short:"p"   long:"peers"          description:"Comma-serparated list of Cassandra hosts (hostname:port)" value-name:"HOSTS"`
    Migrations    string `short:"m"   long:"migrations"     description:"Directory containing timestamp-prefixed migration files" value-name:"DIRECTORY"`

    Delay         int64  `short:"d"   long:"delay"          description:"Wait n milliseconds between migrations" value-name:"MS"`

    File          string `short:"f"   long:"file"           description:"File to do operations with [used in config, backfill]" value-name:"FILE"`
    Output        string `short:"o"   long:"output"         description:"File or path to output operation to"`

    // start pseudo-commands

    // help
    Describe      string `short:"D"   long:"describe"       description:"Print out the current layout as reported by the DB. ['all', keyspace, or keyspace.table]" default:"none" value-name:"ITEM"`
    Backfill      string `short:"b"   long:"backfill"       description:"Generate migrations equating the the diff of the existing table and the JSON descriptor given by --file" default:"none" value-name:"ITEM"`
}


type Config struct {
    Protocol       int
    Consistency    string

    Peers          []string
    Migrations     string

    Delay          int64
    File           string
    Output         string
}


//
// Parse the supplied cli arguments
//
func HandleArguments() {
    if _, help := flags.Parse(&Opts) ; help != nil {
        os.Exit(1)
    } else {
        // handle config first so any other specified arguments overwrite it
        if (len(Opts.Config) > 0) {
            if (Verbosity >= SOFT) {
                fmt.Printf("Loading config from %s\n", Opts.Config)
            }
            handleConfig()
        }


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
            SettleTime, delayErr = time.ParseDuration(strconv.FormatInt(Opts.Delay, 10) + "ms")
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

    if (Opts.Backfill != "none") {
        handleBackfill()
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


//
//  handleBackfill
//      Initiates the generation of migrations that bring the current table to equal a JSON descriptor
//
func handleBackfill() {
    if (len(Opts.File) <= 0) {
        fmt.Println("ERROR: must supply (-f, --file) flag to backfill")
        os.Exit(1)
    }

    if (strings.Index(Opts.Backfill, ".") < 1) {
        fmt.Println("ERROR: backfill can only be used on {keyspace}.{table} items")
        os.Exit(1)
    }

    // read target JSON
    var contents, err = ioutil.ReadFile(Opts.File)
    if (err != nil) {
        fmt.Printf("ERROR: could not read descriptor JSON:\n%s\n\n", err)
        os.Exit(1)
    }

    // unmarshal target JSON
    var target map[string]interface{}
    var jsonErr = json.Unmarshal(contents, &target)
    if (jsonErr != nil) {
        fmt.Printf("ERROR: could not parse descriptor JSON:\n%s\n\n", jsonErr)
        os.Exit(1)
    }

    delete(target, "_")

    // get existing table
    var parts = strings.Split(Opts.Backfill, ".")
    var table = db.Table(parts[0], parts[1])

    var migrations = BackfillTable(table, target)

    for _, mig := range migrations {
        // if no output path specified, just print
        if (len(Opts.Output) == 0) {
            fmt.Printf("%s\n", mig.Query)
        } else {
            // gave us a string, hopefully path
            var err = ioutil.WriteFile(filepath.Join(Opts.Output, mig.Name), []byte(mig.Query), 0777)
            if (err != nil) {
                fmt.Printf("ERROR: could not save migration to directory [%s]\n%s\n\n", Opts.Output, err)
            }
        }
    }
}


//
//  handleConfig
//      Load a configuration map from a given JSON file
//
func handleConfig() {
    var contents, err = ioutil.ReadFile(Opts.Config)
    if (err != nil) {
        fmt.Printf("ERROR: cannot read config file [%s]\n%s\n\n", Opts.Config, err)
        os.Exit(1)
    }

    var formatted map[string]interface{}
    var jsonErr = json.Unmarshal(contents, &formatted)
    if (jsonErr != nil) {
        fmt.Printf("ERROR: cannot read config file json:\n%s\n\n", jsonErr)
        os.Exit(1)
    }

    for key, val := range formatted {
        switch(key) {
            case "Protocol":
                Opts.Protocol = int(val.(float64))
                break

            case "Consistency":
                Opts.Consistency = val.(string)
                break


            case "Peers":
                for _, p := range val.([]interface{}) {
                    if ( len(Opts.Hosts) > 0 ) { Opts.Hosts += "," }
                    Opts.Hosts += p.(string)
                }
                // Opts.Hosts = strings.Join(val, ",")
                break

            case "Migrations":
                Opts.Migrations = val.(string)
                break

            case "Delay":
                Opts.Delay = int64(val.(float64))
                break

            case "File":
                Opts.File = val.(string)
                break

            case "Output":
                Opts.Output = val.(string)
                break
        }
    }
}