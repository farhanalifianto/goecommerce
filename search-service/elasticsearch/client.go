package elasticsearch

type ElasticClient struct {
	BaseURL string
}

func NewElasticClient(baseURL string) *ElasticClient {
	return &ElasticClient{BaseURL: baseURL}
}
