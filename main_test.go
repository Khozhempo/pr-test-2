// main_test.go
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHabra(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil) // Создаем запрос
	if err != nil {
		t.Fatal(err)
	}

	// Создаем ResponseRecorder для получения ответа
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleHabra)

	// Осуществляем запрос
	handler.ServeHTTP(rr, req)

	// Проверяем код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Проверяем тело ответа, чтобы хотя бы одно слово прошло замену
	if !strings.Contains(rr.Body.String(), "PopSci™") {
		t.Error("Handler returned unprocessed body")
	}

}

func TestUpdateHtmlWithMark(t *testing.T) {
	sendableHtml := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Colour</title>
</head>
<body>

<p>
    A list of colours and some awesom
</p>

<ul>
    <li>red</li>
    <li>green</li>
    <li>blue</li>
    <li>yellow</li>
    <li>orange</li>
    <li>brown</li>
    <li>pink</li>
</ul>

</body>
</html>`

	expectedHtml := `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"/><title>Colour™</title></head><body><p>A list of colours and some awesom™</p><ul><li>red</li><li>green</li><li>blue</li><li>yellow™</li><li>orange™</li><li>brown</li><li>pink</li></ul></body></html>`

	var putBuf bytes.Buffer
	putBuf.Write([]byte(sendableHtml))
	got, err := UpdateHtmlWithMark(&putBuf)
	// fmt.Println(string(got))
	if err != nil || string(got) != expectedHtml {
		t.Errorf("Expected/return:\r\n %s \r\n%s", expectedHtml, string(got))
	}
}

func TestAddMarkToEveryLargeWord(t *testing.T) {
	sendableStr := []string{" крышка,  ", "  лифт", "А также стремящиеся вытеснить традиционное производство, нанотехнологии указаны как претенденты на роль ключевых факторов. С учётом сложившейся международной обстановки, граница обучения кадров напрямую зависит от соответствующих условий активизации. А также некоторые особенности внутренней политики, инициированные исключительно синтетически, своевременно верифицированы."}
	expectedStr := []string{"крышка™,", "лифт", "А также стремящиеся вытеснить традиционное производство, нанотехнологии указаны как претенденты на роль ключевых факторов. С учётом™ сложившейся международной обстановки, граница обучения кадров™ напрямую зависит от соответствующих условий активизации. А также некоторые особенности внутренней политики, инициированные исключительно синтетически, своевременно верифицированы."}

	for k, v := range sendableStr {
		got := AddMarkToEveryLargeWord(v)
		if got != expectedStr[k] {
			t.Errorf("Expected (%s), return (%s)", expectedStr[k], got)
		}
	}
}

func TestIsHTTPSHabrUrl(t *testing.T) {
	sendable := make(map[string]bool)
	sendable["www.habrahabr.ru"] = true
	sendable["habrahabr.ru"] = true
	sendable["lenta.ru/index.html?query=something"] = false
	sendable["habr.com/ru/all/"] = false

	for k, _ := range sendable {
		if IsHTTPSHabrUrl(k) != sendable[k] {
			t.Errorf("On %s expected %t, return %t", k, sendable[k], !sendable[k])
		}
	}
}

func TestCleanString(t *testing.T) {
	sendableStr := "  ,testIng saas. "
	expectedStr := "testIng saas"

	got := CleanString(sendableStr)
	if got != expectedStr {
		t.Errorf("Expected %s, return %s", expectedStr, got)
	}
}
