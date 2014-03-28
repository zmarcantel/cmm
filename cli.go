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
    List          bool   `short:"l"   long:"list"           description:"Return a list of complete and remaining migrations."`
    JsonList      bool   `short:"j"   long:"list.json"      description:"Same as above, but returns the sets as distinct JSON arrays (complete, remaining) within a parent object"`
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
    }

    // handle verbosity
    Verbosity = len(Opts.Verbose)

    // handle config first so any other specified arguments overwrite it
    // if a config file is specified, use it
    // otherwise, check ~/.cmm/config.json and then /etc/cmm/config.json
    if (len(Opts.Config) > 0) {
        if (Verbosity >= SOFT) {
            fmt.Printf("Loading config from %s\n", Opts.Config)
        }
        handleConfig()
    } else {
        if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".cmm/config.json")); os.IsNotExist(err) {
            // the "default" config does not exist
            // atttempt to load a global one
            if _, etcErr := os.Stat("/etc/cmm/config.json"); os.IsNotExist(etcErr) {
                // global does not exist either. ignore
                if (Verbosity >= SOFT) { fmt.Println("No config files found.") }
            } else {
                Opts.Config = "/etc/cmm/config.json"
                handleConfig()
            }
        } else {
            Opts.Config = filepath.Join(os.Getenv("HOME"), ".cmm/config.json")
            handleConfig()
        }
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

    // handle consistency
    if (len(Opts.Consistency) == 0) {
        Consistency = gocql.Quorum
    } else {
        var consistencies = map[string]gocql.Consistency {
            "any":              gocql.Any,
            "one":              gocql.One,
            "two":              gocql.Two,
            "three":            gocql.Three,
            "quorum":           gocql.Quorum,
            "all":              gocql.All,
            "localquorum":      gocql.LocalQuorum,
            "eachquorum":       gocql.EachQuorum,
            "serial":           gocql.Serial,
            "localserial":      gocql.LocalSerial,
        }

        if val, exists := consistencies[strings.ToLower(Opts.Consistency)] ; exists {
            Consistency = val
        } else {
            Consistency = gocql.Quorum
        }

        if (Verbosity > QUIET) {
            fmt.Printf("Using consistency: %s\n", Consistency)
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


//
//  handlePseudocommands
//      This will be the "catch point" for flags that do not run migrations
//
func handlePseudocommands() {
    if (Opts.Describe != "none") {
        var jsonString = Describe(Opts.Describe)
        fmt.Println(jsonString)
        os.Exit(1)
    }

    if (Opts.Backfill != "none") {
        var migs = Backfill(Opts.Describe, Opts.File)
        // if no output path specified, just print
        if (len(Opts.Output) == 0) {
            migs.Print()
        } else {
            migs.Save(Opts.Output)
        }
        os.Exit(1)
    }

    if (Opts.List) {
        fmt.Println(List(Opts.JsonList))
        os.Exit(1)
    }

    if (Opts.JsonList) {
        fmt.Println(ListToJSON(List(Opts.JsonList)))
        os.Exit(1)
    }
}