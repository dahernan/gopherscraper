package routes

import (
	"net/http"

	esearch "github.com/dahernan/gopherscraper/elasticsearch"
	"github.com/julienschmidt/httprouter"
)

type ItemsRoute struct {
	es *esearch.ItemElastic
}

func NewItemsRoute() *ItemsRoute {
	return &ItemsRoute{esearch.NewItemElastic("gopherscrap")}
}

func (r *ItemsRoute) Get(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	itemId := params.ByName("id")
	index := params.ByName("index")

	item, err := r.es.Get(index, itemId)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	Render().JSON(w, http.StatusOK, item)
}

func (r *ItemsRoute) MultiGet(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	index := params.ByName("index")

	var docs map[string][]string
	err := RequestToJsonObject(req, &docs)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	items, err := r.es.MultiGet(index, docs["ids"])
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	Render().JSON(w, http.StatusOK, items)
}

func (r *ItemsRoute) Search(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	index := params.ByName("index")

	var query interface{}
	err := RequestToJsonObject(req, &query)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	response, err := r.es.Search(index, query)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	Render().JSON(w, http.StatusOK, response)
}
