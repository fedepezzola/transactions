# Transactions
This project reads a file that contains bank transactions and sends an email with the summary.

## Running the project 

First thing to do is to setup the configuration. Copy the configuration example into a `.env` file and complete the configuration parameters.
```sh
cp .env.example .env
```
Database parameters can be left as is for the local environment.  
Email parameters will need some credentials from your side to make it work. You can check how to create an app password for your gmail account [here](https://support.google.com/accounts/answer/185833?visit_id=638469871835911446-731919034&p=InvalidSecondFactor&rd=1).

Then source your file to set the environment variables (sourcing env variables may vary between shells):
```sh
source .env
# or 
export $(grep -v '^#' .env | xargs -d '\n')
```

### Setting up the database
To create the database server run:
```sh
docker compose up -d db
```

Then create the database running:
```sh
make create-database
```

Install migrations tool and run the database migrations:
```sh
go get -u -d github.com/golang-migrate/migrate

make migrate
```

### Build and Run
The program accepts the transaction file in either one of 2 ways. File can be sent by stdin or using the parameter `-f <filename>` with transactions csv file.

Using make build will create an executable in the dist folder
```sh
make build
```
that we can run like:
```sh
./dist/transactions -f txns.csv
#or 
./dist/transactions < txns.csv
```
We can also run:
```sh
go run transactions.go -f txns.csv
```
More configuration options running:
```sh
go run transactions.go --help
```

### Using docker
Build the docker image:
```sh
docker build . -t transactions:latest
```
Run using compose, it runs and uses `txns.csv` file included in this repo:
```sh
docker compose up app
```
Run using docker run:
```sh
docker run -i --env-file ./.env  -e TRANSACTIONS_DB_HOST=transactions-db --network transactions-network transactions:latest < ./txns.csv 
```