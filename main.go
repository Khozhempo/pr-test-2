package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func main() {
	log.Println("started")
	server := &http.Server{ // формируем параметры хэндле сервера
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if IsHTTPSHabrUrl(r.Host) { // выявление обращений к хабру
				HandleHabra(w, r) // вызов обработчика к хабру
				return
			}

			// классический прокси по прочим вызовам
			if r.Method == http.MethodConnect {
				handleHTTPS(w, r) // обработчик обращений к https сайтам
			} else {
				handleHTTP(w, r) // обработчик к https сайтам
			}
		}),
	}
	err := server.ListenAndServe() // запускаем сервер
	if err != nil {
		log.Fatalf("Error while start server, %s", err)
	}
}

// Обработчик обращений к habra
func HandleHabra(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}
	res, err := client.Get("https://habr.com" + req.URL.Path) // берем контент с ключевого сайта
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer res.Body.Close()

	body, err := UpdateHtmlWithMark(res.Body) // обновляем содержимое
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	copyHeader(w.Header(), res.Header) // копируем заголовки
	w.Write(body)                      // возвращаем обновленное содержимое
}

// UpdateHtmlWithMark
// Обновление html файла
func UpdateHtmlWithMark(r io.Reader) (w []byte, err error) {
	var buf bytes.Buffer

	node, err := html.Parse(r) // парсим входящий Html файл
	if err != nil {
		return nil, err
	}

	htmlReplace(node) // обновляем содержимое

	err = html.Render(&buf, node) // рендерим обратно в html
	if err != nil {
		return nil, err
	}

	w = buf.Bytes()

	return w, nil
}

// htmlReplace
// замена текста в html
func htmlReplace(n *html.Node) {
	if n.Type == html.TextNode { // выборка всех текстовых элементов html файла
		n.Data = AddMarkToEveryLargeWord(n.Data) // их обработка и переназначение
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling { // перевызов для вложенных элементов
		htmlReplace(child)
	}
}

// AddMarkToEveryLargeWord
// Добавляем к каждому слову размером в 6 букв установленный знак
// На входе и выходе текстовая строка
func AddMarkToEveryLargeWord(inStr string) string {
	inStr = strings.TrimSpace(inStr)        // убираем лишние пробелы
	strInSlice := strings.Split(inStr, " ") // разбиваем строку на слова
	var outStrSlice []string                // сборка обработанных слов обратно в слайс
	reg := regexp.MustCompile("([\\p{L}]{6})")

	for _, word := range strInSlice {
		if len([]rune(CleanString(word))) == 6 { // если длина слова (с учетом кодировки) нужной длины
			word = reg.ReplaceAllString(word, "$1"+"™") // обновление, regexp плохо выделяет слова unicode, поэтому делаем это вручную
		}
		outStrSlice = append(outStrSlice, word) // сборка обработанных слов в слайс
	}

	return strings.Join(outStrSlice, " ") // экспорт в строку
}

// Очистка строки от пробелов и знаков препинания
func CleanString(inStr string) string {
	inStr = strings.TrimSpace(inStr)
	inStr = strings.Replace(inStr, ",", "", -1)
	return strings.Replace(inStr, ".", "", -1)
}

// Классический proxy для прочих обращений, HTTP
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req) // Запрос к серверу
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header) // Копирование параметров http заголовка от запрашиваемого сервера к клиенту
	w.WriteHeader(resp.StatusCode)      // Возвращаем клиенту статус статус ответа запрашиваемого сервера
	io.Copy(w, resp.Body)               // Копирование потока от запрашиваемого сервера к клиенту
}

// Копирование параметров http заголовка в ответный
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Соединение каналов
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()

	io.Copy(destination, source)
}

// Классический proxy для прочих обращений, HTTPS
func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second) // Запрос к целевому серверу
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack() // Перехватываем управление каналом, закрытие обеспечивается в transfer'е
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	//
	go transfer(dest_conn, client_conn) // соединения каналов от клиента к серверу
	go transfer(client_conn, dest_conn) // соединения каналов от сервера к клиенту
}

// IsHTTPSHabrUrl подтверждает является ли запрашиваемый адрес из пула habrahabr сайтов
// на входе url в формате r.Host
// на выходе bool, true - если адрес принадлежит habrahabr
func IsHTTPSHabrUrl(url string) bool {
	habrPool := []string{"www.habrahabr.ru", "habrahabr.ru"}

	for _, v := range habrPool {
		if strings.Contains(url, v) {
			return true
		}
	}
	return false
}
