package elastic

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/dahernan/gopherscraper/model"
	. "github.com/smartystreets/goconvey/convey"
)

const itemData = `{
		"id": "123",
		"link": "http://localhost/123",
		"image": "http://localhost/123.jpg",
		"title": "Test",
		"price": 33.0,
		"scrapUrl": "http://localhost:9999/item1.html",
	  	"userId": "dahernan"
}`

func TestMarshal(t *testing.T) {
	Convey("Basic item marshal", t, func() {
		var item model.Item
		reader := bytes.NewReader([]byte(itemData))
		err := NewModelFromReader(reader, &item)

		So(err, ShouldBeNil)

		Convey("The item object has the right data ", func() {

			So(item.Id, ShouldEqual, "123")
			So(item.Link, ShouldEqual, "http://localhost/123")
			So(item.Image, ShouldEqual, "http://localhost/123.jpg")
			So(item.Price, ShouldEqual, 33.0)
			So(item.Title, ShouldEqual, "Test")
			So(item.ScrapUrl, ShouldEqual, "http://localhost:9999/item1.html")

			b, err := json.Marshal(item)
			t.Log(string(b))
			So(err, ShouldBeNil)
		})

	})
}
