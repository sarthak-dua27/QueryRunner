package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Query struct {
	Query map[string]interface{} `json:"query"`
}

type ResultOutput struct {
	Query   QueryResult `json:"query_result"`
	Success bool        `json:"success"`
}

type SearchHit struct {
	Index  string          `json:"index"`
	ID     string          `json:"id"`
	Score  float64         `json:"score"`
	Fields json.RawMessage `json:"fields,omitempty"`
}

type SearchResult struct {
	Status   interface{} `json:"status"`
	Total    int         `json:"total_hits"`
	Hits     []SearchHit `json:"hits"`
	Took     int64       `json:"took"`
	MaxScore float64     `json:"max_score"`
}

type BatchSearcher struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

func NewBatchSearcher(host string, username, password string) *BatchSearcher {
	return &BatchSearcher{
		baseURL:  host,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func createSearchPayload(query string) ([]byte, error) {
	return []byte(query), nil
}

func (bs *BatchSearcher) performSearch(ctx context.Context, indexName, query string) (*SearchResult, error) {
	url := fmt.Sprintf("%s/api/index/%s/query", bs.baseURL, indexName)

	payload, err := createSearchPayload(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(bs.username + ":" + bs.password))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	resp, err := bs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &result, nil
}

type QueryResult struct {
	QueryIndex int
	Result     *SearchResult
	Error      error
}

func (bs *BatchSearcher) RunBatchSearch(ctx context.Context, indexName string, queries []string, batchSize int) (int64, int64, []QueryResult) {
	var (
		successCount int64
		failureCount int64
		rateLimiter  = make(chan struct{}, batchSize)
		results      = make([]QueryResult, len(queries))
		wg           sync.WaitGroup
	)

	for i, query := range queries {
		wg.Add(1)
		rateLimiter <- struct{}{}

		go func(queryIndex int, searchQuery string) {
			defer wg.Done()
			defer func() { <-rateLimiter }()

			result, err := bs.performSearch(ctx, indexName, searchQuery)
			if err != nil {
				atomic.AddInt64(&failureCount, 1)
				results[queryIndex] = QueryResult{
					QueryIndex: queryIndex,
					Error:      err,
				}
				log.Printf("Query %d failed: %v", queryIndex, err)
			} else {
				atomic.AddInt64(&successCount, 1)
				results[queryIndex] = QueryResult{
					QueryIndex: queryIndex,
					Result:     result,
				}
			}
		}(i, query)
	}

	wg.Wait()

	return successCount, failureCount, results
}

func main() {
	host := flag.String("host", "", "Couchbase FTS endpoint")
	username := flag.String("user", "username", "Username")
	password := flag.String("pass", "password", "Password")
	index := flag.String("index", "indexname", "FTS index name")
	concurrency := flag.Int("concurrency", 20, "Number of concurrent requests")
	iterations := flag.Int("iterations", 1, "Number of times to run each query")
	numQueries := flag.Int("numqueries", 300, "Must be multiple of 3")
	printResults := flag.Bool("print-results", true, "Print search results")
	flag.Parse()

	queriesFile := "queries.json"
	var queries []Query

	if _, err := os.Stat(queriesFile); os.IsNotExist(err) {
		fmt.Println("queries.json not found, generating it...")
		GenerateQueries(*numQueries)
	}

	data, err := ioutil.ReadFile(queriesFile)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", queriesFile, err)
		return
	}

	if err := json.Unmarshal(data, &queries); err != nil {
		fmt.Printf("Failed to parse JSON from %s: %v\n", queriesFile, err)
		return
	}

	allQueries := make([]string, 0, len(queries)*(*iterations))
	for i := 0; i < *iterations; i++ {
		for _, query := range queries {
			queryJSON, err := json.Marshal(query)
			if err != nil {
				log.Printf("Failed to serialize query: %v", err)
				continue
			}
			allQueries = append(allQueries, string(queryJSON))
		}
	}

	ctx := context.Background()
	searcher := NewBatchSearcher(*host, *username, *password)
	successCount, failureCount, results := searcher.RunBatchSearch(ctx, *index, allQueries, *concurrency)

	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failureCount)

	if *printResults {
		resultsFile := "results.json"
		file, err := os.Create(resultsFile)
		if err != nil {
			log.Fatalf("Failed to create results file: %v\n", err)
		}
		defer file.Close()
	
		// Structure to hold all results for writing to file
		type ResultOutput struct {
			Query   QueryResult `json:"query_result"`
			Success bool        `json:"success"`
		}
	
		var output []ResultOutput
	
		for _, result := range results {
			if result.Error != nil {
				output = append(output, ResultOutput{
					Query:   result,
					Success: false,
				})
			} else {
				output = append(output, ResultOutput{
					Query:   result,
					Success: true,
				})
			}
		}
	
		// Write the results to the file in JSON format
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Failed to serialize results: %v\n", err)
		}
	
		if _, err := file.Write(data); err != nil {
			log.Fatalf("Failed to write to results file: %v\n", err)
		}
	
		fmt.Printf("Results written to %s\n", resultsFile)
	}
	
	
}
