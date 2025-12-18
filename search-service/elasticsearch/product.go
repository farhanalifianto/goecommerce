package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (es *ElasticClient) IndexProduct(product map[string]interface{}) error {
	index := "products"

	idValue, ok := product["id"]
	if !ok {
		return fmt.Errorf("missing id field in product")
	}
	id := fmt.Sprintf("%v", idValue)

	doc, _ := json.Marshal(product)
	req, err := http.NewRequestWithContext(
		context.Background(),
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
		return fmt.Errorf("failed to index product: %s", resp.Status)
	}

	log.Printf("Indexed product %s", id)
	return nil
}

func (es *ElasticClient) DeleteProduct(id string) error {
	url := fmt.Sprintf("%s/products/_doc/%s", es.BaseURL, id)
	req, _ := http.NewRequest("DELETE", url, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete product: %s", resp.Status)
	}

	log.Printf("Deleted product %s", id)
	return nil
}
func (es *ElasticClient) SearchProducts(
	query string,
	categoryID string,
	minPrice string,
	maxPrice string,
) ([]map[string]interface{}, error) {

	boolQuery := map[string]interface{}{
		"must": []interface{}{
			map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  query,
					"fields": []string{"name", "desc"},
				},
			},
		},
		"filter": []interface{}{},
	}

	// filter category
	if categoryID != "" {
		boolQuery["filter"] = append(
			boolQuery["filter"].([]interface{}),
			map[string]interface{}{
				"term": map[string]interface{}{
					"category_id": categoryID,
				},
			},
		)
	}

	// filter price range
	priceRange := map[string]interface{}{}
	if minPrice != "" {
		priceRange["gte"] = minPrice
	}
	if maxPrice != "" {
		priceRange["lte"] = maxPrice
	}
	if len(priceRange) > 0 {
		boolQuery["filter"] = append(
			boolQuery["filter"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"price": priceRange,
				},
			},
		)
	}

	searchBody := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}

	bodyBytes, _ := json.Marshal(searchBody)
	url := fmt.Sprintf("%s/products/_search", es.BaseURL)

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
	items := result["hits"].(map[string]interface{})["hits"].([]interface{})

	for _, item := range items {
		doc := item.(map[string]interface{})["_source"].(map[string]interface{})
		hits = append(hits, doc)
	}

	return hits, nil
}