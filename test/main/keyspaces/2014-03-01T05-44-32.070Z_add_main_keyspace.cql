--delay:500

CREATE KEYSPACE main
WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 }