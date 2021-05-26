# Cadence Samples
These are some samples to demonstrate various capabilities of Cadence client and server.  You can learn more about cadence at:
* Cadence: https://github.com/uber/cadence
* Cadence Client: https://github.com/uber-go/cadence-client

## Prerequisite
Run Cadence Server

See instructions for running the Cadence Server: https://github.com/uber/cadence/blob/master/README.md

## Get cadence

- git clone https://github.com/uber/cadence
- cd cadence

## Build tools

- make bins

## Setup db

- update install-schema-mysql to use proper mysql access (host, pwd, user)
- make install-schema-mysql

## Run cadence-server

- cp config/development_mysql.yaml config/development.yaml
- update development.yaml to use proper mysql access (host, pwd, user)
- ./cadence-server start --services=frontend,matching,history,worker

## Create domain register

- Register a domain: cadence --domain simpledomain domain register