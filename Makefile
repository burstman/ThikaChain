# Makefile to manage Hyperledger Fabric environment variables

# Base paths
FABRIC_SAMPLES := $(HOME)/fabric-samples
TEST_NETWORK := $(FABRIC_SAMPLES)/test-network
BIN_DIR := $(FABRIC_SAMPLES)/bin
CONFIG_DIR := $(FABRIC_SAMPLES)/config

# Default to localhost, but allow overriding for remote servers
PEER_HOST ?= localhost
ORG1_PORT ?= 7051
ORG2_PORT ?= 9051
ORG3_PORT ?= 11051

.PHONY: env-org1 env-org2 env-org3 help package clean

help:
	@echo "Usage: eval \$$(make <target>)"
	@echo ""
	@echo "Targets:"
	@echo "  env-org1    Set environment for Organization 1 (Port 7051)"
	@echo "  env-org2    Set environment for Organization 2 (Port 9051)"
	@echo "  env-org3    Set environment for Organization 3 (Port 11051)"
	@echo "  package     Package chaincode for production deployment"
	@echo "  clean       Remove generated package files"

env-org1:
	@echo "export PATH=$(BIN_DIR):\$$PATH"
	@echo "export FABRIC_CFG_PATH=$(CONFIG_DIR)"
	@echo "export CORE_PEER_TLS_ENABLED=true"
	@echo "export CORE_PEER_LOCALMSPID=Org1MSP"
	@echo "export CORE_PEER_TLS_ROOTCERT_FILE=$(TEST_NETWORK)/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
	@echo "export CORE_PEER_MSPCONFIGPATH=$(TEST_NETWORK)/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
	@echo "export CORE_PEER_ADDRESS=$(PEER_HOST):$(ORG1_PORT)"
	@echo "export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com"

env-org2:
	@echo "export PATH=$(BIN_DIR):\$$PATH"
	@echo "export FABRIC_CFG_PATH=$(CONFIG_DIR)"
	@echo "export CORE_PEER_TLS_ENABLED=true"
	@echo "export CORE_PEER_LOCALMSPID=Org2MSP"
	@echo "export CORE_PEER_TLS_ROOTCERT_FILE=$(TEST_NETWORK)/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
	@echo "export CORE_PEER_MSPCONFIGPATH=$(TEST_NETWORK)/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"
	@echo "export CORE_PEER_ADDRESS=$(PEER_HOST):$(ORG2_PORT)"
	@echo "export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com"

env-org3:
	@echo "export PATH=$(BIN_DIR):\$$PATH"
	@echo "export FABRIC_CFG_PATH=$(CONFIG_DIR)"
	@echo "export CORE_PEER_TLS_ENABLED=true"
	@echo "export CORE_PEER_LOCALMSPID=Org3MSP"
	@echo "export CORE_PEER_TLS_ROOTCERT_FILE=$(TEST_NETWORK)/organizations/peerOrganizations/org3.example.com/peers/peer0.org3.example.com/tls/ca.crt"
	@echo "export CORE_PEER_MSPCONFIGPATH=$(TEST_NETWORK)/organizations/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp"
	@echo "export CORE_PEER_ADDRESS=$(PEER_HOST):$(ORG3_PORT)"
	@echo "export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org3.example.com"

# Production: Package the chaincode
package:
	peer lifecycle chaincode package thika.tar.gz --path . --lang golang --label thika_1.0

clean:
	rm -f thika.tar.gz