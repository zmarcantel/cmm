package main

import (
    "os"
    "fmt"
    "strings"
    "testing"
    "io/ioutil"

    "github.com/tux21b/gocql"

    "github.com/zmarcantel/cmm/db"
)

var GOOD_HOSTS []string

func TestSetup(t *testing.T) {
    var testHosts = os.Getenv("TEST_HOSTS")
    fmt.Printf("Test Hosts: %s\n", testHosts)

    if len(testHosts) > 0 {
        GOOD_HOSTS = strings.Split(testHosts, ",")
        _ = os.Setenv("TEST_HOSTS", "")
    } else {
        GOOD_HOSTS = []string{ "192.168.50.100" } //"192.168.33.100", "192.168.33.101", "192.168.33.150"}
    }
}

func TestConfig(t *testing.T) {
    var testConf = "./test/config.json"
    Opts.Config = testConf
    handleConfig()

    // check protocol
    if (Opts.Protocol != 2) {
        t.Error(
            "For", "Opts.Protocol",
            "expected", 2,
            "got", Opts.Protocol,
        )
    }

    // check consistency
    if (Opts.Consistency != "quorum") {
        t.Error(
            "For", "Opts.Consistency",
            "expected", "quorum",
            "got", Opts.Consistency,
        )
    }

    // but we may need to alter this depending on size of test cluster
    if (len(GOOD_HOSTS) == 1) {
        Consistency = gocql.One
    }

    // check peers array is loaded correctly
    var configHosts = strings.Split(Opts.Hosts, ",")
    if (len(configHosts) != 3) {
        t.Error(
            "For", "len(configHosts)",
            "expected", 3,
            "got", len(configHosts),
        )
    }
    if (configHosts[0] != "127.0.0.1") {
        t.Error(
            "For", configHosts[0],
            "expected", "127.0.0.1",
            "got", configHosts[0],
        )
    } else if (configHosts[1] != "192.168.33.100:9000") {
        t.Error(
            "For", configHosts[1],
            "expected", "192.168.33.100:9000",
            "got", configHosts[1],
        )
    } else if (configHosts[2] != "dbtwo") {
        t.Error(
            "For", configHosts[2],
            "expected", "dbtwo",
            "got", configHosts[2],
        )
    }


    // check migrations directory
    if (Opts.Migrations != "./test") {
        t.Error(
            "For", "Opts.Migrations",
            "expected", "./test",
            "got", Opts.Migrations,
        )
    }


    // check global delay
    if (Opts.Delay != 0) {
        t.Error(
            "For", "Opts.Delay",
            "expected", 0,
            "got", Opts.Delay,
        )
    }


    // check file input
    if (Opts.File != "./test/schemas/users.json") {
        t.Error(
            "For", "Opts.File",
            "expected", "./test/schemas/users.json",
            "got", Opts.File,
        )
    }

    // check file output
    if (Opts.Output != "./test/main/users") {
        t.Error(
            "For", "Opts.Output",
            "expected", "./test/main/users",
            "got", Opts.Output,
        )
    }
}


func TestHosts(t *testing.T) {
    BuildHosts(Opts.Hosts)

    if (len(Hosts) != 3) {
        t.Error(
            "For", "len(Hosts)",
            "expected", 3,
            "got", len(Hosts),
        )
    }

    if (Hosts[0] != "127.0.0.1") {
        t.Error(
            "For", Hosts[0],
            "expected", "127.0.0.1",
            "got", Hosts[0],
        )
    } else if (Hosts[1] != "192.168.33.100:9000") {
        t.Error(
            "For", Hosts[1],
            "expected", "192.168.33.100:9000",
            "got", Hosts[1],
        )
    } else if (Hosts[2] != "dbtwo") {
        t.Error(
            "For", Hosts[2],
            "expected", "dbtwo",
            "got", Hosts[2],
        )
    }
}

func TestConnection(t *testing.T) {
    Opts.Hosts = strings.Join(GOOD_HOSTS, ",")
    BuildHosts(Opts.Hosts)

     _, Session = connectCluster()
    db.Init(Session)

    if _, err := db.Keyspace("system") ; err != nil {
        t.Error(
            "For", "DB Connect, get system keyspace description",
            "expected", nil,
            "got", err,
        )
    }

    CreateMigrationTable(Session)
}

func TestLoadMigrations(t *testing.T) {
    GetMigrationFiles(Opts.Migrations)
    Consistency = gocql.Quorum;

    if (Migrations.Len() != 4) {
        t.Error(
            "For", "Migrations.Len()",
            "expected", 4,
            "got", Migrations.Len(),
        )
    }

    //
    // Create cmm_main keyspace
    //

    if (Migrations[0].Name != "2014-03-01T05-44-32.070Z_add_main_keyspace.cql") {
        t.Error(
            "For", "Migrations[0].Name",
            "expected", "2014-03-01T05-44-32.070Z_add_main_keyspace.cql",
            "got", Migrations[0].Name,
        )
    } else if (Migrations[0].Path != "test/main/keyspaces/2014-03-01T05-44-32.070Z_add_main_keyspace.cql") {
        t.Error(
            "For", "Migrations[0].Path",
            "expected", "test/main/keyspaces/2014-03-01T05-44-32.070Z_add_main_keyspace.cql",
            "got", Migrations[0].Path,
        )
    }



    //
    // Create cmm_main.users table
    //

    if (Migrations[1].Name != "2014-03-02T05-44-32.070Z_create_user_table.cql") {
        t.Error(
            "For", "Migrations[1].Name",
            "expected", "2014-03-02T05-44-32.070Z_create_user_table.cql",
            "got", Migrations[1].Name,
        )
    } else if (Migrations[1].Path != "test/main/users/2014-03-02T05-44-32.070Z_create_user_table.cql") {
        t.Error(
            "For", "Migrations[1].Path",
            "expected", "test/main/users/2014-03-02T05-44-32.070Z_create_user_table.cql",
            "got", Migrations[1].Path,
        )
    }



    //
    // Create cmm_main.items table
    //

    if (Migrations[2].Name != "2014-03-02T06-13-03.495Z_create_item_table.cql") {
        t.Error(
            "For", "Migrations[2].Name",
            "expected", "2014-03-02T06-13-03.495Z_create_item_table.cql",
            "got", Migrations[2].Name,
        )
    } else if (Migrations[2].Path != "test/main/items/2014-03-02T06-13-03.495Z_create_item_table.cql") {
        t.Error(
            "For", "Migrations[2].Path",
            "expected", "test/main/items/2014-03-02T06-13-03.495Z_create_item_table.cql",
            "got", Migrations[2].Path,
        )
    }



    //
    // Alter cmm_main.users table
    //

    if (Migrations[3].Name != "2014-03-02T06-14-04.626Z_add_items_to_user_table.cql") {
        t.Error(
            "For", "Migrations[3].Name",
            "expected", "2014-03-02T06-14-04.626Z_add_items_to_user_table.cql",
            "got", Migrations[3].Name,
        )
    } else if (Migrations[3].Path != "test/main/users/2014-03-02T06-14-04.626Z_add_items_to_user_table.cql") {
        t.Error(
            "For", "Migrations[3].Path",
            "expected", "test/main/users/2014-03-02T06-14-04.626Z_add_items_to_user_table.cql",
            "got", Migrations[3].Path,
        )
    } else if complete, err := Migrations[3].IsComplete() ; complete {
        if (err != nil) {
            t.Error(
                "For", "Migrations[3].IsComplete() Error",
                "expected", nil,
                "got", err,
            )
        }

        t.Error(
            "For", "Migrations[3].IsComplete()",
            "expected", false,
            "got", complete,
        )
    }
}


func TestMigrations(t *testing.T) {
    DoMigrations(Migrations, SettleTime)

    for _, mig := range Migrations {
        if complete, err := mig.IsComplete() ; complete == false {
            t.Error(
                "For", "Complete: " + mig.Name,
                "expected", true,
                "got", false,
            )
        } else if (err != nil) {
            t.Error(
                "For", "Completion Error",
                "expected", nil,
                "got", err,
            )
        }
    }
}

func TestBackfillAdd(t *testing.T) {
    Opts.Backfill = "cmm_main.users"
    Opts.File = "test/schemas/users_fields_added.json"

    var migs = Backfill(Opts.Backfill, Opts.File)
    var acceptable = []string{
        "ALTER TABLE cmm_main.users ADD purchases SET<UUID>;",
        "ALTER TABLE cmm_main.users ADD ratings MAP<UUID,FLOAT>;",
        "ALTER TABLE cmm_main.users ADD comments LIST<TEXT>;",
    }

    for _, mig := range migs {
        if (len(mig.Name) == 0) {
            t.Error(
                "For", "len(mig.Name)",
                "expected", ">1",
                "got", len(mig.Name),
            )
        }

        if (!contains(acceptable, mig.Query)) {
            t.Error(
                "For", "migration query",
                "expected", acceptable,
                "got", mig.Query,
            )
        }
    }
}

func TestBackfillRemove(t *testing.T) {
    Opts.Backfill = "cmm_main.users"
    Opts.File = "test/schemas/users_fields_removed.json"

    var migs = Backfill(Opts.Backfill, Opts.File)
    var acceptable = []string{
        "ALTER TABLE cmm_main.users DROP items;",
        "ALTER TABLE cmm_main.users DROP join_date;",
    }

    for _, mig := range migs {
        if (len(mig.Name) == 0) {
            t.Error(
                "For", "len(mig.Name)",
                "expected", ">1",
                "got", len(mig.Name),
            )
        }

        if (!contains(acceptable, mig.Query)) {
            t.Error(
                "For", "migration query",
                "expected", acceptable,
                "got", mig.Query,
            )
        }
    }
}


func TestBackfillMixed(t *testing.T) {
    Opts.Backfill = "cmm_main.users"
    Opts.File = "test/schemas/users_fields_mixed.json"

    var migs = Backfill(Opts.Backfill, Opts.File)
    var acceptable = []string{
        "ALTER TABLE cmm_main.users DROP items;",
        "ALTER TABLE cmm_main.users DROP join_date;",
        "ALTER TABLE cmm_main.users ADD purchases SET<UUID>;",
        "ALTER TABLE cmm_main.users ADD ratings MAP<UUID,FLOAT>;",
        "ALTER TABLE cmm_main.users ADD comments LIST<TEXT>;",
    }

    for _, mig := range migs {
        if (len(mig.Name) == 0) {
            t.Error(
                "For", "len(mig.Name)",
                "expected", ">1",
                "got", len(mig.Name),
            )
        }

        if (!contains(acceptable, mig.Query)) {
            t.Error(
                "For", "migration query",
                "expected", acceptable,
                "got", mig.Query,
            )
        }
    }
}

func TestDescribeUsers(t *testing.T) {
    Opts.Describe = "cmm_main.users"

    var output = Describe(Opts.Describe)
    var known, err = ioutil.ReadFile("test/outputs/cmm_main.users.json")
    if err != nil {
        fmt.Printf("Error while loading expected output of DESCRIBE cmm_main.users:\n%s\n\n", err)
        os.Exit(1)
    }

    var outputLines = strings.Split(output, "\n")
    var knownLines = strings.Split(string(known), "\n")
    for i, line := range knownLines {
        if line != outputLines[i] {
            t.Error(
                "For", "DESCRIBE line in JSON",
                "\nexpected", strings.TrimSpace(line),
                "\n     got", strings.TrimSpace(outputLines[i]),
            )
        }
    }
}


func TestClose(t *testing.T) {
    var keyErr = Session.Query(`DROP KEYSPACE cmm_main`).Exec()
    if keyErr != nil {
        t.Error(
            "For", "DROP cmm_main",
            "expected", nil,
            "got", keyErr,
        )
    }

    for _, mig := range Migrations {
        if err := Session.Query(`DELETE FROM migrations.completed WHERE name = ?`, mig.Name).Exec() ; err != nil {
            t.Error(
                "For", "Remove completion of: " + mig.Name,
                "expected", nil,
                "got", err,
            )
        }
    }


    Session.Close()
}


func contains(list []string, target string) bool {
    for _, val := range list {
        if target == val {
            return true
        }
    }

    return false
}
