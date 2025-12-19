package idempotency

import (
	"context"
	"errors"
	"sync"

	dyn "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// simpleMock is a very small in-memory mock for PutItem/GetItem/UpdateItem used in unit tests.
// NOTE: This is intentionally minimal and not production-grade.
type simpleMock struct {
	mu    sync.Mutex
	table map[string]map[string]types.AttributeValue
	putCalls int
	getCalls int
	updateCalls int
	transactCalls int
}

func newSimpleMock() *simpleMock {
	return &simpleMock{
		table: map[string]map[string]types.AttributeValue{},
	}
}

func (m *simpleMock) PutItem(ctx context.Context, params *dyn.PutItemInput, optFns ...func(*dyn.Options)) (*dyn.PutItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putCalls++
	// implement ConditionExpression: attribute_not_exists(idempotency_key)
	if params.ConditionExpression != nil && *params.ConditionExpression == "attribute_not_exists(idempotency_key)" {
		keyAttr := params.Item["idempotency_key"]
		if keyAttr == nil {
			return nil, errors.New("missing key")
		}
		k := keyAttr.(*types.AttributeValueMemberS).Value
		if _, ok := m.table[k]; ok {
			// simulate conditional failure
			return nil, &types.ConditionalCheckFailedException{}
		}
		m.table[k] = params.Item
		return &dyn.PutItemOutput{}, nil
	}
	// otherwise simple put (overwrite)
	if params.Item == nil {
		return nil, errors.New("nil item")
	}
	k := params.Item["idempotency_key"].(*types.AttributeValueMemberS).Value
	m.table[k] = params.Item
	return &dyn.PutItemOutput{}, nil
}

func (m *simpleMock) GetItem(ctx context.Context, params *dyn.GetItemInput, optFns ...func(*dyn.Options)) (*dyn.GetItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	keyAttr := params.Key["idempotency_key"]
	if keyAttr == nil {
		return nil, errors.New("missing key")
	}
	k := keyAttr.(*types.AttributeValueMemberS).Value
	item, ok := m.table[k]
	if !ok {
		return &dyn.GetItemOutput{}, nil
	}
	return &dyn.GetItemOutput{Item: item}, nil
}

func (m *simpleMock) UpdateItem(ctx context.Context, params *dyn.UpdateItemInput, optFns ...func(*dyn.Options)) (*dyn.UpdateItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	keyAttr := params.Key["idempotency_key"]
	if keyAttr == nil {
		return nil, errors.New("missing key")
	}
	k := keyAttr.(*types.AttributeValueMemberS).Value
	item, ok := m.table[k]
	if !ok {
		return nil, errors.New("item not found")
	}
	// very naive update: support SET #s = :done, response_body = :rb, response_status = :rs, updated_at = :ua
	// extract values from ExpressionAttributeValues if present
	if v, ok := params.ExpressionAttributeValues[":rb"]; ok {
		item["response_body"] = v
	}
	if v, ok := params.ExpressionAttributeValues[":rs"]; ok {
		item["response_status"] = v
	}
	if v, ok := params.ExpressionAttributeValues[":ua"]; ok {
		item["updated_at"] = v
	}
	if v, ok := params.ExpressionAttributeValues[":done"]; ok {
		item["status"] = v
	}
	if v, ok := params.ExpressionAttributeValues[":failed"]; ok {
		item["status"] = v
	}
	m.table[k] = item
	return &dyn.UpdateItemOutput{Attributes: item}, nil
}

func (m *simpleMock) TransactWriteItems(ctx context.Context, params *dyn.TransactWriteItemsInput, optFns ...func(*dyn.Options)) (*dyn.TransactWriteItemsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactCalls++
	// minimal implementation: if any Put has ConditionExpression attribute_not_exists(idempotency_key) and key exists -> conditional fail
	for _, it := range params.TransactItems {
		if p := it.Put; p != nil {
			if p.ConditionExpression != nil && *p.ConditionExpression == "attribute_not_exists(idempotency_key)" {
				kattr := p.Item["idempotency_key"]
				if kattr == nil {
					return nil, errors.New("missing key in transact put")
				}
				k := kattr.(*types.AttributeValueMemberS).Value
				if _, ok := m.table[k]; ok {
					// simulate conditional failure
					return nil, &types.TransactionCanceledException{}
				}
			}
		}
	}
	// if no conflicts, perform puts where TableName matches our internal table optionally
	for _, it := range params.TransactItems {
		if p := it.Put; p != nil {
			// assume it's for same table (since mock doesn't have multiple tables)
			kattr := p.Item["idempotency_key"]
			if kattr != nil {
				k := kattr.(*types.AttributeValueMemberS).Value
				m.table[k] = p.Item
			}
		}
	}
	return &dyn.TransactWriteItemsOutput{}, nil
}
