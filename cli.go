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

    "github.com/jessevdk/go-flags"
)

type Options struct {
    Verbose     []bool `short:"v"   long:"verbose"     description:"Show verbose log information. Supports -v[vvv] syntax."`

    Protocol    int    `short:"P"   long:"protocol"    description:"Protocol version to use [1 or 2]"`

    Hosts       string `short:"p"   long:"peers"       description:"Comma-serparated list of Cassandra hosts (hostname:port)"`
    Migrations  string `short:"m"   long:"migrations"  description:"Directory containing timestamp-prefixed migration files"`

    Delay       int    `short:"d"   long:"delay"       description:"Wait n milliseconds between migrations"`
}

var Opts Options

func HandleArguments() {
    if _, help := flags.Parse(&Opts) ; help != nil {
        os.Exit(1)
    } else {
        // handle hosts list
        if (len(Opts.Hosts) == 0) {
            fmt.Println("Defaulting to single host { localhost:9042 }")
            Hosts = []string{"localhost:9042"}
        }

        // handle migration directory
        if (len(Opts.Migrations) == 0) {
            fmt.Println("Defaulting to current directory for migrations")
            Opts.Migrations = "./"
        }

        // handle delay timer
        var delayErr error
        if (Opts.Delay > 0) {
            SettleTime, delayErr = time.ParseDuration(strconv.Itoa(Opts.Delay) + "ms")
            fmt.Printf("Adding %dms delay after all queries\n", Opts.Delay)
        } else {
            SettleTime, delayErr = time.ParseDuration("0ms")
        }
        if (delayErr != nil) { panic(delayErr) }

        Verbosity = len(Opts.Verbose)
    }
}

func BuildHosts(peerList string) {
    // TODO: validate and imprint formatting
    Hosts = strings.Split(peerList, ",") // defined in main.go

    if (Verbosity >= SOFT) {
        fmt.Println("Gathered Cassandra hosts")
    }
}

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