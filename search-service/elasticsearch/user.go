package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)



func (es *ElasticClient) IndexUser(data map[string]interface{}) error {

    id := fmt.Sprintf("%v", data["id"])
    doc, _ := json.Marshal(data)

    req, err := http.NewRequestWithContext(
        context.Background(),
        "PUT",
        fmt.Sprintf("%s/user/_doc/%s", es.BaseURL, id),
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
        return fmt.Errorf("failed to index user: %s", resp.Status)
    }

    log.Printf("Indexed user %s to Elasticsearch", id)
    return nil
}

func (es *ElasticClient) DeleteUser(id string) error {
	req, err := http.NewRequestWithContext(
		context.Background(),
		"DELETE",
		fmt.Sprintf("%s/user/_doc/%s", es.BaseURL, id),
		nil,
	)

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete user: %s", resp.Status)
	}

	log.Printf("Deleted user %s", id)
	return nil
}

func (es *ElasticClient) SearchUsers(query string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/user/_search?q=%s", es.BaseURL, query)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
