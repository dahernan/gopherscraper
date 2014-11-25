package elastic

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	elasticrest "github.com/dahernan/gopherscraper/jsonrequest"
	. "github.com/smartystreets/goconvey/convey"
)

// a request with mock data
func mockRequest() elasticrest.Request {
	return &RequestMock{}
}

const esDoc = `{"_index":"dahernan","_type":"board","_id":"1","_version":3,"found":true, "_source" :		
		{
	      "name": "Kanban Board",
	      "numberOfColumns": 2,
	      "columns": [
	        {"name": "Backlog", "cards": [
	          {"title": "card 1"},
	          {"title": "card 2"}
	        ]},
	        {"name": "Done", "cards": [
	          {"title": "card 3",
	            "details": "Testing Card 3 Details"},
	          {"title": "card 4",
	            "details": "Testing Card 4 Details"}
	        ]}        
	      ]
		}
	}`

const esResponse = `
	{
	    "found" : true,
	    "_index" : "twitter",
	    "_type" : "tweet",
	    "_id" : "1",
	    "_version" : 2,
	    "created" : false
	}
	`

const esError = `{"error":"MapperParsingException","status":400}`

const queryResponse = `
{
"took":27,
"timed_out":false,
"_shards":{"total":5,"successful":5,"failed":0},
"hits":{
	"total":4,
	"max_score":1.0,
	"hits":[
		{"_index":"dahernan","_type":"1_cards","_id":"ywKviQN6TeOmsRyvRDgc1w","_score":1.0,
			"_source" : {"id":"1","_version":0,"title":"one","columnName":"Backlog","details":"one","boardId":"1","organization":"dahernan"}
		},
		{"_index":"dahernan","_type":"1_cards","_id":"sF1IWT6CSRSQyJF9h8lSNg","_score":1.0,
			"_source" : {"id":"2","_version":0,"title":"one","columnName":"Backlog","details":"one","boardId":"1","organization":"dahernan"}
		}
	]
 } 
}
`

func TestMarshalElasticModel(t *testing.T) {
	Convey("Basic ElasticModel marshal", t, func() {
		var model ElasticModel
		reader := bytes.NewReader([]byte(esDoc))
		err := NewModelFromReader(reader, &model)

		So(err, ShouldBeNil)
		So(model.Id, ShouldEqual, "1")
		So(model.Index, ShouldEqual, "dahernan")
		So(model.Type, ShouldEqual, "board")
		So(model.Version, ShouldEqual, 3)

		b, err := json.Marshal(model)
		t.Log(string(b))
		So(err, ShouldBeNil)

	})
}

func TestMarshalElasticResponse(t *testing.T) {
	Convey("Basic Elastic Response marshal", t, func() {
		var responseModel ElasticResponse
		reader := bytes.NewReader([]byte(esResponse))
		err := NewModelFromReader(reader, &responseModel)

		So(err, ShouldBeNil)
		So(responseModel.Id, ShouldEqual, "1")
		So(responseModel.Index, ShouldEqual, "twitter")
		So(responseModel.Type, ShouldEqual, "tweet")
		So(responseModel.Version, ShouldEqual, 2)
		So(responseModel.Found, ShouldEqual, true)
		So(responseModel.Created, ShouldEqual, false)

	})
}

func TestMarshalElasticQueryResponse(t *testing.T) {
	Convey("Basic Elastic Query Response marshal", t, func() {
		var responseQuery ElasticQueryResponse
		reader := bytes.NewReader([]byte(queryResponse))
		err := NewModelFromReader(reader, &responseQuery)

		So(err, ShouldBeNil)
		So(responseQuery.Took, ShouldEqual, 27)
		So(responseQuery.TimedOut, ShouldEqual, false)

		So(responseQuery.Hits.MaxScore, ShouldEqual, 1)
		So(responseQuery.Hits.Total, ShouldEqual, 4)

		So(responseQuery.Hits.Hits[0].Id, ShouldEqual, "ywKviQN6TeOmsRyvRDgc1w")
		So(responseQuery.Hits.Hits[1].Id, ShouldEqual, "sF1IWT6CSRSQyJF9h8lSNg")

	})
}

func TestMarshalElasticResponseESerror(t *testing.T) {
	Convey("Elastic Response marshal an ES error", t, func() {
		var responseModel ElasticResponse
		reader := bytes.NewReader([]byte(esError))
		err := NewModelFromReader(reader, &responseModel)

		So(err, ShouldBeNil)
		So(responseModel.Error, ShouldEqual, "MapperParsingException")
		So(responseModel.Status, ShouldEqual, 400)

	})
}

func TestGet(t *testing.T) {
	Convey("Get from elasticsearch works", t, func() {
		modelHanlder := NewModelHandler(mockRequest())
		model, err := modelHanlder.Get("/")

		So(err, ShouldBeNil)
		So(model.Id, ShouldEqual, "1")
	})
}

func TestSend(t *testing.T) {
	Convey("Send to elasticsearch works", t, func() {
		modelHanlder := NewModelHandler(mockRequest())

		model, err := modelHanlder.Send("PUT", "/", nil)

		So(err, ShouldBeNil)
		So(model.Id, ShouldEqual, "1")
		So(model.Version, ShouldEqual, 3)
	})
}

// mock for a request
type RequestMock struct {
}

func (r *RequestMock) Do(method string, endpoint string, requestBody interface{}, jsonResponse interface{}) (elasticrest.StatusCode, error) {
	err := json.Unmarshal([]byte(esDoc), jsonResponse)
	So(err, ShouldBeNil)
	return http.StatusOK, nil
}
