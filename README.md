cmm
===

Cassandra Migrations Manager that makes managing schemas a breeze.

1. Loads `.cql` files from a directory (or many nested directories)
2. Sorts them to ensure they are run in order
3. Executes any new migrations

Production Focused Features

1. [Connection pooling](#hosts)
2. [Set per-migration delay times](#migration-file)
3. [Schema to JSON](#describe)
4. [Setting protocol version](#protocol)


See the [Examples](#examples) section and see how easy it can be.

See the [Options](#command-flags) section for all the available settings.

See the [Commands](#informational-commands) section for all the query tools.


### Planned Features:

* Load peers from file
* Model->Migration backfill
  * Intake JSON, csv, other? of field->type mapping
  * Inspect current table
  * Generate migrations
* Support Solr
  * For the [DataStax](http://www.datastax.com/what-we-offer/products-services/datastax-enterprise) users
  * Upload `schema.xml`
  * Core exist: `reload` Else: `create`


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





Command Flags
=============

    Verbose       short: "v"   long: "verbose"        description: "Show verbose log information. Supports -v[vvv] syntax."

    Protocol      short: "P"   long: "protocol"       description: "Protocol version to use [1 or 2]"
    Consistency   short: "c"   long: "consistency"    description:"Cassandra consistency to use: one, quorum, all, etc"`

    Hosts         short: "p"   long: "peers"          description: "Comma-serparated list of Cassandra hosts (hostname:port)"
    Migrations    short: "m"   long: "migrations"     description: "Directory containing timestamp-prefixed migration files"

    Delay         short: "d"   long: "delay"          description: "Wait n milliseconds between migrations"

    
    // Pseudo-Commands -- results in information, not commands being run

    Help          short: "h"   long: "help"           description: "Show the help menu"
    Describe      short: "D"   long: "describe"       description: "Prints a JSON represntation of ['all', 'none','{keyspace}', '{keyspace}.{table}']"


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
    * prefix title with ISO-8601 timestamp
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
* MAP(X,Y)
* SET(X)
* ... I'm probably missing a few but you get the idea