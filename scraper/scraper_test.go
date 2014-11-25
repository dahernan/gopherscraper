package scraper

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/dahernan/gopherscraper/redis"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	example1 = `
	<html>
	<body>
		<div class="product-info">
			<h2 id="123"><a href="http://localhost/123">Test</a></h2>
			<img src="http://localhost/123.jpg"></img>
			<div class="price">£ 33.21</div>
			<div class="stars">Start 1.2</div>
		</div>
		<div class="product-info">
			<h2 id="124"><a href="http://localhost/124">Test2</a></h2>
			<img src="http://localhost/124.jpg"></img>
			<div class="price">USD 34.22</div>
			<div class="stars">Start 4.7</div>
		</div>
	</body>
	</html>
	`
)

func init() {
	redis.UseRedis("127.0.0.1")
}

func TestBasicScrap(t *testing.T) {
	Convey("Basic Scrap of two Items", t, func() {

		s := ScrapSelector{
			Url:         "http://test",
			Base:        ".product-info",
			IdPrefix:    "LO",
			Id:          Selector{Exp: "h2[id]", Attr: "id"},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{Exp: ".stars"},
		}

		scrapper := ScrapperFromReader(strings.NewReader(example1))

		jobId, items, err := scrapper.Scrap(s)
		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")
		t.Log("Items", items)

		itr := <-items
		it := itr.Item

		So(it.Title, ShouldEqual, "Test")
		So(it.Id, ShouldEqual, "LO123")
		So(it.Link, ShouldEqual, "http://localhost/123")
		So(it.Image, ShouldEqual, "http://localhost/123.jpg")
		So(it.Price, ShouldEqual, 33.21)
		So(it.Currency, ShouldEqual, "£")
		So(it.Stars, ShouldEqual, 1.2)

		itr = <-items
		it = itr.Item

		So(it.Title, ShouldEqual, "Test2")
		So(it.Id, ShouldEqual, "LO124")
		So(it.Link, ShouldEqual, "http://localhost/124")
		So(it.Image, ShouldEqual, "http://localhost/124.jpg")
		So(it.Currency, ShouldEqual, "USD")
		So(it.Price, ShouldEqual, 34.22)
		So(it.Stars, ShouldEqual, 4.7)

		itr, opened := <-items
		// assert the channel is close
		So(opened, ShouldBeFalse)

	})
}

func TestScrapIdFromUrl(t *testing.T) {
	Convey("Scrap Id from url", t, func() {

		s := ScrapSelector{
			Url:         "http://www.test.co.uk/Lauren+Ralph+Lauren+Chelo+sleeveless+wrap+gown/202817900,default,pd.html",
			Base:        ".product-info",
			IdFrom:      SelectorIdFromUrl,
			IdExtractor: ExtractId{UrlPathIndex: 2, SplitString: ",", SplitIndex: 0},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		scrapper := ScrapperFromReader(strings.NewReader(example1))

		jobId, items, err := scrapper.Scrap(s)

		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")
		t.Log("Items", items)

		itr := <-items
		it := itr.Item

		So(it.Id, ShouldEqual, "202817900")

		itr = <-items
		it = itr.Item

		So(it.Id, ShouldEqual, "202817900")

		itr, opened := <-items
		// assert the channel is close
		So(opened, ShouldBeFalse)

	})
}

func TestScrapPaginatedUrl(t *testing.T) {
	Convey("Scrap paginated  url", t, func() {

		s := ScrapSelector{
			Url:  "http://www.test2.co.uk/123?page=1",
			Base: ".product-info",
			// multiple pages
			PageParam: "page",
			PageStart: 0,
			PageIncr:  1,
			PageLimit: 3,

			Id:          Selector{Exp: "h2[id]", Attr: "id"},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		r := paginatedUrlSelector(s)
		t.Log("PAGES", r)
		So(r[0].Url, ShouldEqual, "http://www.test2.co.uk/123?page=0")
		So(r[1].Url, ShouldEqual, "http://www.test2.co.uk/123?page=1")
		So(r[2].Url, ShouldEqual, "http://www.test2.co.uk/123?page=2")
		So(len(r), ShouldEqual, 3)
	})
}

func TestScrapPaginatedUrlIncr(t *testing.T) {
	Convey("Scrap paginated with incremental url", t, func() {

		s := ScrapSelector{
			Url:  "http://www.test2.co.uk/123?start=0&size=10",
			Base: ".product-info",
			// multiple pages
			PageParam: "start",
			PageStart: 0,
			PageIncr:  10,
			PageLimit: 30,

			Id:          Selector{Exp: "h2[id]", Attr: "id"},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		r := paginatedUrlSelector(s)
		t.Log("PAGES", r)
		So(r[0].Url, ShouldEqual, "http://www.test2.co.uk/123?size=10&start=0")
		So(r[1].Url, ShouldEqual, "http://www.test2.co.uk/123?size=10&start=10")
		So(r[2].Url, ShouldEqual, "http://www.test2.co.uk/123?size=10&start=20")
		So(len(r), ShouldEqual, 3)

	})
}

func TestScrapIdFromLink(t *testing.T) {
	Convey("Scrap Id from url", t, func() {

		s := ScrapSelector{
			Url:         "test",
			Base:        ".product-info",
			IdFrom:      SelectorIdFromLink,
			IdExtractor: ExtractId{UrlPathIndex: 1},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		scrapper := ScrapperFromReader(strings.NewReader(example1))

		jobId, items, err := scrapper.Scrap(s)

		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")
		t.Log("Items", items)

		itr := <-items
		it := itr.Item

		So(it.Id, ShouldEqual, "123")

		itr = <-items
		it = itr.Item

		So(it.Id, ShouldEqual, "124")

		itr, opened := <-items
		// assert the channel is close
		So(opened, ShouldBeFalse)

	})
}

func TestExtractSnippet(t *testing.T) {
	Convey("Basic extract of the snippet", t, func() {

		s := ScrapSelector{
			Url:  "test",
			Base: ".product-info",
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(example1))
		So(err, ShouldBeNil)

		snip, err := baseSelectorSnip(s, doc)
		So(err, ShouldBeNil)

		t.Log("Snippet: ", snip)

	})
}

func TestExtractIdFromURL(t *testing.T) {
	Convey("Extract id of the item from a url", t, func() {

		Convey("amazon url", func() {
			url := "http://www.amazon.co.uk/The-Blood-Olympus-Heroes-Book/dp/0141339225/ref=zg_bs_books_25"
			id, err := ExtractIdFromURL(url, 3, "", 0)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "0141339225")
		})

		Convey("HOF url", func() {
			url := "http://www.testing.co.uk/Lauren+Ralph+Lauren+Chelo+sleeveless+wrap+gown/202817900,default,pd.html"
			id, err := ExtractIdFromURL(url, 2, ",", 0)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "202817900")
		})

		Convey("negative index url", func() {
			url := "http://www.testing.co.uk/Lauren+Ralph+Lauren+Chelo+sleeveless+wrap+gown/202817900,default,pd.html"
			id, err := ExtractIdFromURL(url, -1, ",", 0)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "202817900")
		})

		Convey("EBAY url", func() {
			url := "http://www.testing.co.uk/itm/Sony-PlayStation-3-Slimline-160-GB-Black-Console-PAL-with-8-Games-And-Gta-5-/321559037479?pt=UK_VideoGames_VideoGameConsoles_VideoGameConsoles&hash=item4ade698627"
			id, err := ExtractIdFromURL(url, 3, "", 0)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "321559037479")
		})

		Convey("HUT url", func() {
			url := "http://www.testing.co.uk/swarovski-lucky-clover-ornament-1054588.html"
			id, err := ExtractIdFromURL(url, 1, "-", 4)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "1054588")
		})

		Convey("Err index url", func() {
			url := "http://www.testing.co.uk/itm/Sony-PlayStation-3-Slimline-160-GB-Black-Console-PAL-with-8-Games-And-Gta-5-/321559037479?pt=UK_VideoGames_VideoGameConsoles_VideoGameConsoles&hash=item4ade698627"
			_, err := ExtractIdFromURL(url, 100, "", 0)
			So(err, ShouldNotBeNil)
			t.Log(err)

		})

		Convey("Err split index url", func() {
			url := "http://www.testing.co.uk/Lauren+Ralph+Lauren+Chelo+sleeveless+wrap+gown/202817900,default,pd.html"
			_, err := ExtractIdFromURL(url, 1, ",", 100)
			So(err, ShouldNotBeNil)
			t.Log(err)

		})

		Convey("Relative url", func() {
			url := "/dresses-c101/curvy-blue-embellished-detail-maxi-dress-p2679"
			id, err := ExtractIdFromURL(url, 2, "-", -1)
			So(err, ShouldBeNil)
			So(id, ShouldEqual, "p2679")
		})

		//

	})

}

func TestSanitizeURL(t *testing.T) {
	Convey("Sanitizer", t, func() {

		Convey("Do nothing", func() {
			scrapUrl := "http://www.swag.com/123.html"
			url := "http://www.test.com/123.html"
			expected := "http://www.test.com/123.html"
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("Convert relative to abosulte", func() {
			scrapUrl := "http://www.swag.com/is-bin/INTERSHOP.enfinity/WFS/SCO-Web_GB-Site/en_US/-/GBP/SPAG_HLSPage-ProductPaging"
			url := "/is-bin/intershop.static/WFS/SCO-Media-Site/-/-/publicimages//CG/B2C/PROD/180/5007735W180.jpg"
			expected := "http://www.swag.com/is-bin/intershop.static/WFS/SCO-Media-Site/-/-/publicimages//CG/B2C/PROD/180/5007735W180.jpg"
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("Remove Sid", func() {
			scrapUrl := "http://www.swag.com/is-bin/INTERSHOP.enfinity/WFS/SCO-Web_GB-Site/en_US/-/GBP/SPAG_HLSPage-ProductPaging"
			url := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html;sid=RUAbZQtsnVoQZV82ermCQ2JmQb3xioeH6T2XjqcR58r6Pg=="
			expected := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html;RUAbZQtsnVoQZV82ermCQ2JmQb3xioeH6T2XjqcR58r6Pg=="
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("Remove Sid 2", func() {
			scrapUrl := "http://www.swag.com/is-bin/INTERSHOP.enfinity/WFS/SCO-Web_GB-Site/en_US/-/GBP/SPAG_HLSPage-ProductPaging"
			url := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html?sid=RUAbZQtsnVoQZV82ermCQ2JmQb3xioeH6T2XjqcR58r6Pg=="
			expected := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html"
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("trim url", func() {
			scrapUrl := "http://www.swag.com/is-bin/INTERSHOP.enfinity/WFS/SCO-Web_GB-Site/en_US/-/GBP/SPAG_HLSPage-ProductPaging"
			url := "               http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html   "
			expected := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html"
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("trim url2", func() {
			scrapUrl := "http://www.swag.com/is-bin/INTERSHOP.enfinity/WFS/SCO-Web_GB-Site/en_US/-/GBP/SPAG_HLSPage-ProductPaging"
			url := " \t http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html  \r\n "
			expected := "http://www.swag.com/Web_GB/en/1106375/product/Angelic_Set.html"
			result := SanitizeURL(scrapUrl, url, 0)
			So(result, ShouldEqual, expected)
		})

		Convey("link slash limit 1", func() {
			scrapUrl := "http://www.amazon.co.uk/gp/bestsellers/computers/ref=sv_computers_1"
			url := "http://www.amazon.co.uk/HyperX-120GB-2-5-inch-Height-Adapter/dp/B00KW3MTBS/ref=zg_bs_computers_2/280-3309495-9013066"
			expected := "http://www.amazon.co.uk/HyperX-120GB-2-5-inch-Height-Adapter/dp/B00KW3MTBS"
			result := SanitizeURL(scrapUrl, url, 2)
			So(result, ShouldEqual, expected)
		})

	})
}

func TestExtractFloat(t *testing.T) {
	Convey("Extract Float Price from a string", t, func() {

		Convey("normal", func() {
			price := "33.22"
			f := extractFloatFromString(price)
			So(f, ShouldEqual, 33.22)
		})

		Convey("spaces", func() {
			price := "  31.21 "
			f := extractFloatFromString(price)
			So(f, ShouldEqual, 31.21)
		})

		Convey("currency £", func() {
			price := "£31.21 "
			f := extractFloatFromString(price)
			So(f, ShouldEqual, 31.21)
		})
		Convey("after £", func() {
			price := "31.21 £"
			f := extractFloatFromString(price)
			So(f, ShouldEqual, 31.21)
		})
		Convey("currency £ and long price", func() {
			price := "£ 1,0011.21 "
			f := extractFloatFromString(price)
			So(f, ShouldEqual, 10011.21)
		})
	})
}

// integration test

// needs the test web serving at http://localhost:9999/item1.html
func TestScrapIntegrationFromUrl(t *testing.T) {
	Convey("Scrap Id from http://localhost:9999/item1.html ", t, func() {

		s := ScrapSelector{
			Url:         "http://localhost:9999/item1.html",
			Base:        ".product-info",
			IdFrom:      SelectorIdFromUrl,
			IdExtractor: ExtractId{UrlPathIndex: -1},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		t.Log("Selector: ", s)

		scrapper := NewScrapper()

		jobId, items, err := scrapper.Scrap(s)

		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")

		// first
		itr := <-items
		it := itr.Item

		So(it.Link, ShouldEqual, "http://localhost/123")
		So(it.Id, ShouldEqual, "item1.html")

		_, opened := <-items
		// assert the channel is close

		So(opened, ShouldBeFalse)
		t.Log("Items", items)

	})
}

// needs the test web serving at http://localhost:9999/list.html
func TestScrapIntegrationRecursiveFromUrl(t *testing.T) {
	Convey("Scrap Id from url http://localhost:9999/list.html", t, func() {

		sList := ScrapSelector{
			Stype:       SelectorTypeList,
			Url:         "http://localhost:9999/list.html",
			Base:        ".item",
			Recursive:   true,
			IdFrom:      SelectorIdFromLink,
			IdExtractor: ExtractId{UrlPathIndex: -1},
			Link:        Selector{Exp: "a", Attr: "href"},
		}

		sDetail := ScrapSelector{
			Url:         "http://localhost:9999/item1.html",
			Base:        ".product-info",
			Stype:       SelectorTypeDetail,
			IdFrom:      SelectorIdFromUrl,
			IdExtractor: ExtractId{UrlPathIndex: -1},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		// saves Detail Selector
		data := NewRedisScrapdata()
		err := data.SaveSelector(sDetail)
		So(err, ShouldBeNil)

		scrapper := NewRecursiveScrapper()

		jobId, items, err := scrapper.Scrap(sList)

		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")
		t.Log("Items", items)

		result := make(map[string]ItemResult)
		// first
		itr := <-items
		result[itr.Item.Id] = itr

		// second
		itr = <-items
		result[itr.Item.Id] = itr

		// third
		itr = <-items
		result[itr.Item.Id] = itr

		itr, opened := <-items
		// assert the channel is close
		So(opened, ShouldBeFalse)

		So(result["item1.html"].Item.Link, ShouldEqual, "http://localhost/123")
		So(result["item2.html"].Item.Link, ShouldEqual, "http://localhost/I2")
		So(result["item3.html"].Item.Link, ShouldEqual, "http://localhost/I3")

	})
}

// needs the test web serving at http://localhost:9999/item1.html
func TestIntegrationFromUrlAndRedisStorages(t *testing.T) {
	Convey("Scrap and Redis Storage from http://localhost:9999/item1.html ", t, func() {

		s := ScrapSelector{
			Url:         "http://localhost:9999/item1.html",
			Base:        ".product-info",
			IdFrom:      SelectorIdFromUrl,
			IdExtractor: ExtractId{UrlPathIndex: -1},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		t.Log("Selector: ", s)

		scrapper := NewScrapper()
		ss := NewScrapAndStore(scrapper, []StorageItems{NewRedisStorage()})

		jobId, err := ss.ScrapAndStore(s)

		So(err, ShouldBeNil)
		So(jobId, ShouldNotEqual, "")

		rdata := NewRedisScrapdata()
		data, err := rdata.ScrapJob(jobId)

		So(err, ShouldBeNil)
		t.Log("Redis Data: ", jobId, data)

	})
}
