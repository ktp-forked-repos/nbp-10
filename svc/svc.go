package svc

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

const (
	// a - tabela kursów średnich walut obcych;
	// c - tabela kursów kupna i sprzedaży;
	avg                = "a"
	both               = "c"
	nbpAPI             = "http://www.nbp.pl/kursy/xml/"
	errCannotParseDate = "Couldn't parse the given date"
	errNbpAPIProblem   = "Couldn't get data from NBP API"
)

func getFormatedDate(date string) (string, error) {
	tmpDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", errors.New("Couldn't not parse the given date")
	}
	date = tmpDate.Format("060102")
	return date, nil
}

func getFormatedYear(date string) (string, error) {
	tmpDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", errors.New(errCannotParseDate)
	}

	fullYear := tmpDate.Format("2006")
	currentYear := time.Now().Format("2006")

	yr := ""
	if currentYear != fullYear {
		yr = fullYear
	}

	return yr, nil
}

// GetResourceLocation returns name of file that contains the currencies for given date
// We're searching the index which is just a txt file.
// E.g.: dir2015.txt - contains references to files that have currencies for 2015 year
// E.g.: dir.txt - contains references for the current year.
func GetResourceLocation(sDate string, sType string) (string, error) {

	yr, err := getFormatedYear(sDate)
	fmtDate, err := getFormatedDate(sDate)
	resp, err := http.Get(nbpAPI + "dir" + yr + ".txt")
	defer func() {
		err = resp.Body.Close()
	}()
	if err != nil {
		return "", errors.New(errNbpAPIProblem)
	}

	lns := bufio.NewReader(resp.Body)
	scn := bufio.NewScanner(lns)

	fLsts := []string{}

	for scn.Scan() {
		if strings.Contains(scn.Text(), fmtDate) {
			fLsts = append(fLsts, scn.Text())
		}
	}

	var resourceName string
	for _, entry := range fLsts {
		switch sType {
		case "avg":
			if string(entry[0]) == avg {
				resourceName = string(entry)
			}
		case "both":
			if string(entry[0]) == both {
				resourceName = string(entry)
			}
		default:
			resourceName = ""
		}
	}
	return resourceName, nil
}

// GetData fetches the currency/currencies in given file.
// There is one file for one date. E.g.: 2015-01-02 is a20123123.xml
func GetData(file string, code string) (Query, error) {

	resp, err := http.Get(nbpAPI + file + ".xml")
	defer func() {
		err = resp.Body.Close()
	}()
	if err != nil {
		return Query{}, errors.New("Couldn't not get data from NBP api")
	}

	var q Query
	currencyData, _ := ioutil.ReadAll(resp.Body)
	cData := bytes.NewReader(currencyData)

	decoder := xml.NewDecoder(cData)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&q)

	res := q
	res.Currencies = res.Currencies[:0]

	if code == "*" {
		return q, nil
	}

	codes := strings.Split(code, ",")
	for _, c := range q.Currencies {

		for _, cd := range codes {
			if cd == c.Code {
				res.Currencies = append(res.Currencies, c)
			}
		}
	}
	return res, nil
}

// Query ...
type Query struct {
	FromData    string     `xml:"data_publikacji" json:"fromDate"`
	TableNumber string     `xml:"numer_tabeli" json:"tableNumber"`
	Currencies  []currency `xml:"pozycja" json:"currencies"`
}

type currency struct {
	Code    string `xml:"kod_waluty" json:"code"`
	Name    string `xml:"nazwa_waluty" json:"name"`
	Ratio   string `xml:"przelicznik" json:"ratio"`
	Average string `xml:"kurs_sredni" json:"average"`
	Buy     string `xml:"kurs_kupna" json:"buy"`
	Sell    string `xml:"kurs_sprzedazy" json:"sell"`
}
