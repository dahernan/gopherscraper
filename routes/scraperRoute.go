package routes

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/yosssi/gohtml"

	"github.com/dahernan/gopherscraper/model"
	"github.com/dahernan/gopherscraper/scraper"
)

type ScraperRoute struct {
}

type ItemsResponse struct {
	Items []model.Item `json:"items"`
}

func NewScraperRoute() *ScraperRoute {
	return &ScraperRoute{}
}

func (route *ScraperRoute) Selector(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var selector scraper.ScrapSelector
	err := RequestToJsonObject(r, &selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	rdata := scraper.NewRedisScrapdata()
	s, err := rdata.Selector(selector.Url, selector.Stype)
	if err != nil {
		if err == scraper.ErrSelectorNotFound {
			Render().JSON(w, http.StatusNotFound, "Url not register for scrapping")
			return
		}
		HandleHttpErrors(w, err)
		return
	}

	Render().JSON(w, http.StatusOK, s)

}

func (route *ScraperRoute) TestURL(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var selector scraper.ScrapSelector
	err := RequestToJsonObject(r, &selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	// make sure only test one page
	selector.PageParam = ""

	scr := scraper.NewScrapper()

	jobId, itemsc, err := scr.Scrap(selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	var items []model.Item
	for it := range itemsc {
		err = it.Err
		if err != nil {
			HandleHttpErrors(w, err)
			return
		}
		items = append(items, it.Item)
	}

	snippet, err := scraper.SnippetBase(selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	scrapped := &ItemsResponse{
		Items: items,
	}

	result := map[string]interface{}{
		"jobId":    jobId,
		"snippet":  gohtml.Format(snippet),
		"scrapped": scrapped,
	}

	Render().JSON(w, http.StatusOK, result)

}

func (route *ScraperRoute) Scrap(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var selector scraper.ScrapSelector
	err := RequestToJsonObject(r, &selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	es := scraper.NewElasticScrapAndStore()
	jobId, err := es.ScrapAndStore(selector)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	response := map[string]interface{}{
		"jobId": jobId,
	}

	Render().JSON(w, http.StatusOK, response)

}

func (route *ScraperRoute) StatusJob(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	jobId := params.ByName("id")

	data := scraper.NewRedisScrapdata()
	resp, err := data.ScrapJob(jobId)
	if err != nil {
		HandleHttpErrors(w, err)
		return
	}

	Render().JSON(w, http.StatusOK, resp)

}
