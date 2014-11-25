package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/unrolled/render.v1"

	"github.com/dahernan/gopherscraper/elasticsearch"
	"github.com/dahernan/gopherscraper/scraper"
)

var (
	rnd *render.Render
)

func init() {
	// TODO remove indentation
	rnd = render.New(render.Options{IndentJSON: true})
	//rnd = render.New(render.Options{})
}

func Render() *render.Render {
	return rnd
}

type ErrJsonMarshalling struct {
	nested error
}

func (e ErrJsonMarshalling) Error() string {
	return fmt.Sprintf("Error Marshalling JSON body request with message '%v'", e.nested)
}

func RequestToJsonObject(req *http.Request, jsonDoc interface{}) error {
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(jsonDoc)
	if err != nil {
		return ErrJsonMarshalling{err}
	}
	return nil
}

func HandleHttpErrors(writer http.ResponseWriter, err error) {
	msg := make(map[string]string)
	msg["error"] = err.Error()

	modelErr, ok := err.(elastic.ModelError)
	if ok {
		Render().JSON(writer, int(modelErr.StatusCode), msg)
		return
	}

	_, ok = err.(ErrJsonMarshalling)
	if ok {
		Render().JSON(writer, http.StatusBadRequest, msg)
		return
	}

	if err == scraper.ErrNoBaseSelector {
		Render().JSON(writer, http.StatusBadRequest, msg)
		return
	}

	if err == scraper.ErrJobNotFound {
		Render().JSON(writer, http.StatusNotFound, msg)
		return
	}

	Render().JSON(writer, http.StatusInternalServerError, msg)
	return

}

func RouterWrap(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		f(w, req)
	}
}
