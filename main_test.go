package main

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Mocks ---

// MockTransactionContext mocks the transaction context
type MockTransactionContext struct {
	contractapi.TransactionContextInterface
	mock.Mock
}

func (m *MockTransactionContext) GetStub() shim.ChaincodeStubInterface {
	args := m.Called()
	return args.Get(0).(shim.ChaincodeStubInterface)
}

func (m *MockTransactionContext) GetClientIdentity() cid.ClientIdentity {
	args := m.Called()
	return args.Get(0).(cid.ClientIdentity)
}

// MockChaincodeStub mocks the chaincode stub (ledger interaction)
type MockChaincodeStub struct {
	shim.ChaincodeStubInterface
	mock.Mock
}

func (m *MockChaincodeStub) GetState(key string) ([]byte, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockChaincodeStub) PutState(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockChaincodeStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	args := m.Called(key)
	return args.Get(0).(shim.HistoryQueryIteratorInterface), args.Error(1)
}

func (m *MockChaincodeStub) GetTxTimestamp() (*timestamppb.Timestamp, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*timestamppb.Timestamp), args.Error(1)
}

// MockClientIdentity mocks the client identity (MSP ID)
type MockClientIdentity struct {
	cid.ClientIdentity
	mock.Mock
}

func (m *MockClientIdentity) GetMSPID() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// MockHistoryQueryIterator mocks the iterator for history results
type MockHistoryQueryIterator struct {
	shim.HistoryQueryIteratorInterface
	mock.Mock
}

func (m *MockHistoryQueryIterator) HasNext() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockHistoryQueryIterator) Next() (*queryresult.KeyModification, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queryresult.KeyModification), args.Error(1)
}

func (m *MockHistoryQueryIterator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// --- Tests ---

func TestCreateRecord(t *testing.T) {
	t.Log("Starting TestCreateRecord: Verifying creation of a new record")
	ctx := new(MockTransactionContext)
	stub := new(MockChaincodeStub)
	clientIdentity := new(MockClientIdentity)

	ctx.On("GetStub").Return(stub)
	ctx.On("GetClientIdentity").Return(clientIdentity)

	t.Log("Setting expectations: Checking if record exists and putting new state")
	// Expectation: Record does not exist yet
	stub.On("GetState", "REC001").Return(nil, nil)
	clientIdentity.On("GetMSPID").Return("Org1MSP", nil)
	stub.On("GetTxTimestamp").Return(timestamppb.Now(), nil)
	stub.On("PutState", "REC001", mock.Anything).Return(nil)

	t.Log("Invoking CreateRecord smart contract function...")
	contract := new(HistoryContract)
	err := contract.CreateRecord(ctx, "REC001", "Initial Draft", "CREATED")

	assert.NoError(t, err)
	t.Log("CreateRecord returned no error")
	stub.AssertExpectations(t)
}

func TestGetRecordHistory(t *testing.T) {
	t.Log("Starting TestGetRecordHistory: Verifying history retrieval")
	ctx := new(MockTransactionContext)
	stub := new(MockChaincodeStub)
	iterator := new(MockHistoryQueryIterator)

	ctx.On("GetStub").Return(stub)

	// Prepare mock history data
	t.Log("Preparing mock history data...")
	record := VerificationRecord{ID: "REC001", Description: "Draft", Status: "CREATED"}
	recordBytes, _ := json.Marshal(record)
	modification := &queryresult.KeyModification{
		TxId:      "tx123",
		Value:     recordBytes,
		Timestamp: timestamppb.Now(),
		IsDelete:  false,
	}

	stub.On("GetHistoryForKey", "REC001").Return(iterator, nil)
	iterator.On("HasNext").Return(true).Once()
	iterator.On("Next").Return(modification, nil).Once()
	iterator.On("HasNext").Return(false).Once()
	iterator.On("Close").Return(nil)

	t.Log("Invoking GetRecordHistory smart contract function...")
	contract := new(HistoryContract)
	history, err := contract.GetRecordHistory(ctx, "REC001")

	assert.NoError(t, err)
	t.Logf("Retrieved %d history record(s)", len(history))
	assert.Len(t, history, 1)
	assert.Equal(t, "tx123", history[0].TxId)
	assert.Equal(t, "Draft", history[0].Record.Description)
	t.Log("Verification of history record content passed")
}

func TestCreateRecordOrg3(t *testing.T) {
	t.Log("Starting TestCreateRecordOrg3: Verifying creation for Org3")
	ctx := new(MockTransactionContext)
	stub := new(MockChaincodeStub)
	clientIdentity := new(MockClientIdentity)

	ctx.On("GetStub").Return(stub)
	ctx.On("GetClientIdentity").Return(clientIdentity)

	t.Log("Setting expectations: Checking if record exists and putting new state for Org3")
	// Expectation: Record does not exist yet
	stub.On("GetState", "REC003").Return(nil, nil)
	clientIdentity.On("GetMSPID").Return("Org3MSP", nil)
	stub.On("GetTxTimestamp").Return(timestamppb.Now(), nil)
	stub.On("PutState", "REC003", mock.Anything).Return(nil)

	t.Log("Invoking CreateRecord smart contract function as Org3...")
	contract := new(HistoryContract)
	err := contract.CreateRecord(ctx, "REC003", "Org3 Draft", "CREATED")

	assert.NoError(t, err)
	t.Log("CreateRecord returned no error for Org3")
	stub.AssertExpectations(t)
}

func TestCreateRecordOrg4(t *testing.T) {
	t.Log("Starting TestCreateRecordOrg4: Verifying creation for Org4")
	ctx := new(MockTransactionContext)
	stub := new(MockChaincodeStub)
	clientIdentity := new(MockClientIdentity)

	ctx.On("GetStub").Return(stub)
	ctx.On("GetClientIdentity").Return(clientIdentity)

	t.Log("Setting expectations: Checking if record exists and putting new state for Org4")
	// Expectation: Record does not exist yet
	stub.On("GetState", "REC004").Return(nil, nil)
	clientIdentity.On("GetMSPID").Return("Org4MSP", nil)
	stub.On("GetTxTimestamp").Return(timestamppb.Now(), nil)
	stub.On("PutState", "REC004", mock.Anything).Return(nil)

	t.Log("Invoking CreateRecord smart contract function as Org4...")
	contract := new(HistoryContract)
	err := contract.CreateRecord(ctx, "REC004", "Org4 Draft", "CREATED")

	assert.NoError(t, err)
	t.Log("CreateRecord returned no error for Org4")
	stub.AssertExpectations(t)
}
