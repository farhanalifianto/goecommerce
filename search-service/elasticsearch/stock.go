package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (es *ElasticClient) UpdateProductStock(productID string, quantity interface{}) error {
	body := map[string]interface{}{
		"doc": map[string]interface{}{
			"stock": quantity,
		},
	}

	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		fmt.Sprintf("%s/products/_update/%s", es.BaseURL, productID),
		bytes.NewBuffer(b),
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
		return fmt.Errorf("failed to update stock: %s", resp.Status)
	}

	log.Printf("Updated stock for product %s", productID)
	return nil
}
