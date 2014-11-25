package elastic

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/dahernan/gopherscraper/model"
)

type ItemElastic struct {
	handler          *ModelHandler
	index            string
	funcEndpoint     EndpointFunc
	funcEndpointItem EndpointFuncWithItem
}

func NewItemElastic(index string) *ItemElastic {
	return &ItemElastic{
		index:            index,
		handler:          Handler(),
		funcEndpoint:     ItemEndpoint,
		funcEndpointItem: ItemEndpointWithItem,
	}
}

type EndpointFunc func(string, string, string) (string, error)
type EndpointFuncWithItem func(string, *model.Item) (string, error)

func ItemEndpoint(index string, indexType string, itemId string) (string, error) {
	if index == "" || indexType == "" || itemId == "" {
		return "", NewModelError("index or indexType or itemId are empty building ItemEndpoint", http.StatusBadRequest, nil)
	}

	// http://elasticsearch:9200/:index/:type/:id
	// http://elasticsearch:9200/my_scrapping/www.amazon.com/a33z83d388

	return path.Clean(fmt.Sprintf("/%s/%s/%s", index, indexType, itemId)), nil

}
func ItemEndpointWithItem(index string, item *model.Item) (string, error) {
	u, err := url.Parse(item.ScrapUrl)
	if err != nil {
		return "", err
	}

	return ItemEndpoint(index, u.Host, item.Id)
}

func (ie *ItemElastic) send(method string, item *model.Item) (ElasticResponse, error) {
	var jsonResponse ElasticResponse

	endpoint, err := ie.funcEndpointItem(ie.index, item)

	if err != nil {
		return ElasticResponse{}, err
	}
	jsonResponse, err = ie.handler.Send(method, endpoint, item)
	return jsonResponse, err
}

func (ie *ItemElastic) Get(indexType string, itemId string) (*model.Item, error) {
	var item model.Item

	endpoint, err := ie.funcEndpoint(ie.index, indexType, itemId)
	if err != nil {
		return nil, err
	}

	es, err := ie.handler.Get(endpoint)
	if err != nil {
		return nil, err
	}

	err = NewModelFromRaw(es.Source, &item)
	if err != nil {
		return nil, err
	}

	// populate Id and Version from ElasticSearch
	item.Id = es.Id
	item.Version = es.Version

	return &item, nil
}

func (ie *ItemElastic) MultiGet(indexType string, itemIds []string) ([]*model.Item, error) {
	endpoint, err := ie.funcEndpoint(ie.index, indexType, "_mget")
	result := make([]*model.Item, len(itemIds))

	if err != nil {
		return nil, err
	}

	mget := map[string][]string{
		"ids": itemIds,
	}

	response := struct {
		Docs []ElasticModel `json:"docs"`
	}{}

	err = ie.handler.SendRAW("GET", endpoint, &mget, &response)
	if err != nil {
		return nil, err
	}

	eitem := response.Docs
	for i, _ := range eitem {
		if eitem[i].Found {
			var item model.Item
			err = NewModelFromRaw(eitem[i].Source, &item)
			item.Id = eitem[i].Id
			item.Version = eitem[i].Version
			result[i] = &item
		}
	}

	return result, nil

}

func (ie *ItemElastic) Search(indexType string, query interface{}) (interface{}, error) {
	endpoint, err := ie.funcEndpoint(ie.index, indexType, "_search")

	if err != nil {
		return nil, err
	}

	var response interface{}
	err = ie.handler.SendRAW("GET", endpoint, &query, &response)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func (ie *ItemElastic) Head(indexType string, itemId string) (bool, error) {
	endpoint, err := ie.funcEndpoint(ie.index, indexType, itemId)
	if err != nil {
		return false, err
	}

	_, err = ie.handler.Send("HEAD", endpoint, nil)
	if err != nil {
		return false, err
	}

	return true, nil

}

func (ie *ItemElastic) Delete(indexType string, itemId string) (ElasticResponse, error) {
	endpoint, err := ie.funcEndpoint(ie.index, indexType, itemId)
	if err != nil {
		return ElasticResponse{}, err
	}
	jsonResponse, err := ie.handler.Send("DELETE", endpoint, nil)
	return jsonResponse, err
}

func (i *ItemElastic) Put(item *model.Item) (ElasticResponse, error) {
	return i.send("PUT", item)
}

func (i *ItemElastic) Post(item *model.Item) (ElasticResponse, error) {
	return i.send("POST", item)
}
