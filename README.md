cmm
===

Cassandra Migrations Manager that makes managing schemas a breeze.

1. Loads `.cql` files from a directory (or many nested directories)
2. Sorts them to ensure they are run in order
3. Executes any new migrations
4. Works backward to _create_ migrations
  * Get from where you are now to where you need to be

Production Focused Features

1. [Connection pooling](#hosts)
2. [Schema to JSON](#describe)
3. [JSON to Schema](#backfill)
4. [Set per-migration delay times](#migration-file)
5. [Setting protocol version](#protocol)


Table of Contents:
* [Examples](#examples) -- see how easy it can be
* [Options](#command-flags) -- all the available settings
* [Migration File](#migration-file) -- how to create migrations
* [Config File](#config-file) -- load any/all options from a json file
* [Query Commands](#informational-commands) -- easily query metadata about your db, keyspaces, or columnfamiles
  * [describe](#describe) -- schema to json
  * [backfill](#backfill) -- json to schema
  * [list](#list) -- print report of completed/remaining migrations
* Testing
  * [Automated Testing in VM](#testing-in-a-vm)
  * [Against Your Cluster](#testing-with-your-cluster)


Planned Features
================

* Support Solr
  * For the [DataStax](http://www.datastax.com/what-we-offer/products-services/datastax-enterprise) users
  * Upload `schema.xml`
  * Core exist: `reload` Else: `create`
* Expand Backfilling
  * Allow changing of `PRIMARY KEY`
  * Better rename (currently add_new + remove_old)


How To Install
==============

Binaries coming soon!

#### Pre-requisite

* [go](http://golang.org/)

#### Steps

The below steps are usable on all machines. (Install is different on windows)


1. Checkout code

    `git clone https://github.com/zmarcantel/cmm`

2. Build

    `cd cmm && make`

3. Install

    `sudo make install`

___Note:___ install may not require `sudo`

On ___Windows___, `make install` will fail as it tries to copy into `/usr/local/bin`. Windows does not have this directory, but then again, Cassandra is typically not run on Windows. Just move `bin/cmm` to somewhere exectuable by `cmd.exe`


Migration File
==============

All migration files are loaded, split into individual queries (split by ';'), and run sequentially.

Optionally, a comment sepcifying a per-migration delay can be used.

* `-- delay: ###`
  * spaces do not matter
  * ### is the amount of milliseconds

#### Example

    -- delay: 1500
    -- the above comment is parsed, adding a 1500ms delay to end of this query

    ALTER TABLE main.users ADD friends SET<UUID>;
    ALTER TABLE main.users ADD recommended_friends SET<UUID>;

    CREATE TABLE main.friend_recommendations (
        id              UUID PRIMARY KEY,
        user            UUID,
        recommending    UUID,
        date            TIMESTAMP,
        rating          FLOAT
    );




Config File
===========

`cmm` supports loading all command flags (except for pseudo-commands) from a JSON file.

This is helpful when scripting certain actions or dealing with frequently-appearing yet fairly static options like `peers` or `migrations`

__NOTE:__ any options provided on the command line overrule any given in the config file

### How to load config

Simply supply the `-C` or `--config` flag followed by a path to the file.

TODO: automatically load `cmm.json` in current directory

### Example Config

There is an example in `test/config.json` as well.

````json
{
    "Protocol":       2,                        # cassandra protocol to use
    "Consistency":    "quorum",                 # consistency to require of queries

    "Peers":          [                         # array of seed peers
        "192.168.33.100",
        "192.168.33.101",
        "192.168.33.150"
    ],

    "Migrations":     "./migrations",           # root directory of migrations

    "Delay":          250,                      # Delay between migrations (highly optional)
    "File":           "./schemas/user.json",    # Default file to use for pseudo-commands
    "Output":         "./migrations"            # Default folder to save output migrations to
}
````



Command Flags
=============

    Verbose       short: "v"   long: "verbose"        description: "Show verbose log information. Supports -v[vvv] syntax."
    Config        short: "C"   long: "config"         description: "Provide a path to a JSON file containing hosts,migrations,version,etc"`

    Protocol      short: "P"   long: "protocol"       description: "Protocol version to use [1 or 2]"
    Consistency   short: "c"   long: "consistency"    description: "Cassandra consistency to use: one, quorum, all, etc"`

    Hosts         short: "p"   long: "peers"          description: "Comma-serparated list of Cassandra hosts (hostname:port)"
    Migrations    short: "m"   long: "migrations"     description: "Directory containing timestamp-prefixed migration files"

    Delay         short: "d"   long: "delay"          description: "Wait n milliseconds between migrations"

    File          short: "f"   long: "file"           description: "Generic file input -- used in giving backfill a JSON file"
    Output        short: "o"   long: "output"         description: "File or path to output operation to"


    // Pseudo-Commands -- results in information, not commands being run

    Help          short: "h"   long: "help"           description: "Show the help menu"
    Describe      short: "D"   long: "describe"       description: "Prints a JSON represntation of ['all', 'none','{keyspace}', '{keyspace}.{table}']"
    Backfill      short: "b"   long: "backfill"       description: "Backfill migrations based on an existing table and a JSON descriptor provided by --file"
    List          short: "l"   long: "list"           description: "Prints a list of migrations that have been completed and those that need to be run"
    List          short: "j"   long: "list.json"      description: "Same as the list function above, but prints out JSON"


#### Examples:

* Run all the migrations in this directory, pooling a single peer at localhost
    * `cmm`
* Run all the migrations in this directory, with two hosts
    * `cmm -p foo,bar`
* Run all the migrations in this directory, with two hosts on custom ports
    * `cmm -p foo:10000,bar:10000`
* Run all the migrations in a different directory, pooling a single peer at localhost
    * `cmm -m ~/project/migrations`
* Run all the migrations in a different directory, with two hosts
    * `cmm -m ~/project/migrations -p foo,bar`
* Run all the migrations in a different directory, with two hosts and a 750ms delay
    * `cmm -m ~/project/migrations -p foo,bar -d 750`


Hosts
-----

Supply a comma-delimited list of Cassandra peers.

Peers can include a port or default to `9042`.

___Example:___ `cmm --peers 127.0.0.1,dbone:9999,dbtwo`

You can see in the above example that an IP or hostname is valid.

The example implies:

* The `localhost` is running Cassandra on port `9042`
* There is a machine with resolveable hostname `dbone` running on `9999`
* There is a machine with resolveable hostname `dbtwo` on standard `9042`

#### Argument

    Short:  `-p`
    Long:   `--peers`

#### Default: `{ localhost:9042 }`




Migrations
----------

Directory containing the migrations to be run.

Individual migration files should have extension `.cql`.

#### Nesting

This directory __can__ be nested! Feel free to use an organizing structure:

* keyspaces
* users
  * messaging
  * profiles
  * analytics
* items
  * for sale
    * auction
    * upfront
  * not for sale


#### Sorting

`cmm` sorts migrations alphabetically before running them. This provides a few naming strategies:

* timestamp
    * prefix title with RFC3339 timestamp
    * __preferred__ method
    * `2014-03-02T06-13-03.495Z_create_item_table.cql`
    * Easy to create
      * unix/mac: `date -u +"%Y-%m-%dT%H:%M:%SZ"`
      * js/browser: `(new Date()).toISOString()`

* blocked
    * use range prefix
    * `0-99 = users`, `100-199 = items`
    * `01-add-user-keyspace.cql`
    * use large sized blocks
    * good only in shallow tree structures

* numerical
    * `1-foo.cql`, `2-bar.cql`
    * good if using only one directory

#### Argument

    Short:  `-m`
    Long:   `--migrations`

#### Default: `./`



Delay
-----

Sometimes creation queries need to setlle (typically keyspace/table creation or BATCHes > 10).

Add a `delay` to the end off all queries using this flag.

#### Argument

    Short:  `-d`
    Long:   `--delay`

#### Default: `0ms`




Verbosity
---------

Change the level of verbosity in logging output.

If you want more verbose logs, add more v's! `-vvvv` for most verbose.

#### Argument

    Short: `-v[vvv]`

#### Default: `silent`



Protocol
--------

Change the Cassnadra protocol version number.

Older (<=1.2.1) versions of Cassandra require version `1` of the protocol.

New versions (>=2.0.0) utilize version `2`. (default)

#### Argument

    Short:  `-P`
    Long:   `--protocol`

#### Default: `2`


Consistency
-----------

Change Cassandra's distributed conistency requirement.

#### Available

* any
* one
* two
* three
* quorum
* all
* localquorum
* eachquorum
* serial
* localserial

#### Argument

    Short:  `-c`
    Long:   `--consistency`

#### Default: `quorum`




Informational Commands
======================

* [Describe](#describe) -- describes the entire system, a keyspace, or keyspace.table in pretty-printed JSON
* [Backfill](#backfill) -- generates a series of migrations to get from the current table layout to some desired one
* [List](#list) -- print report of completed/remaining migrations


Describe
--------

Prints a JSON representation of the requested schemas.

Available argument formats include `all`, `none`, `{keyspace}`, and `{keyspace}.{table}`.

#### __cmm --describe all__:

````json
[
    {
        "Name": "keyspace_name",
        "Options": {
            "replication_factor": "3",
            "strategy_option_2": "value",
            "etc": "etc"
        },
        "Tables": [
            {
                "Name": "table_name",
                "Columns": [
                    {
                        "Name": "column_name",
                        "Type": "column_type",
                        "Primary": true
                    }
                ]
            }
        ]
    }
]
````

#### __cmm --describe keyspace__:

````json
{
    "Name": "keyspace_name",
    "Options": {
        "replication_factor": "3",
        "strategy_option_2": "value",
        "etc": "etc"
    },
    "Tables": [
        {
            "Name": "table_name",
            "Columns": [
                {
                    "Name": "column_name",
                    "Type": "column_type",
                    "Primary": true
                }
            ]
        }
    ]
}
````

#### __cmm --describe keyspace.table__:

````json
{
    "Name": "table_name",
    "Columns": [
        {
            "Name": "column_name",
            "Type": "column_type",
            "Primary": true
        }
    ]
}
````

where `column_type` consists of the all-caps options:

* TEXT
* INT32
* INT64
* FLOAT
* UUID
* TIMEUUID
* BYTE
* MAP<X,Y>
* SET<X>
* ... I'm probably missing a few but you get the idea



Backfill
--------

Backfilling creates all the migrations needed to get from the current table state, to some desired state as described by a JSON file.

If you `--backfill` a columnfamily that does not exist, a `CREATE TABLE` query will be output assuming your schema.json file is valid.


#### Example JSON Schema: (more in test/schemas)

````json
{
  "id":           "UUID PRIMARY KEY",
  "name":         "TEXT",
  "email":        "TEXT",
  "date_joined":  "TIMESTAMP"
}
````

### How It Works

Migrations are spit out to the console one-per-line.

Let's pretend we have an existing table `main.users` that looks similar to the above example, except:

* `date_joined` not in table
* `post_count` in table, but not in the target schema
* `user_email` renamed to `email` in target schema

Running `cmm --backfill main.users --file schema/users.json` will print the following lines:

    ALTER TABLE main.users ADD date_joined TIMESTAMP;
    ALTER TABLE main.users DROP post_count;
    ALTER TABLE main.users DROP user_email;
    ALTER TABLE main.users ADD email TEXT;

__NOTE:__ for minimal complication, renames are a sequential remove+add -- will be combined in future releases

Any half-decent developer is going to be looking awkwardly at the `DROP` and subsequent `ADD`. You should.

Running a schema migration is that, a schema migration. If you are concerned with data loss, Apply the migrations in steps along with __in-house__ data migration.

Data migration is a bit out of scope I feel. It would be a fantastic feature, and potentially a separate project I will integrate/call.

The basic algorithm would be:

1. Copy (id, data) from any dropped columns
2. DROP the columns
3. ADD the new columns
3. Copy the (id, data) to the new columns

We could do this on a per-run basis as in detecting rename and generating a script... but what language, environment, etc?

I feel `cmm` provides the tools to implement the above steps in a small number of semantic lines.

If someone would like to issue a pull request with a `bash` or similar style script that implements the above that would be fantastic. I may get around to it.

### Saving

Naturally, you'll need to save these migrations to actually run them.

__Option 1:__


Use `--output` to specify directory.

This generates one file per migration following `{RFC3339}_some_descriptive_text.cql` format. ___note:___ generated files included nanoseconds as part of the RFC3339 prefix. We just generate them too fast to use milliseconds :)

Example:

    cmm -b main.users -f users.json -o ./migrations

__Option 2:__

Directing (>) or piping (|) to a file at the command line.

Direct to file with or without overwrite:

1. `cmm -b main.users -f users.json > single_migration.cql`
2. `cmm -b main.users -f users.json >> multiple_migrations.cql`

Pipe to file with or without overwrite:

1. `cmm -b main.users -f users.json | tee single_migration.cql`
2. `cmm -b main.users -f users.json | tee -a multiple_migrations.cql`


List
----

List which migrations _have_ and _have not_ been run.

#### Option 1: ANSI-Colored Output

A standard `--list` will print one migration per line.

This line will follow the format

    +/-  NAME

similar to a diff file (where + is done and - is remaining).

If your terminal supports ANSI coloring, completed migrations will be printed in green whereas remaining migrations are printed in red.


#### Option 2: JSON Output

Using either `-j` or `--list.json`, a json object of the format

````json
{
    "Complete": [
        {
            "Name": "FILENAME",
            "Path": "PATH_TO_MIGRATION_FILE",
            "Query": "CQL_QUERY_STATEMENTS"
        }
    ],

    "Remaining": [
        {
            "Name": "FILENAME",
            "Path": "PATH_TO_MIGRATION_FILE",
            "Query": "CQL_QUERY_STATEMENTS"
        }
    ]
}
````


Testing
=======

Test cases can always be expanded.

Al the moment they are rapidly expanding, but please feel free to contribute.

Many functions/pseudocommands are tested implicitly rather than explicitly but this will change as cases rather than infrastructure have become a focus.


Testing In A VM
---------------

All testing has been automated using a VM initiated by [Vagrant](http://vagrantup.com) and a 3-instance cluster thanks to [docker](http://docker.io).

Vagrant depends on virtualbox to spin up a VM using Ubuntu Precise, install `docker`, pull a [cassandra docker image](http://github.com/zmarcantel/docker-cassandra), and use that image to sping up a 3-container cluster.

The `Vagrantfile` in the root directory controls all of this. To initiate it:

    # using vagrant
    vagrant up

    # using make
    make test

    # to re-run tests, either of the below
    make test
    vagrant provision



Testing With Your Cluster
-------------------------

If you are working on a system you do not wish to start an isolated test environment on, the following steps will describe how to test from a local command prompt.

After cloning the source, set the following environment variable as so:

    export TEST_HOSTS=192.168.50.100,127.0.0.1:1234,cassHostname

The tests check for the `TEST_HOSTS` variable and use the comma separated list exactly the same as the peer command to the binary.

You can also simply prefix `go test` with the variable as so:

    TEST_HOSTS=192.168.50.100 go test

Alternatively, alter the source of `cmm_test.go` which contains a fallback IP (the one immediately above). If you submit a pull request with this as a line change I will remove the request. Please don't make me a liar in the README.
