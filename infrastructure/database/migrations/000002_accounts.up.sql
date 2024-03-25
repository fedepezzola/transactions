CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL,
    account_number VARCHAR NOT NULL,
    balance FLOAT NOT NULL,
    PRIMARY KEY (id)
);
