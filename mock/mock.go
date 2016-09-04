package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

const (
	SmsBalanceOK = iota
	SmsBalanceWrongJson
	SmsBalanceWrongData
	SmsBalanceTimeout
	SmsBalanceLargeData

	SmsSendOk
	SmsSendNotAuthorized
	SmsSendTimeout
	SmsSendNoMoney

	SmsGetStatusOK
	SmsGetStatusTimeout
	SmsGetStatusNotAuthorized
	SmsGetStatusBadRequest
)

const SmsStatusDelivered = "delivered"

const TestTimeout = time.Duration(100 * time.Millisecond) // таймаут соединения для тестов

type SmsResponseStatus int

type StelecomMockServer struct {
	*httptest.Server
	ResponseStatus SmsResponseStatus // для управления типом ответа мока (успех, ошибка - какая)
	DeliveryStatus string            // для управления возвращаемым статусом SMS
	timeout        time.Duration
}

func NewServer(timeout time.Duration) *StelecomMockServer {
	var server StelecomMockServer
	server = StelecomMockServer{
		Server:         httptest.NewServer(http.HandlerFunc((&server).streamTelecomImitation)),
		ResponseStatus: SmsBalanceOK,
		DeliveryStatus: SmsStatusDelivered,
		timeout:        timeout,
	}
	return &server
}

func (server *StelecomMockServer) GetTimeout() int64 {
	return int64(server.timeout)
}

func (server *StelecomMockServer) smsImitation(w http.ResponseWriter) {
	switch server.ResponseStatus {
	case SmsBalanceOK:
		jsonData, _ := json.Marshal(123.45)
		fmt.Fprintf(w, string(jsonData))
	case SmsBalanceWrongJson:
		fmt.Fprintf(w, string("wrong Json"))
	case SmsBalanceWrongData:
		jsonData, _ := json.Marshal("string")
		fmt.Fprintf(w, string(jsonData))
	case SmsBalanceTimeout:
		time.Sleep(2 * TestTimeout)
		jsonData, _ := json.Marshal(123.45)
		fmt.Fprintf(w, string(jsonData))
	case SmsBalanceLargeData:
		s := strings.Repeat("s", 2049)
		jsonData, _ := json.Marshal(s)
		fmt.Fprintf(w, string(jsonData))
		return
	}
}

func (server *StelecomMockServer) sendSmsImitation(w http.ResponseWriter, r *http.Request) {
	switch server.ResponseStatus {
	case SmsSendOk:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`["1404497075"]`))
		return
	case SmsSendNotAuthorized:
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"Code":4,"Desc":"SessionID expired"}`))
		return
	case SmsSendTimeout:
		time.Sleep(2 * TestTimeout)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`["1404497075"]`))
		return
	case SmsSendNoMoney:
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"Code":5,"Desc":"Direction is closed"}`))
		return
	}
}

func (server *StelecomMockServer) getSmsStatusImitation(w http.ResponseWriter, r *http.Request) {
	responseOK := fmt.Sprintf(`{
"State":0,
"ReportedDateUtc":"\/Date(1440744480000)\/",
"StateDescription":"%s",
"Price":"1.130"
}`, server.DeliveryStatus)

	switch server.ResponseStatus {
	case SmsGetStatusOK:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseOK))
		return
	case SmsGetStatusNotAuthorized:
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"Code":4,"Desc":"SessionID expired"}`))
		return
	case SmsGetStatusTimeout:
		time.Sleep(2 * TestTimeout)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseOK))
		return
	case SmsGetStatusBadRequest:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"Code":1,"Desc":"MessageID can not be null or empty\r\nParameter name: messageId"}`))
	}
}

func (server *StelecomMockServer) authImitation(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	password := r.URL.Query().Get("password")

	if login != "login" || password != "password" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s := "5f9df3e9733582f614a0b460a3e47d77"
	jsonData, _ := json.Marshal(s)
	fmt.Fprintf(w, string(jsonData))
}

func (server *StelecomMockServer) streamTelecomImitation(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "/Balance") {
		server.smsImitation(w)
		return
	}

	if strings.Contains(r.RequestURI, "/Session") {
		server.authImitation(w, r)
		return
	}

	if strings.Contains(r.RequestURI, "/SendSms") {
		server.sendSmsImitation(w, r)
		return
	}

	if strings.Contains(r.RequestURI, "/State") {
		server.getSmsStatusImitation(w, r)
		return
	}
	return
}
