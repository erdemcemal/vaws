package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"vaws/internal/log"
	"vaws/internal/model"
)

// maxConcurrentDynamoDBCalls limits concurrent API calls to avoid throttling
const maxConcurrentDynamoDBCalls = 10

// ListTables lists all DynamoDB tables in the region with their details.
func (c *Client) ListTables(ctx context.Context) ([]model.Table, error) {
	log.Debug("Listing DynamoDB tables...")

	var tableNames []string
	paginator := dynamodb.NewListTablesPaginator(c.dynamodb, &dynamodb.ListTablesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		tableNames = append(tableNames, page.TableNames...)
	}

	if len(tableNames) == 0 {
		log.Info("No DynamoDB tables found")
		return nil, nil
	}

	log.Debug("Found %d tables, fetching details in parallel...", len(tableNames))

	// Fetch table details in parallel
	type tableResult struct {
		index int
		table *model.Table
		err   error
	}

	results := make(chan tableResult, len(tableNames))
	sem := make(chan struct{}, maxConcurrentDynamoDBCalls) // Limit concurrency

	var wg sync.WaitGroup
	for i, name := range tableNames {
		wg.Add(1)
		go func(idx int, tableName string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			table, err := c.DescribeTable(ctx, tableName)
			results <- tableResult{index: idx, table: table, err: err}
		}(i, name)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	tables := make([]model.Table, len(tableNames))
	validCount := 0

	for result := range results {
		if result.err != nil {
			log.Warn("Failed to describe table: %v", result.err)
			continue
		}
		tables[result.index] = *result.table
		validCount++
	}

	// Filter out empty entries (failed fetches)
	validTables := make([]model.Table, 0, validCount)
	for _, t := range tables {
		if t.Name != "" {
			validTables = append(validTables, t)
		}
	}

	// Sort by name
	sort.Slice(validTables, func(i, j int) bool {
		return validTables[i].Name < validTables[j].Name
	})

	log.Info("Loaded %d DynamoDB tables", len(validTables))
	return validTables, nil
}

// DescribeTable gets detailed information about a single DynamoDB table.
func (c *Client) DescribeTable(ctx context.Context, tableName string) (*model.Table, error) {
	output, err := c.dynamodb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}

	table := convertTable(output.Table)

	// Fetch TTL info separately
	ttlOutput, err := c.dynamodb.DescribeTimeToLive(ctx, &dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(tableName),
	})
	if err == nil && ttlOutput.TimeToLiveDescription != nil {
		if ttlOutput.TimeToLiveDescription.TimeToLiveStatus == dbtypes.TimeToLiveStatusEnabled {
			table.TTLEnabled = true
			table.TTLAttribute = aws.ToString(ttlOutput.TimeToLiveDescription.AttributeName)
		}
	}

	return table, nil
}

// convertTable converts an AWS DynamoDB TableDescription to our model.Table.
func convertTable(t *dbtypes.TableDescription) *model.Table {
	if t == nil {
		return nil
	}

	table := &model.Table{
		Name:      aws.ToString(t.TableName),
		ARN:       aws.ToString(t.TableArn),
		Status:    model.TableStatus(t.TableStatus),
		ItemCount: derefInt64(t.ItemCount),
		SizeBytes: derefInt64(t.TableSizeBytes),
	}

	if t.CreationDateTime != nil {
		table.CreatedAt = *t.CreationDateTime
	}

	// Key schema
	for _, k := range t.KeySchema {
		table.KeySchema = append(table.KeySchema, model.KeySchemaElement{
			AttributeName: aws.ToString(k.AttributeName),
			KeyType:       string(k.KeyType),
		})
	}

	// Billing mode
	if t.BillingModeSummary != nil {
		table.BillingMode = model.BillingMode(t.BillingModeSummary.BillingMode)
	} else if t.ProvisionedThroughput != nil {
		table.BillingMode = model.BillingModeProvisioned
	}

	// Provisioned throughput
	if t.ProvisionedThroughput != nil {
		table.ReadCapacityUnits = derefInt64(t.ProvisionedThroughput.ReadCapacityUnits)
		table.WriteCapacityUnits = derefInt64(t.ProvisionedThroughput.WriteCapacityUnits)
	}

	// Global Secondary Indexes
	for _, gsi := range t.GlobalSecondaryIndexes {
		var keySchema []model.KeySchemaElement
		for _, k := range gsi.KeySchema {
			keySchema = append(keySchema, model.KeySchemaElement{
				AttributeName: aws.ToString(k.AttributeName),
				KeyType:       string(k.KeyType),
			})
		}
		table.GlobalSecondaryIndexes = append(table.GlobalSecondaryIndexes, model.GlobalSecondaryIndex{
			IndexName:  aws.ToString(gsi.IndexName),
			KeySchema:  keySchema,
			Status:     string(gsi.IndexStatus),
			ItemCount:  derefInt64(gsi.ItemCount),
			SizeBytes:  derefInt64(gsi.IndexSizeBytes),
		})
	}

	// Local Secondary Indexes
	for _, lsi := range t.LocalSecondaryIndexes {
		var keySchema []model.KeySchemaElement
		for _, k := range lsi.KeySchema {
			keySchema = append(keySchema, model.KeySchemaElement{
				AttributeName: aws.ToString(k.AttributeName),
				KeyType:       string(k.KeyType),
			})
		}
		table.LocalSecondaryIndexes = append(table.LocalSecondaryIndexes, model.LocalSecondaryIndex{
			IndexName:  aws.ToString(lsi.IndexName),
			KeySchema:  keySchema,
			ItemCount:  derefInt64(lsi.ItemCount),
			SizeBytes:  derefInt64(lsi.IndexSizeBytes),
		})
	}

	// Streams
	if t.StreamSpecification != nil && t.StreamSpecification.StreamEnabled != nil {
		table.StreamEnabled = *t.StreamSpecification.StreamEnabled
		table.StreamViewType = string(t.StreamSpecification.StreamViewType)
	}

	// Deletion protection
	if t.DeletionProtectionEnabled != nil {
		table.DeletionProtection = *t.DeletionProtectionEnabled
	}

	return table
}

// derefInt64 safely dereferences an int64 pointer.
func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// QueryTable executes a query on a DynamoDB table.
func (c *Client) QueryTable(ctx context.Context, params model.QueryParams, lastKey map[string]interface{}) (*model.QueryResult, error) {
	log.Debug("Querying table %s with PK=%s", params.TableName, params.PartitionKeyVal)

	// Build key condition expression
	keyCondExpr := "#pk = :pkval"
	exprAttrNames := map[string]string{
		"#pk": params.PartitionKeyName,
	}
	exprAttrValues := map[string]dbtypes.AttributeValue{
		":pkval": &dbtypes.AttributeValueMemberS{Value: params.PartitionKeyVal},
	}

	// Add sort key condition if provided
	if params.SortKeyName != "" && params.SortKeyVal != "" {
		exprAttrNames["#sk"] = params.SortKeyName
		switch params.SortKeyCondition {
		case model.SortKeyConditionEquals, "":
			keyCondExpr += " AND #sk = :skval"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionLessThan:
			keyCondExpr += " AND #sk < :skval"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionLessEqual:
			keyCondExpr += " AND #sk <= :skval"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionGreater:
			keyCondExpr += " AND #sk > :skval"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionGreaterEq:
			keyCondExpr += " AND #sk >= :skval"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionBeginsWith:
			keyCondExpr += " AND begins_with(#sk, :skval)"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
		case model.SortKeyConditionBetween:
			keyCondExpr += " AND #sk BETWEEN :skval AND :skval2"
			exprAttrValues[":skval"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal}
			exprAttrValues[":skval2"] = &dbtypes.AttributeValueMemberS{Value: params.SortKeyVal2}
		}
	}

	// Add filter expression if provided
	if params.FilterExpression != "" && params.FilterAttrName != "" && params.FilterAttrValue != "" {
		exprAttrNames["#filterAttr"] = params.FilterAttrName
		exprAttrValues[":filterVal"] = &dbtypes.AttributeValueMemberS{Value: params.FilterAttrValue}
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(params.TableName),
		KeyConditionExpression:    aws.String(keyCondExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ScanIndexForward:          aws.Bool(params.ScanIndexForward),
		ReturnConsumedCapacity:    dbtypes.ReturnConsumedCapacityTotal,
	}

	// Add filter expression
	if params.FilterExpression != "" && params.FilterAttrName != "" && params.FilterAttrValue != "" {
		input.FilterExpression = aws.String(params.FilterExpression)
	}

	if params.Limit > 0 {
		input.Limit = aws.Int32(params.Limit)
	} else {
		input.Limit = aws.Int32(25) // Default limit
	}

	if params.IndexName != "" {
		input.IndexName = aws.String(params.IndexName)
	}

	// Set exclusive start key for pagination
	if lastKey != nil {
		input.ExclusiveStartKey = convertToAttributeValueMap(lastKey)
	}

	output, err := c.dynamodb.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return convertQueryOutput(output, params.PartitionKeyName, params.SortKeyName), nil
}

// ScanTable executes a scan on a DynamoDB table.
func (c *Client) ScanTable(ctx context.Context, params model.ScanParams, lastKey map[string]interface{}) (*model.QueryResult, error) {
	log.Debug("Scanning table %s", params.TableName)

	input := &dynamodb.ScanInput{
		TableName:              aws.String(params.TableName),
		ReturnConsumedCapacity: dbtypes.ReturnConsumedCapacityTotal,
	}

	if params.Limit > 0 {
		input.Limit = aws.Int32(params.Limit)
	} else {
		input.Limit = aws.Int32(25) // Default limit
	}

	if params.IndexName != "" {
		input.IndexName = aws.String(params.IndexName)
	}

	// Add filter expression if provided
	if params.FilterExpression != "" && params.FilterAttrName != "" && params.FilterAttrValue != "" {
		input.FilterExpression = aws.String(params.FilterExpression)
		input.ExpressionAttributeNames = map[string]string{
			"#filterAttr": params.FilterAttrName,
		}
		input.ExpressionAttributeValues = map[string]dbtypes.AttributeValue{
			":filterVal": &dbtypes.AttributeValueMemberS{Value: params.FilterAttrValue},
		}
	}

	// Set exclusive start key for pagination
	if lastKey != nil {
		input.ExclusiveStartKey = convertToAttributeValueMap(lastKey)
	}

	output, err := c.dynamodb.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return convertScanOutput(output, params.PartitionKeyName, params.SortKeyName), nil
}

// convertQueryOutput converts DynamoDB query output to our model.
func convertQueryOutput(output *dynamodb.QueryOutput, pkName, skName string) *model.QueryResult {
	result := &model.QueryResult{
		Count:        int(output.Count),
		ScannedCount: int(output.ScannedCount),
		HasMorePages: output.LastEvaluatedKey != nil,
	}

	if output.ConsumedCapacity != nil && output.ConsumedCapacity.CapacityUnits != nil {
		result.ConsumedCapacity = *output.ConsumedCapacity.CapacityUnits
	}

	if output.LastEvaluatedKey != nil {
		result.LastEvaluatedKey = convertFromAttributeValueMap(output.LastEvaluatedKey)
	}

	for _, item := range output.Items {
		result.Items = append(result.Items, convertItem(item, pkName, skName))
	}

	return result
}

// convertScanOutput converts DynamoDB scan output to our model.
func convertScanOutput(output *dynamodb.ScanOutput, pkName, skName string) *model.QueryResult {
	result := &model.QueryResult{
		Count:        int(output.Count),
		ScannedCount: int(output.ScannedCount),
		HasMorePages: output.LastEvaluatedKey != nil,
	}

	if output.ConsumedCapacity != nil && output.ConsumedCapacity.CapacityUnits != nil {
		result.ConsumedCapacity = *output.ConsumedCapacity.CapacityUnits
	}

	if output.LastEvaluatedKey != nil {
		result.LastEvaluatedKey = convertFromAttributeValueMap(output.LastEvaluatedKey)
	}

	for _, item := range output.Items {
		result.Items = append(result.Items, convertItem(item, pkName, skName))
	}

	return result
}

// convertItem converts a DynamoDB item to our model.
func convertItem(item map[string]dbtypes.AttributeValue, pkName, skName string) model.DynamoDBItem {
	raw := convertFromAttributeValueMap(item)

	// Generate JSON representation
	jsonBytes, _ := formatItemAsJSON(raw)
	jsonStr := string(jsonBytes)

	ddbItem := model.DynamoDBItem{
		Raw:  raw,
		JSON: jsonStr,
	}

	// Extract PK/SK values for display
	if pkName != "" {
		if val, ok := raw[pkName]; ok {
			ddbItem.PartitionKeyValue = fmt.Sprintf("%v", val)
		}
	}
	if skName != "" {
		if val, ok := raw[skName]; ok {
			ddbItem.SortKeyValue = fmt.Sprintf("%v", val)
		}
	}

	return ddbItem
}

// convertFromAttributeValueMap converts DynamoDB attribute values to Go native types.
func convertFromAttributeValueMap(attrs map[string]dbtypes.AttributeValue) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range attrs {
		result[k] = convertAttributeValue(v)
	}
	return result
}

// convertAttributeValue converts a single DynamoDB attribute value to a Go native type.
func convertAttributeValue(av dbtypes.AttributeValue) interface{} {
	switch v := av.(type) {
	case *dbtypes.AttributeValueMemberS:
		return v.Value
	case *dbtypes.AttributeValueMemberN:
		return v.Value // Keep as string to preserve precision
	case *dbtypes.AttributeValueMemberB:
		return fmt.Sprintf("[binary %d bytes]", len(v.Value))
	case *dbtypes.AttributeValueMemberBOOL:
		return v.Value
	case *dbtypes.AttributeValueMemberNULL:
		return nil
	case *dbtypes.AttributeValueMemberSS:
		return v.Value
	case *dbtypes.AttributeValueMemberNS:
		return v.Value
	case *dbtypes.AttributeValueMemberBS:
		return fmt.Sprintf("[binary set %d items]", len(v.Value))
	case *dbtypes.AttributeValueMemberL:
		list := make([]interface{}, len(v.Value))
		for i, item := range v.Value {
			list[i] = convertAttributeValue(item)
		}
		return list
	case *dbtypes.AttributeValueMemberM:
		return convertFromAttributeValueMap(v.Value)
	default:
		return fmt.Sprintf("%v", av)
	}
}

// convertToAttributeValueMap converts Go native types back to DynamoDB attribute values.
func convertToAttributeValueMap(data map[string]interface{}) map[string]dbtypes.AttributeValue {
	result := make(map[string]dbtypes.AttributeValue)
	for k, v := range data {
		result[k] = convertToAttributeValue(v)
	}
	return result
}

// convertToAttributeValue converts a single Go native type to a DynamoDB attribute value.
func convertToAttributeValue(v interface{}) dbtypes.AttributeValue {
	switch val := v.(type) {
	case string:
		return &dbtypes.AttributeValueMemberS{Value: val}
	case bool:
		return &dbtypes.AttributeValueMemberBOOL{Value: val}
	case nil:
		return &dbtypes.AttributeValueMemberNULL{Value: true}
	case []interface{}:
		list := make([]dbtypes.AttributeValue, len(val))
		for i, item := range val {
			list[i] = convertToAttributeValue(item)
		}
		return &dbtypes.AttributeValueMemberL{Value: list}
	case map[string]interface{}:
		return &dbtypes.AttributeValueMemberM{Value: convertToAttributeValueMap(val)}
	default:
		// For numbers and other types, use string representation
		return &dbtypes.AttributeValueMemberS{Value: fmt.Sprintf("%v", val)}
	}
}

// formatItemAsJSON formats an item as indented JSON.
func formatItemAsJSON(item map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(item, "", "  ")
}
