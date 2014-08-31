package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const HOST = "localhost:8080"
const PORT = ":8080"
const REGISTER_PATH = "/r/"
const LOOKUP_PATH = "/l/"
const INDEX = `Please enter the link you'd like to shorten into the input box and submit.<br><br><form name="shortener" action="http://` + HOST + REGISTER_PATH + `" method="post"><input name="long" size=40><input type="submit" value="Shorten!"></form>
`

type urlMap struct {
	urls map[string]string
	sync.Mutex
	unsaved []string
}

var um = new(urlMap)

func init() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	um.urls = make(map[string]string)
	um.unsaved = make([]string, 0)
}

func main() {
	last, err := um.loadHistory()
	if err != nil {
		log.Fatal(err)
	}
	tokenChan := make(chan string, 1024)
	go makeTokens(tokenChan, last)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			um.backup()
		}
	}()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>"+INDEX+"</html>") })
	http.HandleFunc(REGISTER_PATH, makeRegisterLink(tokenChan))
	http.HandleFunc(LOOKUP_PATH, redirect)
	http.ListenAndServe(PORT, nil)
}

func makeRegisterLink(tokenChan <-chan string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		long, err := url.Parse(r.FormValue("long"))
		if err != nil || strings.TrimSpace(r.FormValue("long")) == "" {
			fmt.Fprintf(w, "<html>Whoops! We could not parse %v as a valid URL. Want to try again?<br>"+INDEX+"</html>", long)
			return
		}
		if long.Scheme == "" {
			long.Scheme = "http"
		}
		token := <-tokenChan
		um.urls[token] = long.String()
		um.unsaved = append(um.unsaved, token)
		short := &url.URL{"http", "", nil, HOST, LOOKUP_PATH + token, "", ""}
		fmt.Fprintf(w, "<html>Your shortened URL is <a href=\"%v\">%v</a></html>", short.String(), short.String())
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	short := r.URL.Path[len(LOOKUP_PATH):]
	var long string
	var ok bool
	if long, ok = um.urls[short]; !ok {
		fmt.Fprint(w, "<html>We don't know about that shortened URL yet! <br><br>"+INDEX+"<html>")
		return
	}
	http.Redirect(w, r, long, 302)
}

func makeTokens(tokenChan chan<- string, start int64) error {
	for {
		start++
		tokenChan <- strconv.FormatInt(start, 32)
	}
}
