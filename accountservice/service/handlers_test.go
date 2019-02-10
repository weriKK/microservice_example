package service

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
	"github.com/weriK/goblog/accountservice/dbclient"
	"github.com/weriK/goblog/accountservice/model"
	"github.com/weriK/goblog/common/messaging"
)

var mockRepo = &dbclient.MockBoltClient{}
var mockMessagingClient = &messaging.MockMessagingClient{}

// declare mock types to make test code a bit more readable
var anyString = mock.AnythingOfType("string")
var anyByteArray = mock.AnythingOfType("[]uint8") // == []byte

func init() {
	gock.InterceptClient(client)
}

func TestGetAccount(t *testing.T) {
	defer gock.Off()
	gock.New("http://quotes-service:8080").
		Get("/api/quote").
		MatchParam("strength", "4").
		Reply(200).
		BodyString(`{"quote":"May the source be with you. Always.","ipAddress":"10.20.30.40:5060","language":"en"}`)

	mockRepo.On("QueryAccount", "123").Return(model.Account{ID: "123", Name: "Person_123"}, nil)
	mockRepo.On("QueryAccount", "456").Return(model.Account{}, logrus.Errorf("Some error"))

	DBClient = mockRepo

	Convey("Given a HTTP request for /accounts/123", t, func() {
		req := httptest.NewRequest("GET", "/accounts/123", nil)
		resp := httptest.NewRecorder()

		Convey("When the request is handled by the Router", func() {
			NewRouter().ServeHTTP(resp, req)

			Convey("Then the response should be a 200", func() {
				So(resp.Code, ShouldEqual, 200)

				account := model.Account{}
				json.Unmarshal(resp.Body.Bytes(), &account)
				So(account.ID, ShouldEqual, "123")
				So(account.Name, ShouldEqual, "Person_123")
				So(account.Quote.Text, ShouldEqual, "May the source be with you. Always.")
			})
		})
	})

	Convey("Given a HTTP request for /accounts/456", t, func() {
		req := httptest.NewRequest("GET", "/accounts/456", nil)
		resp := httptest.NewRecorder()

		Convey("When the request is handled by the Router", func() {
			NewRouter().ServeHTTP(resp, req)

			Convey("Then the response should be a 404", func() {
				So(resp.Code, ShouldEqual, 404)
			})
		})
	})
}

func TestGetAccountWrongPath(t *testing.T) {
	Convey("Given a HTTP request for /invalid/123", t, func() {
		req := httptest.NewRequest("GET", "/invalid/123", nil)
		resp := httptest.NewRecorder()

		Convey("When the request is handled byt he Router", func() {
			NewRouter().ServeHTTP(resp, req)

			Convey("Then the response should be a 404", func() {
				So(resp.Code, ShouldEqual, 404)
			})
		})
	})
}

func TestNotificationIsSentForVIPAccount(t *testing.T) {
	// Set up the DB client mock
	mockRepo.On("QueryAccount", "10000").Return(model.Account{ID: "10000", Name: "Person_10000"}, nil)
	DBClient = mockRepo

	mockMessagingClient.On("PublishOnQueue", anyByteArray, anyString).Return(nil)
	MessagingClient = mockMessagingClient

	Convey("Given a HTTP req for a VIP account", t, func() {
		req := httptest.NewRequest("GET", "/accounts/10000", nil)
		resp := httptest.NewRecorder()
		Convey("When the request is handled by the Router", func() {
			NewRouter().ServeHTTP(resp, req)
			Convey("Then the response should be a 200 and the MessageClient should have been invoked", func() {
				So(resp.Code, ShouldEqual, 200)
				time.Sleep(time.Millisecond * 10) // Sleep since the Assert below occurs in goroutine
				So(mockMessagingClient.AssertNumberOfCalls(t, "PublishOnQueue", 1), ShouldBeTrue)
			})
		})
	})
}
