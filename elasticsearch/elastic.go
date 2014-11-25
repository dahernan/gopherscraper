package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"

	elasticrest "github.com/dahernan/gopherscraper/jsonrequest"
)

var (
	defaultHandler *ModelHandler
)

func UserHandler(m *ModelHandler) {
	defaultHandler = m
}
func Handler() *ModelHandler {
	if defaultHandler == nil {
		panic("Can not use the ModelHandler because the defaultModelHandler is nil")
	}
	return defaultHandler
}

func NewModelFromReader(reader io.Reader, object interface{}) error {
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(object)
	return err
}

func NewModelFromRaw(source json.RawMessage, object interface{}) error {
	return json.Unmarshal(source, object)
}

/* Elastic Search basic response
{
     _index: "dahernan",
     _type: "board",
     _id: "1",
     _version: 3,
     found: true,
     _source: {....}
}
*/

type ElasticModel struct {
	Id      string          `json:"_id"`
	Type    string          `json:"_type"`
	Index   string          `json:"_index"`
	Version int             `json:"_version"`
	Found   bool            `json:"found,omitempty"`
	Fields  json.RawMessage `json:"fields,omitempty"`
	Source  json.RawMessage `json:"_source,omitempty"`
}

/* Elastic search Put/Post response
{
		_index: "dahernan"
		_type: "boards"
		_id: "1"
		_version: 2
		created: false
		found : true,
}
*/
/*  ES error
`{"error":"MapperParsingException","status":400}`
*/

type ElasticResponse struct {
	Id      string  `json:"_id"`
	Type    string  `json:"_type"`
	Index   string  `json:"_index"`
	Version int     `json:"_version"`
	Created bool    `json:"created"`
	Found   bool    `json:"found"`
	Error   string  `json:"error"`
	Status  int     `json:"status"`
	Score   float64 `json:"_score"`
}

/* ElasticQueryResponse
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
 } // hits
}
*/

type ElasticQueryResponse struct {
	Took     float64     `json:"took"`
	TimedOut bool        `json:"timed_out"`
	Hits     ElasticHits `json:"hits"`
}

type ElasticHits struct {
	Total    int            `json:"total"`
	MaxScore float64        `json:"max_score"`
	Hits     []ElasticModel `json:"hits"`
}

type ModelHandler struct {
	HttpClient elasticrest.Request
}

type ModelError struct {
	Nested     error
	Msg        string
	StatusCode elasticrest.StatusCode
}

func NewModelError(msg string, statusCode elasticrest.StatusCode, nested error) ModelError {
	return ModelError{
		Nested:     nested,
		Msg:        msg,
		StatusCode: statusCode,
	}

}

func (m ModelError) Error() string {
	if m.Nested != nil {
		return fmt.Sprintf("ModelError: [%v] %s with nested error %s", m.StatusCode, m.Msg, m.Nested)
	}
	return fmt.Sprintf("ModelError: [%v] %s", m.StatusCode, m.Msg)
}

func NewModelHandler(httpClient elasticrest.Request) *ModelHandler {
	modelHandler := ModelHandler{
		HttpClient: httpClient,
	}
	return &modelHandler
}

func (h *ModelHandler) Get(endpoint string) (*ElasticModel, error) {
	var model ElasticModel

	statusCode, err := h.HttpClient.Do("GET", endpoint, nil, &model)

	if err != nil {
		log.Println("ERROR: ModelHandler Get: ", err.Error())
		return nil, ModelError{Nested: err, Msg: "error in the request", StatusCode: statusCode}
	}

	if statusCode >= 400 && statusCode <= 499 {
		return nil, ModelError{Nested: err, Msg: "error in the request", StatusCode: statusCode}
	}

	if !(statusCode >= 200 && statusCode <= 299) {
		return nil, ModelError{Nested: err, Msg: "error with the get response ", StatusCode: statusCode}
	}

	return &model, nil
}

func (h *ModelHandler) Send(method string, endpoint string, jsonBody interface{}) (ElasticResponse, error) {
	var doc ElasticResponse

	statusCode, err := h.HttpClient.Do(method, endpoint, jsonBody, &doc)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return doc, ModelError{Nested: err, Msg: doc.Error, StatusCode: statusCode}
	}
	if !(statusCode >= 200 && statusCode <= 299) {
		return doc, ModelError{Nested: err, Msg: doc.Error, StatusCode: statusCode}
	}

	return doc, nil
}

func (h *ModelHandler) SendRAW(method string, endpoint string, jsonBody interface{}, response interface{}) error {

	statusCode, err := h.HttpClient.Do(method, endpoint, jsonBody, response)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return ModelError{Nested: err, Msg: err.Error(), StatusCode: statusCode}
	}
	if !(statusCode >= 200 && statusCode <= 299) {
		return ModelError{Nested: err, Msg: err.Error(), StatusCode: statusCode}
	}

	return nil
}

func (h *ModelHandler) Query(endpoint string, jsonBody interface{}) (*ElasticQueryResponse, error) {
	var doc ElasticQueryResponse

	statusCode, err := h.HttpClient.Do("GET", endpoint, jsonBody, &doc)
	if err != nil {
		log.Println("ERROR: ModelHandler Query: ", err.Error())
		return nil, ModelError{Nested: err, Msg: "ModelHandler:Query, error in the request", StatusCode: statusCode}
	}

	if statusCode >= 400 && statusCode <= 499 {
		return nil, ModelError{Nested: err, Msg: "ModelHandler:Query, error in the request", StatusCode: statusCode}
	}

	if !(statusCode >= 200 && statusCode <= 299) {
		return nil, ModelError{Nested: err, Msg: "ModelHandler:Query, error with the get response ", StatusCode: statusCode}
	}

	return &doc, nil
}

func BuildQuery(t *template.Template, data interface{}) (map[string]interface{}, error) {
	var jsonQuery map[string]interface{}
	var queryDoc bytes.Buffer

	err := t.Execute(&queryDoc, data)
	if err != nil {
		return nil, err
	}

	queryReader := bytes.NewReader(queryDoc.Bytes())
	NewModelFromReader(queryReader, &jsonQuery)

	return jsonQuery, nil

}
