// Package aws provides AWS SDK client management and operations.
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"vaws/internal/log"
	"vaws/internal/model"
)

// FetchLogs retrieves logs from CloudWatch for a specific log stream.
// startTime is milliseconds since epoch for incremental fetching.
// Returns log entries, the next startTime to use, and any error.
func (c *Client) FetchLogs(ctx context.Context, logGroup, logStream string, startTime int64, limit int32) ([]model.CloudWatchLogEntry, int64, error) {
	log.Debug("Fetching CloudWatch logs: group=%s, stream=%s, startTime=%d", logGroup, logStream, startTime)

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:   aws.String(logGroup),
		LogStreamNames: []string{logStream},
		Limit:          aws.Int32(limit),
	}

	if startTime > 0 {
		input.StartTime = aws.Int64(startTime)
	}

	var entries []model.CloudWatchLogEntry
	var lastTimestamp int64

	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(c.cwlogs, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, startTime, fmt.Errorf("failed to fetch logs: %w", err)
		}

		for _, event := range page.Events {
			entry := model.CloudWatchLogEntry{
				Timestamp:     time.UnixMilli(aws.ToInt64(event.Timestamp)),
				Message:       aws.ToString(event.Message),
				IngestionTime: time.UnixMilli(aws.ToInt64(event.IngestionTime)),
				LogStreamName: logStream,
			}
			entries = append(entries, entry)

			if aws.ToInt64(event.Timestamp) > lastTimestamp {
				lastTimestamp = aws.ToInt64(event.Timestamp)
			}
		}
	}

	if len(entries) > 0 {
		log.Debug("Fetched %d log entries from CloudWatch", len(entries))
	}

	// Return lastTimestamp + 1 to avoid duplicate on next fetch
	nextStartTime := startTime
	if lastTimestamp > 0 {
		nextStartTime = lastTimestamp + 1
	}

	return entries, nextStartTime, nil
}

// BuildLogStreamName constructs the log stream name from components.
// Format: {prefix}/{container-name}/{task-id}
func BuildLogStreamName(prefix, containerName, taskID string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, containerName, taskID)
}
