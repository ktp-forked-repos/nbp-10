package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karolgorecki/nbp/svc"

	"github.com/julienschmidt/httprouter"
)

// RegisterHandlers does something
func RegisterHandlers() *httprouter.Router {
	rt := httprouter.New()
	rt.GET("/:date/:type/:code", errorHandler(IndexHandler))

	rt.NotFound = ntHandler

	fmt.Println("Running on: http://localhost:" + os.Getenv("PORT"))
	return rt
}

// IndexHandler Does something
func IndexHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) error {
	rDate := p.ByName("date")
	rType := p.ByName("type")
	rCode := p.ByName("code")

	// Is the given date OK?
	date, err := time.Parse("2006-01-02", rDate)
	if err != nil {
		handleOutput(w, http.StatusBadRequest, "Given date is wrong. Use 'YYYY-MM-DD'")
		return nil
	}

	// Is the type OK?
	if rType != "avg" && rType != "both" {
		handleOutput(w, http.StatusBadRequest, "Given type is wrong. Use 'avg' or 'both'")
		return nil
	}

	// Disable future date
	sDate := date.Format("2006-01-02")
	if time.Now().Before(date) {
		handleOutput(w, http.StatusBadRequest, "Given date is wrong. Can't use future date")
		return nil
	}

	// Disable date before 2002-01-02 -> first record in NBP
	minDate, _ := time.Parse("2006-01-02", "2002-01-02")
	if date.Before(minDate) {
		handleOutput(w, http.StatusBadRequest, "Given date is wrong. Min date is 2002-01-02")
		return nil
	}

	// Get the file containing the the currency data
	f, err := svc.GetResourceLocation(sDate, rType)
	if err != nil {
		handleOutput(w, http.StatusBadRequest, "There was some problem with your request")
		return nil
	}

	// When the currency rate was not found for given date try to go back one day.
	// It's used to get currencies for holidays, or weekends
	prevData := date
	for f == "" {
		prevData = prevData.AddDate(0, 0, -1)
		f, err = svc.GetResourceLocation(prevData.Format("2006-01-02"), rType)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if f == "" {
		handleOutput(w, http.StatusBadRequest, "Resource for given date was not found")
		return nil
	}
	res, err := svc.GetData(f, rCode)
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

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Fatal(err)
	}
}

func ntHandler(w http.ResponseWriter, r *http.Request) {
	handleOutput(w, http.StatusNotFound, "The resource you're looking was not found")
}
