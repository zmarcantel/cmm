--delay:500

CREATE KEYSPACE cmm_main
WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 }