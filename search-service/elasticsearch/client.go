package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ElasticClient struct {
	BaseURL string
}

func NewElasticClient(baseURL string) *ElasticClient {
	return &ElasticClient{BaseURL: baseURL}
}

func (es *ElasticClient) IndexAddress(address map[string]interface{}) error {
	index := "address"
	id := fmt.Sprintf("%v", address["id"])

	doc, _ := json.Marshal(address)
	req, err := http.NewRequestWithContext(context.Background(),
		"PUT",
		fmt.Sprintf("%s/%s/_doc/%s", es.BaseURL, index, id),
		bytes.NewBuffer(doc),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to index document: %s", resp.Status)
	}

	log.Printf("✅ Indexed address %s to Elasticsearch", id)
	return nil
}

func (es *ElasticClient) SearchAddresses(query string) ([]map[string]interface{}, error) {
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"name", "desc"},
			},
		},
	}

	body, _ := json.Marshal(searchQuery)
	url := fmt.Sprintf("%s/address/_search", es.BaseURL) // FIXED

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search failed: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	hits := []map[string]interface{}{}
	if h, ok := result["hits"].(map[string]interface{}); ok {
		if inner, ok := h["hits"].([]interface{}); ok {
			for _, item := range inner {
				src := item.(map[string]interface{})["_source"].(map[string]interface{})
				hits = append(hits, src)
			}
		}
	}

	return hits, nil
}
