package stelecom

import (
	"github.com/mihteh/stelecom/mock"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	mockServer = mock.NewServer(mock.TestTimeout)
	defer mockServer.Close()

	stelecomClient = Client(mockServer.URL, mock.TestTimeout)
	if _, err := stelecomClient.Authorize("login", "password"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

var (
	mockServer     *mock.StelecomMockServer
	testTimeout    time.Duration = 3 * time.Millisecond
	stelecomClient Exploiter
)

func TestCorrectAnswer(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceOK

	val, err := stelecomClient.GetBalance()
	if err != nil {
		t.Fatal(err)
	}

	if val == -1 {
		t.Fatal("Нет данных")
	}
}

func TestAuthOnGetBalance(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceOK
	stelecomClient.Authorize("login2", "password213")
	_, err := stelecomClient.GetBalance()
	if err == nil {
		t.Fatal("Ожидается ошибка связанная с тем что нет")
	}
}

func TestWrongJson(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceWrongJson
	_, err := stelecomClient.GetBalance()
	if err == nil {
		t.Fatal("Должна быть ошибка Wrong Json")
	}
}

func TestWrongData(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceWrongData

	_, err := stelecomClient.GetBalance()
	if err == nil {
		t.Fatal("Должна быть ошибка Wrong Data")
	}
}

func TestTimeout(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceTimeout

	_, err := stelecomClient.GetBalance()
	if err == nil {
		t.Fatal("Дожна быть ошибка по таймауту")
	}
}

func TestLargeAnswer(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsBalanceLargeData

	_, err := stelecomClient.GetBalance()
	if err == nil {
		t.Fatal("Должна быть ошибка Big Data")
	}
}

func TestGetSessionId(t *testing.T) {
	client := Client(mockServer.URL, mock.TestTimeout)

	sessionId, err := client.Authorize("login", "password")
	if err != nil {
		t.Fatal(err)
	}

	if sessionId != "5f9df3e9733582f614a0b460a3e47d77" {
		t.Fatalf("Ожидалось: [5f9df3e9733582f614a0b460a3e47d77], получено [%s]", sessionId)
	}

	if sessionId != client.GetSessionId() {
		t.Fatal("Сессии не совпадают [%s] [%s]", sessionId, client.GetSessionId())
	}
}

func TestAuthorizeFail(t *testing.T) {
	client := Client(mockServer.URL, mock.TestTimeout)

	sessionId, err := client.Authorize("login2", "password2")
	if err != nil {
		t.Fatal(err)
	}

	if sessionId != "" {
		t.Fatalf("Ожидается пустая строка, но вернулась: [%s]", sessionId)
	}
}

func TestSendSmsNotAuthorized(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsSendNotAuthorized

	sms := getValidSmsForSend()

	responseFromSendSms := stelecomClient.SendSms(sms)
	if responseFromSendSms.Error != nil {
		t.Fatal(responseFromSendSms.Error)
	}
	if responseFromSendSms.HttpStatusCode != http.StatusInternalServerError {
		t.Fatalf("Ожидался код ответа %d, получен код %d", http.StatusInternalServerError, responseFromSendSms.HttpStatusCode)
	}
}

func TestSendSms(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsSendOk

	sms := getValidSmsForSend()

	responseFromSendSms := stelecomClient.SendSms(sms)
	if responseFromSendSms.Error != nil {
		t.Fatal(responseFromSendSms.Error)
	}
	if responseFromSendSms.HttpStatusCode != http.StatusOK {
		t.Fatalf("Ожидался код ответа %d, получен код %d", http.StatusOK, responseFromSendSms.HttpStatusCode)
	}
	if !reflect.DeepEqual(responseFromSendSms.SmsIds, []string{"1404497075"}) {
		t.Fatalf("Ожидался StreamTelecomId sms: %v, получено: %v", "1404497075", responseFromSendSms.SmsIds)
	}
}

func TestSendSmsTimeOut(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsSendTimeout

	sms := getValidSmsForSend()

	responseFromSendSms := stelecomClient.SendSms(sms)
	if responseFromSendSms.Error == nil {
		t.Fatal("Ожидалась ошибка, а возвращён nil")
	}
	if responseFromSendSms.HttpStatusCode != http.StatusInternalServerError {
		t.Fatalf("Ожидался код ответа %d, получен код %d", http.StatusInternalServerError, responseFromSendSms.HttpStatusCode)
	}
}

func TestSendSmsNoMoney(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsSendNoMoney

	sms := getValidSmsForSend()

	responseFromSendSms := stelecomClient.SendSms(sms)

	if responseFromSendSms.HttpStatusCode != http.StatusPaymentRequired {
		t.Fatalf("Ожидался код ответа %d, получен код %d", http.StatusPaymentRequired, responseFromSendSms.HttpStatusCode)
	}
}

func TestGetSmsStatus(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsGetStatusOK
	mockServer.DeliveryStatus = mock.SmsStatusDelivered

	status, err := stelecomClient.GetSmsStatus("1404497075")
	if err != nil {
		t.Fatalf("Ожидался успех, а возвращена ошибка: %v", err)
	}
	if status != mock.SmsStatusDelivered {
		t.Fatalf("Ожидался статус %v, а получен статус %v", mock.SmsStatusDelivered, status)
	}
}

func TestGetSmsStatusIfBadRequest(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsGetStatusBadRequest

	status, err := stelecomClient.GetSmsStatus("")
	if err == nil {
		t.Fatalf("Ожидалась ошибка, а возвращён nil")
	}
	if status != "" {
		t.Fatalf("Ожидалась пустая строка статуса, а возвращено значение %v", status)
	}
}

func TestGetSmsStatusIfTimedout(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsGetStatusTimeout
	mockServer.DeliveryStatus = mock.SmsStatusDelivered

	status, err := stelecomClient.GetSmsStatus("1404497075")
	if err == nil {
		t.Fatalf("Должна была быть возвращена ошибка по таймауту")
	}
	if status != "" {
		t.Fatalf("Ожидалась пустая строка статуса, а возвращено значение %v", status)
	}
}

func TestGetSmsStatusIfNotAuthorized(t *testing.T) {
	mockServer.ResponseStatus = mock.SmsGetStatusNotAuthorized

	status, err := stelecomClient.GetSmsStatus("1404497075")
	if err == nil {
		t.Fatalf("Должна была быть возвращена ошибка")
	}
	if status != "" {
		t.Fatalf("Ожидалась пустая строка статуса, а возвращено значение %v", status)
	}
}
