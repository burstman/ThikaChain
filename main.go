package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// HistoryContract defines the Smart Contract structure
type HistoryContract struct {
	contractapi.Contract
}

// VerificationRecord represents the data shared between parties
type VerificationRecord struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Party       string `json:"party"`     // The organization/user currently acting
	Status      string `json:"status"`    // e.g., "CREATED", "VERIFIED", "REJECTED"
	Timestamp   string `json:"timestamp"` // Application level timestamp
	// You can add more fields here to match organization requirements (e.g., Location, BatchID)
}

// HistoryQueryResult structure used for returning history data
type HistoryQueryResult struct {
	TxId      string              `json:"txId"`
	Timestamp time.Time           `json:"timestamp"`
	IsDelete  bool                `json:"isDelete"`
	Record    *VerificationRecord `json:"record"`
}

// InitLedger adds a base set of records to the ledger
func (s *HistoryContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// Use Transaction Timestamp for determinism, not time.Now()
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	timestampStr := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).Format(time.RFC3339)

	records := []VerificationRecord{
		{ID: "REC001", Description: "Initial Contract Draft", Party: "Org1", Status: "CREATED", Timestamp: timestampStr},
		{ID: "REC002", Description: "Shipping Manifest", Party: "Org2", Status: "PENDING", Timestamp: timestampStr},
	}

	for _, record := range records {
		assetJSON, err := json.Marshal(record)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(record.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// BatchImport allows uploading multiple records at once.
// This is the correct way to import data fetched from an external API:
// 1. The Client App (off-chain) fetches the data from the API.
// 2. The Client App calls this function passing the data as a JSON string.
func (s *HistoryContract) BatchImport(ctx contractapi.TransactionContextInterface, data string) error {
	var records []VerificationRecord
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return fmt.Errorf("failed to unmarshal data: %v", err)
	}

	for _, record := range records {
		assetJSON, err := json.Marshal(record)
		if err != nil {
			return err
		}

		// Production Safety: Check if record exists to prevent accidental data loss
		exists, err := s.RecordExists(ctx, record.ID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("record %s already exists - batch import aborted", record.ID)
		}

		if err := ctx.GetStub().PutState(record.ID, assetJSON); err != nil {
			return fmt.Errorf("failed to put to world state for ID %s: %v", record.ID, err)
		}
	}
	return nil
}

// CreateRecord issues a new record to the world state
func (s *HistoryContract) CreateRecord(ctx contractapi.TransactionContextInterface, id string, description string, status string) error {
	exists, err := s.RecordExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the record %s already exists", id)
	}

	// Get the identity of the submitter (the Party)
	clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client identity: %v", err)
	}

	// Use Transaction Timestamp
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	record := VerificationRecord{
		ID:          id,
		Description: description,
		Party:       clientIdentity,
		Status:      status,
		Timestamp:   time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).Format(time.RFC3339),
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, recordJSON)
}

// UpdateRecord allows a party to update the status or description, creating a new history entry
func (s *HistoryContract) UpdateRecord(ctx contractapi.TransactionContextInterface, id string, description string, status string) error {
	exists, err := s.RecordExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the record %s does not exist", id)
	}

	clientIdentity, _ := ctx.GetClientIdentity().GetMSPID()

	// Use Transaction Timestamp
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	// Overwrite the record. Fabric automatically keeps the old version in the history.
	updatedRecord := VerificationRecord{
		ID:          id,
		Description: description,
		Party:       clientIdentity,
		Status:      status,
		Timestamp:   time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).Format(time.RFC3339),
	}

	recordJSON, err := json.Marshal(updatedRecord)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, recordJSON)
}

// GetRecordHistory returns the chain of custody/history for a specific record
// This is the core function for your verification use case.
func (s *HistoryContract) GetRecordHistory(ctx contractapi.TransactionContextInterface, id string) ([]HistoryQueryResult, error) {

	// GetHistoryForKey is a Fabric API that retrieves all state changes for a key
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var verificationRecord *VerificationRecord
		if !response.IsDelete {
			if err := json.Unmarshal(response.Value, &verificationRecord); err != nil {
				return nil, err
			}
		}

		// Convert Fabric timestamp to Go time
		timestamp := response.Timestamp.AsTime()

		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: timestamp,
			IsDelete:  response.IsDelete,
			Record:    verificationRecord,
		}
		records = append(records, record)
	}

	return records, nil
}

// QueryOtherLedger allows this contract to read data from a different channel's ledger.
// Note: Cross-channel invocations are READ-ONLY. You cannot write to the other ledger.
// This uses Fabric's internal gRPC protocol, not HTTP.
func (s *HistoryContract) QueryOtherLedger(ctx contractapi.TransactionContextInterface, channelName string, chaincodeName string, functionName string, arg string) (string, error) {
	// Arguments must be converted to [][]byte
	// The first argument is the function name to call on the target chaincode
	chainCodeArgs := [][]byte{[]byte(functionName), []byte(arg)}

	// InvokeChaincode calls the specified chaincode on the specified channel
	response := ctx.GetStub().InvokeChaincode(chaincodeName, chainCodeArgs, channelName)

	// Check if the response status is OK (200)
	if response.Status != 200 {
		return "", fmt.Errorf("failed to query other ledger. Message: %s", response.Message)
	}

	return string(response.Payload), nil
}

// RecordExists returns true when asset with given ID exists in world state
func (s *HistoryContract) RecordExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	recordJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return recordJSON != nil, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&HistoryContract{})
	if err != nil {
		fmt.Printf("Error creating history-verification chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting history-verification chaincode: %s", err.Error())
	}
}
