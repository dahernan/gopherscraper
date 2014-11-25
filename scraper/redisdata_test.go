package scraper

import (
	"testing"

	"github.com/dahernan/gopherscraper/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	redis.UseRedis("127.0.0.1")
}

func TestSaveSelector(t *testing.T) {
	Convey("Saves the selector based in the Url", t, func() {

		s := ScrapSelector{
			Url:         "testURL",
			Base:        ".product-info",
			IdPrefix:    "LO",
			Stype:       SelectorTypeDetail,
			Id:          Selector{Exp: "h2[id]", Attr: "id"},
			Link:        Selector{Exp: "h2 a[href]", Attr: "href"},
			Image:       Selector{Exp: "img[src]", Attr: "src"},
			Title:       Selector{Exp: "h2"},
			Description: Selector{},
			Price:       Selector{Exp: ".price"},
			Stars:       Selector{},
		}

		data := NewRedisScrapdata()

		err := data.SaveSelector(s)
		So(err, ShouldBeNil)

		fromRedis, err := data.Selector("testURL", "detail")
		So(err, ShouldBeNil)
		So(s, ShouldResemble, fromRedis)

	})
}

func TestSelectorNotFound(t *testing.T) {
	Convey("get non existing selector", t, func() {

		data := NewRedisScrapdata()

		_, err := data.Selector("404", "")
		So(err, ShouldEqual, ErrSelectorNotFound)

	})
}
