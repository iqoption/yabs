.PHONY: default deps collector processor all clean

OUTPUT_DIR=output
BIN_DIR=bin

VERSION=$(shell git describe --abbrev=0 --tags)
BUILD=$(shell git rev-parse HEAD)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Build=$(BUILD)"

deps:
	@echo Get deppends
	glide install

outputdir:
	mkdir -p ./$(OUTPUT_DIR)/$(BIN_DIR)

collector: outputdir
	@echo Build collector
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BIN_DIR)/collector ./collector/

processor: outputdir
	@echo Build processor
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BIN_DIR)/processor ./processor/
	
cli: outputdir
	@echo Build processor
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BIN_DIR)/yabs-cli ./cli/
	
all: cli collector processor 

clean:
	rm -fr $(OUTPUT_DIR)

.DEFAULT_GOAL := all
