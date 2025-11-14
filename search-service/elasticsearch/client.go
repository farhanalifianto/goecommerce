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
	idValue, ok := address["id"]
	if !ok {
		return fmt.Errorf("❌ missing id field in address")
	}

	id := fmt.Sprintf("%v", idValue)


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

func (es *ElasticClient) DeleteAddress(id string) error {
    url := fmt.Sprintf("%s/address/_doc/%s", es.BaseURL, id)
    req, _ := http.NewRequest("DELETE", url, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        return fmt.Errorf("failed to delete document: %s", resp.Status)
    }

    log.Printf("🗑️ Deleted address %s from Elasticsearch", id)
    return nil
}


func (es *ElasticClient) SearchAddresses(query string) ([]map[string]interface{}, error) {
	searchBody := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"name", "desc"},
			},
		},
	}

	bodyBytes, _ := json.Marshal(searchBody)
	url := fmt.Sprintf("%s/address/_search", es.BaseURL)

	req, err := http.NewRequest("GET", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	hits := []map[string]interface{}{}
	h := result["hits"].(map[string]interface{})["hits"].([]interface{})

	for _, item := range h {
		doc := item.(map[string]interface{})["_source"].(map[string]interface{})
		hits = append(hits, doc)
	}

	return hits, nil
}


