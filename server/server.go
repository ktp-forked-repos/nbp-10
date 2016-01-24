package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karolgorecki/nbp/Godeps/_workspace/src/github.com/julienschmidt/httprouter"
	"github.com/karolgorecki/nbp/nbp"
)

func RegisterHandlers() *httprouter.Router {
	rt := httprouter.New()
	rt.GET("/:date/:type/:code", errorHandler(IndexHandler))

	rt.NotFound = ntHandler{}

	fmt.Println("Running on: http://localhost:" + os.Getenv("PORT"))
	return rt
}

func IndexHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) error {
	rDate := p.ByName("date")
	rType := p.ByName("type")
	rCode := p.ByName("code")

	cDate, _ := time.Parse("2006-01-02", rDate)
	if time.Now().Before(cDate) {
		handleOutput(w, http.StatusBadRequest, "You have given a future date")
		return nil
	}

	f, err := nbp.GetResourceLocation(rDate, rType)
	if err != nil {
		handleOutput(w, http.StatusBadRequest, "There was some problem with your request")
		return nil
	}

	prevData, _ := time.Parse("2006-01-02", rDate)
	for f == "" {
		prevData = prevData.AddDate(0, 0, -1)
		f, err = nbp.GetResourceLocation(prevData.Format("2006-01-02"), rType)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if f == "" {
		handleOutput(w, http.StatusBadRequest, "Resource for given date was not found")
		return nil
	}
	res, err := nbp.GetData(f, rCode)
	if err != nil {
		handleOutput(w, http.StatusBadRequest, "There was some problem with your request")
		return nil
	}

	handleOutput(w, http.StatusOK, res)
	return nil
}

// handleOutput handles the response for each endpoint.
// It follows the JSEND standard for JSON response.
// See https://labs.omniti.com/labs/jsend
func handleOutput(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)

	success := false
	if code == 200 {
		success = true
	}

	// JSend has three possible statuses: success, fail and error
	// In case of error, there is no data sent, only an error message.
	status := "success"
	msgType := "data"
	if !success {
		status = "error"
		msgType = "message"
	}

	res := map[string]interface{}{"status": status}
	if data != nil {
		res[msgType] = data
	}

	json.NewEncoder(w).Encode(res)
}

type ntHandler struct{}

func (n ntHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleOutput(w, http.StatusNotFound, "The resource you're looking was not found")
}
