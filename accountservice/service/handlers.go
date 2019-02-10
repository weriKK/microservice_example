package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/weriK/goblog/accountservice/model"
	"github.com/weriK/goblog/common/messaging"

	"github.com/gorilla/mux"
	"github.com/weriK/goblog/accountservice/dbclient"
)

var DBClient dbclient.IBoltClient
var MessagingClient messaging.IMessagingClient

var isHealthy = true

var client = &http.Client{}

func init() {
	var transport http.RoundTripper = &http.Transport{
		DisableKeepAlives: true,
	}
	client.Transport = transport
}

func GetAccount(w http.ResponseWriter, r *http.Request) {

	var accountID = mux.Vars(r)["accountId"]

	account, err := DBClient.QueryAccount(accountID)
	if err != nil {
		logrus.Infoln("Some error occured serving " + accountID + ": " + err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	account.ServedBy = getIP()

	notifyVIP(account)

	quote, err := getQuote()
	if err == nil {
		account.Quote = quote
	}

	data, _ := json.Marshal(account)
	writeJsonResponse(w, http.StatusOK, data)
}

func notifyVIP(account model.Account) {
	if account.ID == "10000" {
		go func(account model.Account) {
			vipNotification := model.VipNotification{AccountId: account.ID, ReadAt: time.Now().UTC().String()}
			data, _ := json.Marshal(vipNotification)
			err := MessagingClient.PublishOnQueue(data, "vipQueue")
			if err != nil {
				logrus.Infoln(err.Error())
			}
		}(account)
	}
}

func getQuote() (model.Quote, error) {
	req, _ := http.NewRequest("GET", "http://quotes-service:8080/api/quote?strength=4", nil)
	resp, err := client.Do(req)

	if err == nil && resp.StatusCode == 200 {
		quote := model.Quote{}
		bytes, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(bytes, &quote)
		return quote, nil
	} else {
		return model.Quote{}, fmt.Errorf("Some error")
	}
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "error"
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("Unable to determine local IP address (non loopback). Exiting.")
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbUp := DBClient.Check()
	if dbUp && isHealthy {
		data, _ := json.Marshal(healthCheckResponse{Status: "UP"})
		writeJsonResponse(w, http.StatusOK, data)
	} else {
		data, _ := json.Marshal(healthCheckResponse{Status: "Database unaccessible"})
		writeJsonResponse(w, http.StatusServiceUnavailable, data)
	}
}
func writeJsonResponse(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	w.Write(data)
}

type healthCheckResponse struct {
	Status string `json:"status"`
}

func SetHealthyState(w http.ResponseWriter, r *http.Request) {

	// Read the 'state' path parameter from the mux map and convert to a bool
	var state, err = strconv.ParseBool(mux.Vars(r)["state"])

	// If we couldn't parse the state param, return a HTTP 400
	if err != nil {
		logrus.Infoln("Invalid request to SetHealthyState, allowed values are true or false")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Otherwise, mutate the package scoped "isHealthy" variable.
	isHealthy = state
	w.WriteHeader(http.StatusOK)
}
