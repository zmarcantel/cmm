package db

import (
    "os"
    "fmt"
    "strings"
    "strconv"
    "encoding/json"

    "github.com/tux21b/gocql"
)

type ColumnDescriptor struct {
    Name        string
    Type        string
    Primary     bool
}

type TableDescriptor struct {
    Name        string
    Keyspace    string
    Columns     []ColumnDescriptor
}

type KeyspaceDescriptor struct {
    Name        string
    Options     map[string]interface{}
    Tables      []TableDescriptor
}

// save a DB session on init
var Session *gocql.Session

//
//  Init
//      Initialize the DB lookup -- mainly saves the session
//
func Init(session *gocql.Session) {
    Session = session
}


//
//  AllKeyspaces
//      Retrive all keyspaces and all nested attributes
//
func AllKeyspaces() []KeyspaceDescriptor {
    var name string
    var options string
    var keyspaces = make([]KeyspaceDescriptor, 0)

    // create an iterator over keyspace descriptors
    var iter = Session.Query(`SELECT keyspace_name, strategy_options FROM system.schema_keyspaces;`).Iter()

    // iterate over the results
    for iter.Scan(&name, &options) {
        var parsedOption = parseOptions(options)

        // make the keyspace descriptor
        keyspaces = append(keyspaces, KeyspaceDescriptor{
            Name:       name,
            Options:    parsedOption,
            Tables:     AllTables(name),
        })
    }
    if err := iter.Close(); err != nil {
        fmt.Printf("ERROR: could not get keyspaces list:\n%s\n", err)
        os.Exit(1)
    }

    return keyspaces
}


//
//  Keyspace
//      Get a single keyspace
//      Includes a list of tables and their columns
//
func Keyspace(name string) KeyspaceDescriptor {
    var options string

    // create an iterator over keyspace descriptors
    var err = Session.Query(`SELECT strategy_options FROM system.schema_keyspaces WHERE keyspace_name = ?;`, name).Scan(&options)
    if err != nil {
        fmt.Printf("ERROR: could not get keyspace:\n%s\n", err)
        os.Exit(1)
    }

    return KeyspaceDescriptor{
        Name:       name,
        Options:    parseOptions(options),
        Tables:     AllTables(name),
    }
}


//
//  parseOptions
//      Parse the strategy options JSON of the table
//
func parseOptions(options string) map[string]interface{} {
    // map of top-level JSON keys to first-level values (or stringified object)
    var finalMap map[string]interface{}

    // if we have anything to parse
    if (len(options) > 0) {
        finalMap = make(map[string]interface{})

        // create a map of first level keys to first level stringified values
        var objmap map[string]*json.RawMessage
        json.Unmarshal([]byte(options), &objmap)

        // iterate over keys
        for i := range objmap {
            if (*objmap[i] == nil) { continue }

            // attempt to unmarshal the first-level value
            // catches stringified objects
            var value interface{}
            var err = json.Unmarshal(*objmap[i], value)
            if (err != nil) {
                // if not a map, but a value
                // strings, numbers, and any other non-object type
                var err error
                value, err = strconv.Unquote(string(*objmap[i]))
                if (err != nil) {
                    fmt.Printf("ERROR: could not unquote value\n%s\n", err)
                    os.Exit(1)
                }
            }

            // insert into map
            finalMap[i] = value
        }
    }

    return finalMap
}


//
//  AllTables
//      Get descriptors of all the tables in a given keyspace
//
func AllTables(keyspace string) (result []TableDescriptor) {
    var name string

    // create an iterator over keyspace descriptors
    var iter = Session.Query(`SELECT columnfamily_name FROM system.schema_columnfamilies WHERE keyspace_name = ?;`, keyspace).Iter()

    // iterate over the results
    for iter.Scan(&name) {
        result = append(result, TableDescriptor{
            Name:           name,
            Keyspace:       keyspace,
            Columns:        ListColumns(keyspace, name),
        })
    }
    if err := iter.Close(); err != nil {
        fmt.Printf("ERROR: could not get keyspaces list:\n%s\n", err)
        os.Exit(1)
    }

    return result
}


//
//  Table
//      Get the table descriptor for a single table in a given keyspace
//
func Table(keyspace, table string) (result TableDescriptor) {
    return TableDescriptor{
        Name:           table,
        Keyspace:       keyspace,
        Columns:        ListColumns(keyspace, table),
    }
}


//
//  ListColumns
//      Get mapping of name -> type mapping for all columns in keyspace.table
//
func ListColumns(keyspace, table string) (result []ColumnDescriptor) {
    var name string
    var columnType string
    var datatype string

    // create an iterator over keyspace descriptors
    var iter = Session.Query(`SELECT column_name,type,validator FROM system.schema_columns WHERE keyspace_name = ? AND columnfamily_name = ?;`, keyspace, table).Iter()

    // iterate over the results
    for iter.Scan(&name,&columnType,&datatype) {
        var parts = strings.Split(datatype, "Type")
        for i, val := range parts {
            parts[i] = val[strings.LastIndex(val, ".") + 1:]
        }

        var finalType string
        if (len(parts) == 2) { // target + split
            finalType = parts[0]
        } else if (len(parts) == 3) { // (target + split)*2
            finalType = fmt.Sprintf("%s(%s)", parts[0], parts[1])
        } else if (len(parts) == 4) { // (target + split)*3
            finalType = fmt.Sprintf("%s(%s, %s)", parts[0], parts[1], parts[2])
        } else {
            finalType = columnType
        }

        finalType = strings.Replace(finalType, "UTF8", "Text", -1)

        result = append(result, ColumnDescriptor{
            Name:           name,
            Type:           strings.ToUpper(finalType),
            Primary:        columnType == "partition_key",
        })
    }
    if err := iter.Close(); err != nil {
        fmt.Printf("ERROR: could not get keyspaces list:\n%s\n", err)
        os.Exit(1)
    }

    return result
}
