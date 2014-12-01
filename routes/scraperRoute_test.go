package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	elastic "github.com/dahernan/gopherscraper/elasticsearch"

	"github.com/dahernan/gopherscraper/jsonrequest"
	"github.com/dahernan/gopherscraper/redis"
	"github.com/dahernan/gopherscraper/scraper"
	"github.com/julienschmidt/httprouter"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	redis.UseRedis("127.0.0.1")
}

const itemData = `{"_index":"dahernan","_type":"board","_id":"1","_version":3,"found":true, "_source" :		
		{
	    	"todo": "fill this"
		}
	}`

func TestScrapTestWorks(t *testing.T) {
	// it uses the test Items served in http://localhost:9999/item1.html
	// from /test
	Convey("Scraps a single url and returns the json", t, func() {
		scraperRoute := NewScraperRoute("testindex")

		s := scraper.ScrapSelector{
			Url:         "http://localhost:9999/item1.html",
			Base:        ".product-info",
			Id:          scraper.Selector{Exp: "h2[id]", Attr: "id"},
			Link:        scraper.Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       scraper.Selector{Exp: "img[src]", Attr: "src"},
			Title:       scraper.Selector{Exp: "h2"},
			Description: scraper.Selector{},
			Price:       scraper.Selector{Exp: ".price"},
			Stars:       scraper.Selector{},
		}

		router := httprouter.New()
		router.POST("/api/scraper/test", scraperRoute.TestURL)

		ts := httptest.NewServer(router)
		url := ts.URL
		defer ts.Close()

		request := jsonrequest.NewRequest(url)

		var response map[string]interface{}
		status, err := request.Do("POST", "/api/scraper/test", &s, &response)

		So(status, ShouldEqual, 200)
		So(err, ShouldBeNil)

		t.Log(response["snippet"])

		//So(response["snippet"], ShouldStartWith, "<h2")

		// var result ItemsResponse
		scrapped := response["scrapped"].(map[string]interface{})
		items := scrapped["items"].([]interface{})
		item := items[0].(map[string]interface{})
		t.Log(item)

		//scraperRoute_test.go:74: map[scrapped:   map[items:[map[scrapUrl:http://localhost:9999/item1.html id:123 link:http://localhost/123 image:http://localhost/123.jpg title:Test price:33]]] snippet:]

		So(item["title"], ShouldEqual, "Test")
		So(item["id"], ShouldEqual, "123")
		So(item["link"], ShouldEqual, "http://localhost/123")
		So(item["image"], ShouldEqual, "http://localhost/123.jpg")
		So(item["price"], ShouldEqual, 33)

	})
}

func TestScrapStoresItemsInES(t *testing.T) {
	// it uses the test Items served in http://localhost:9999/item1.html
	// from /test
	Convey("Scraps a single url stores json in ES", t, func() {
		elastic.UserHandler(elastic.NewModelHandler(&RequestMock{itemData}))

		scraperRoute := NewScraperRoute("testindex")

		s := scraper.ScrapSelector{
			Url:         "http://localhost:9999/item1.html",
			Base:        ".product-info",
			Id:          scraper.Selector{Exp: "h2[id]", Attr: "id"},
			Link:        scraper.Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       scraper.Selector{Exp: "img[src]", Attr: "src"},
			Title:       scraper.Selector{Exp: "h2"},
			Description: scraper.Selector{},
			Price:       scraper.Selector{Exp: ".price"},
			Stars:       scraper.Selector{},
		}

		router := httprouter.New()
		router.POST("/api/scraper/scrap", scraperRoute.Scrap)

		ts := httptest.NewServer(router)
		url := ts.URL
		defer ts.Close()

		request := jsonrequest.NewRequest(url)

		var result map[string]interface{}
		status, err := request.Do("POST", "/api/scraper/scrap", &s, &result)
		t.Log(result)
		So(status, ShouldEqual, 200)
		So(err, ShouldBeNil)
		So(len(result["jobId"].(string)), ShouldBeGreaterThan, 4)
		t.Log(result["jobId"])

	})
}

// mock for a request
type RequestMock struct {
	data string
}

func (r *RequestMock) Do(method string, endpoint string, requestBody interface{}, jsonResponse interface{}) (jsonrequest.StatusCode, error) {

	err := json.Unmarshal([]byte(r.data), jsonResponse)
	if err != nil {
		panic(fmt.Sprintln("Some error in the test code ", err.Error()))
	}
	return http.StatusOK, nil
}
