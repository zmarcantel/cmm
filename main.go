package main

import (
    "os"
    "fmt"
    "time"

    "github.com/tux21b/gocql"

    "./db"
)

var Hosts           []string
var Migrations      MigrationCollection
var SettleTime      time.Duration
var Session         *gocql.Session
var Verbosity       int
var Consistency     gocql.Consistency

const (
    QUIET   = 0;
    SOFT    = 1;
    MEDIUM  = 2;
    LOUD    = 3;
    ALL     = 4;
)

func main() {
    // handle all cli arguments
    HandleArguments()

    // build cassandra hosts from the cli/default
    BuildHosts(Opts.Hosts)

    // create a cluster of Cassandra connections
    // var cluster *gocql.ClusterConfig
    _, Session = connectCluster()
    defer Session.Close()
    db.Init(Session)

    // handle arguments that do not result in running migrations
    handlePseudocommands()

    // load migration files and sort them
    GetMigrationFiles(Opts.Migrations)
    fmt.Printf("Loaded %d migrations\n", len(Migrations))

    // idempotently create migrations keyspace/table
    CreateMigrationTable(Session)

    // run the migrations
    DoMigrations(Migrations, SettleTime)
}

func connectCluster() (*gocql.ClusterConfig, *gocql.Session) {
    var protoVersion = 2
    if (Opts.Protocol > 0) { protoVersion = Opts.Protocol }

    var cluster = gocql.NewCluster(Hosts...)
    cluster.Consistency = gocql.Quorum
    cluster.ProtoVersion = protoVersion

    var session, err = cluster.CreateSession()

    if (err != nil) {
        fmt.Printf("ERROR: could not create session for cluster\n%s\n\n", err)
        os.Exit(1)
    }

    return cluster, session
}