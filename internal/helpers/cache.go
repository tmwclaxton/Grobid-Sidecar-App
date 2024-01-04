package helpers

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// CacheHelper represents a cache with DynamoDB as the backend.
type CacheHelper struct {
	tableName string
	svc       *dynamodb.DynamoDB
	mu        sync.Mutex
}

// NewCacheHelper creates a new CacheHelper instance with DynamoDB as the backend.
func NewCacheHelper(sess *session.Session, tableName string) (*CacheHelper, error) {
	svc := dynamodb.New(sess)
	return &CacheHelper{
		tableName: tableName,
		svc:       svc,
	}, nil
}

// AddOrIncrCache adds a key to the cache or increments its value if it already exists.
func (c *CacheHelper) AddOrIncrCache(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the item already exists in DynamoDB
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {S: aws.String(key)},
		},
		UpdateExpression: aws.String("SET #cacheValue = if_not_exists(#cacheValue, :start) + :inc"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":start": {N: aws.String("0")},
			":inc":   {N: aws.String("1")},
		},
		ExpressionAttributeNames: map[string]*string{
			"#cacheValue": aws.String("value"),
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	}

	_, err := c.svc.UpdateItem(input)
	if err != nil {
		return err
	}

	return nil
}

// DecrOrDeleteCache decreases the value associated with a key or deletes the key if the value becomes zero.
func (c *CacheHelper) DecrOrDeleteCache(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the item exists in DynamoDB
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {S: aws.String(key)},
		},
	}

	result, err := c.svc.GetItem(getInput)
	if err != nil {
		return err
	}

	// If the item exists, decrease the value or delete the key if the value becomes zero
	if len(result.Item) > 0 {
		// Use the correct primary key attribute name
		primaryKeyValue := result.Item["key"].S

		val := result.Item["value"].N
		if val != nil {
			if *val > "1" {
				// Decrease the value
				updateInput := &dynamodb.UpdateItemInput{
					TableName: aws.String(c.tableName),
					Key: map[string]*dynamodb.AttributeValue{
						"key": {S: primaryKeyValue},
					},
					UpdateExpression: aws.String("SET #cacheValue = #cacheValue - :dec"),
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":dec": {N: aws.String("1")},
					},
					ExpressionAttributeNames: map[string]*string{
						"#cacheValue": aws.String("value"),
					},
					ReturnValues: aws.String("UPDATED_NEW"),
				}

				_, err := c.svc.UpdateItem(updateInput)
				if err != nil {
					return err
				}
			} else {
				// Delete the key if the value becomes zero
				deleteInput := &dynamodb.DeleteItemInput{
					TableName: aws.String(c.tableName),
					Key: map[string]*dynamodb.AttributeValue{
						"key": {S: primaryKeyValue},
					},
				}

				_, err := c.svc.DeleteItem(deleteInput)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetCacheValue returns the value associated with a key.
func (c *CacheHelper) GetCacheValue(key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the item exists in DynamoDB
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {S: aws.String(key)},
		},
	}

	result, err := c.svc.GetItem(getInput)
	if err != nil {
		return "", err
	}

	// If the item exists, return the value
	if len(result.Item) > 0 {
		val := result.Item["value"].N
		if val != nil {
			return *val, nil
		}
	}

	return "", nil
}
