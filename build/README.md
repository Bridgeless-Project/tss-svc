# /build

## Description
Contains the Dockerfile and local build setup for the project

## Components
- `Dockerfile`: Contains the Docker setup for the project
- `Makefile`: Contains the commands for building and running the project locally
- `docker-compose.yml`: Contains the containers configuration for running the TSS network locally
- `/configs`: Contains the configuration files for the TSS nodes
- `/scripts`: Contains the scripts for setting up the TSS network and its components

## Local Run
To spin up the local TSS network, run the following command:

```bash
make docker-up
```

It will update the config files for TSS nodes with new starting times and run the configured `docker-compose` setup.