package main

import (
	"log"
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
	"github.com/dahernan/gopherscraper/scraper"
)

var (
	rnd               *render.Render
	timeout           = time.Duration(2 * time.Second)
	elasticRestClient jsonrequest.Request
)

func init() {
	rnd = render.New(render.Options{})

}

func main() {
	viper.SetDefault("REDIS", "127.0.0.1")
	viper.SetDefault("ES", "http://coreos1:9200")
	viper.SetDefault("PORT", ":3001")
	viper.SetDefault("INDEX", "gopherscrap")
	viper.SetDefault("USER_AGENT", "gopherscraper")
	viper.SetDefault("MAX_CONNECTIONS", 500)

	rhost := viper.GetString("REDIS")
	es := viper.GetString("ES")
	port := viper.GetString("PORT")
	index := viper.GetString("INDEX")
	userAgent := viper.GetString("USER_AGENT")
	maxConnections := viper.GetInt("MAX_CONNECTIONS")

	log.Println("Using Redis: ", rhost)
	log.Println("Using ES: ", es)
	log.Println("Using ES INDEX: ", index)
	log.Println("Using PORT: ", port)
	log.Println("Using USER_AGENT: ", userAgent)
	log.Println("Using MAX_CONNECTIONS: ", maxConnections)

	redis.UseRedis(rhost)

	elasticRestClient = jsonrequest.NewRequestWithTimeout(es, timeout)
	elastic.UserHandler(elastic.NewModelHandler(elasticRestClient))

	scraper.UseUserAgent(userAgent)
	scraper.UseMaxConnections(maxConnections)

	router := httprouter.New()
	router.NotFound = NotFound

	itemsRoute := routes.NewItemsRoute(index)
	scraperRoute := routes.NewScraperRoute(index)

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
	router.GET("/api/scraper/log", scraperRoute.Log)
	router.GET("/api/scraper/job/:id", scraperRoute.StatusJob)

	n := negroni.Classic()
	n.UseHandler(router)

	n.Run(port)

}

func Render() *render.Render {
	return rnd
}

func NotFound(w http.ResponseWriter, req *http.Request) {
	errDoc := make(map[string]interface{})
	errDoc["uri"] = req.RequestURI
	errDoc["status"] = http.StatusNotFound
	Render().JSON(w, http.StatusNotFound, errDoc)
}
