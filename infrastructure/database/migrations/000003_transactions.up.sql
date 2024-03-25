CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL,
    account_id INT NOT NULL,
    processing_timestamp TIMESTAMP NOT NULL,
    file_transaction_id INT NOT NULL, 
    transaction_day INT NOT NULL,
    transaction_month INT NOT NULL,
    amount FLOAT NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_account
      FOREIGN KEY(account_id) 
        REFERENCES accounts(id)
);