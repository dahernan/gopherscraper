package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
	"gopkg.in/unrolled/render.v1"

	"github.com/dahernan/gopherscraper/elasticsearch"
	"github.com/dahernan/gopherscraper/jsonrequest"
	"github.com/dahernan/gopherscraper/redis"
	"github.com/dahernan/gopherscraper/routes"
)

var (
	rnd               *render.Render
	timeout           = time.Duration(2 * time.Second)
	elasticRestClient jsonrequest.Request
)

func init() {
	rnd = render.New(render.Options{})

}

func Render() *render.Render {
	return rnd
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "200 OK!\n")
}

func NotFound(w http.ResponseWriter, req *http.Request) {
	errDoc := make(map[string]interface{})
	errDoc["uri"] = req.RequestURI
	errDoc["status"] = http.StatusNotFound
	Render().JSON(w, http.StatusNotFound, errDoc)
}

func main() {
	viper.SetDefault("REDIS", "127.0.0.1")
	viper.SetDefault("ES", "http://coreos1:9200")
	viper.SetDefault("PORT", ":3001")

	rhost := viper.GetString("REDIS")
	redis.UseRedis(rhost)
	es := viper.GetString("ES")
	elasticRestClient = jsonrequest.NewRequestWithTimeout(es, timeout)

	fmt.Println("Using Redis: ", rhost)
	fmt.Println("Using ES: ", es)

	elastic.UserHandler(elastic.NewModelHandler(elasticRestClient))

	router := httprouter.New()
	router.NotFound = NotFound

	itemsRoute := routes.NewItemsRoute()
	scraperRoute := routes.NewScraperRoute()

	// global items or web items
	// TODO find a better naming
	router.GET("/api/web/:index/:id", itemsRoute.Get)
	router.POST("/api/multi/web/:index", itemsRoute.MultiGet)
	router.GET("/api/multi/web/:index", itemsRoute.MultiGet)
	router.GET("/api/search/web/:index", itemsRoute.Search)
	router.POST("/api/search/web/:index", itemsRoute.Search)

	router.POST("/api/scraper/test", scraperRoute.TestURL)
	router.POST("/api/scraper/scrap", scraperRoute.Scrap)
	router.POST("/api/scraper/selector", scraperRoute.Selector)
	router.GET("/api/scraper/job/:id", scraperRoute.StatusJob)

	n := negroni.Classic()
	n.UseHandler(router)

	port := viper.GetString("PORT")
	n.Run(port)

}
