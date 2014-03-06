-- create user table

CREATE TABLE main.users (
    id              UUID PRIMARY KEY,
    first_name      TEXT,
    last_name       TEXT,
    email           TEXT,

    join_date       TIMESTAMP
);