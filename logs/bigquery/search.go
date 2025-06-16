package bigquery

import (
	"fmt"
	"sync"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
)

// Searcher implements log searching using BigQuery.
type Searcher struct {
	conn          connection.GCPConnection
	mappingConfig *logs.FieldMappingConfig
	bqClient      *bigquery.Client
	clientMutex   sync.Mutex
}

// New creates a new BigQuery log searcher.
func New(conn connection.GCPConnection, mappingConfig *logs.FieldMappingConfig) *Searcher {
	return &Searcher{
		conn:          conn,
		mappingConfig: mappingConfig,
	}
}

// Search executes a log search query using BigQuery.
func (s *Searcher) Search(ctx context.Context, request Request) (*logs.LogResult, error) {
	if err := s.initializeClients(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize clients: %w", err)
	}

	sqlQuery := request.Query
	if sqlQuery == "" {
		return nil, fmt.Errorf("query is required")
	}

	response, err := s.executeQuery(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	mappingConfig := logs.FieldMappingConfig{}
	if s.mappingConfig != nil {
		mappingConfig = *s.mappingConfig
	}

	result, err := response.ToLogResult(mappingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response to log result: %w", err)
	}

	return &result, nil
}

// initializeClients creates and caches BigQuery client.
func (s *Searcher) initializeClients(ctx context.Context) error {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	if s.bqClient != nil {
		return nil
	}

	if err := s.conn.HydrateConnection(ctx); err != nil {
		return fmt.Errorf("failed to hydrate connection: %w", err)
	}

	client, err := bigquery.NewClient(ctx, s.conn.Project)
	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	s.bqClient = client

	return nil
}

// executeQuery runs the BigQuery SQL query and returns results.
func (s *Searcher) executeQuery(ctx context.Context, sqlQuery string) (*Response, error) {
	s.clientMutex.Lock()
	client := s.bqClient
	s.clientMutex.Unlock()

	query := client.Query(sqlQuery)

	query.QueryConfig.DryRun = false
	query.QueryConfig.UseLegacySQL = false

	job, err := query.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	if err := status.Err(); err != nil {
		return nil, fmt.Errorf("query job failed: %w", err)
	}

	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read query results: %w", err)
	}

	var rows []Row
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read result row: %w", err)
		}
		rows = append(rows, row)
	}

	jobStats := s.extractJobStats(job)
	return &Response{
		Rows:          rows,
		Schema:        it.Schema,
		TotalRows:     int64(it.TotalRows),
		NextPageToken: it.PageInfo().Token,
		JobStats:      jobStats,
	}, nil
}

// extractJobStats extracts relevant statistics from a BigQuery job.
func (s *Searcher) extractJobStats(job *bigquery.Job) *JobStats {
	stats := job.LastStatus().Statistics
	if stats == nil {
		return nil
	}

	jobStats := &JobStats{
		CreationTime: stats.CreationTime,
		StartTime:    stats.StartTime,
		EndTime:      stats.EndTime,
	}

	// Extract query-specific statistics if available
	if queryStats, ok := stats.Details.(*bigquery.QueryStatistics); ok {
		jobStats.BytesProcessed = queryStats.TotalBytesProcessed
		jobStats.BytesBilled = queryStats.TotalBytesBilled
		jobStats.TotalSlotMs = queryStats.SlotMillis
	}

	return jobStats
}

// Close closes the clients and releases resources.
func (s *Searcher) Close() error {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	if s.bqClient != nil {
		err := s.bqClient.Close()
		s.bqClient = nil
		return err
	}
	return nil
}
