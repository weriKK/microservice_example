package service

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func StartWebServer(port string) {
	logrus.Infoln("Starting HTTP service at " + port)

	r := NewRouter()
	http.Handle("/", r)
	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		logrus.Infoln("An error occured starting HTTP listener at port " + port)
		logrus.Infoln("Error: ", err.Error())
	}
}
