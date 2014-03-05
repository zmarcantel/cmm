cmm
===

Cassandra Migrations Manager that makes managing schemas a breeze.

1. Loads `.cql` files from a directory (or many nested directories)
2. Sorts them to ensure they are run in order
3. Executes any new migrations

Supports connection pooling and setting protocol version.

See the [Options](#command-flags) section for information on how to use it.

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