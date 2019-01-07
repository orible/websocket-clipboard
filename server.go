package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

const (
	CLIPBOARD_MAX_LEN    = 1024
	CLIPBOARD_RATE_LIMIT = 100
)

type SReadResponse struct {
	ChanResponse chan *SThreadMessage
	Context      context.Context
	Ptr          interface{}
	Type         int
}
type SThreadMessage struct {
	Type int
	Ptr  interface{}
}

const (
	CLIENT_CLIPBOARD_LIST     = 10
	CLIENT_GENERATE_PAIR_UUID = 11
)

type SServerTable struct {
}
type SNetworkPacketJson struct {
	Type      int
	Time      int64
	Callback  int
	Transport interface{}
}

type SNetworkClipboardItem struct {
	Type   int
	Spec   int
	Key    int
	Buffer string
}

type SNetworkGroup struct {
	Data []interface{}
}

var chanRouterMessageIn = make(chan *SThreadMessage, 1024)
var chanRouterMessageOut = make(chan bool)
var queueItems []*SClipboardItem

type SClipboardItem struct {
	msg        string
	sender     int
	userAuth   int
	time_enque time.Time
}

func PushClipboardHistory(url string) {
	queueItems = append(queueItems, &SClipboardItem{ //store task
		msg:        "0",
		sender:     0,
		time_enque: time.Now(),
	})
}

func router() int {
	var msg *SThreadMessage

	dispatchLoop := true
	for dispatchLoop {
		select {
		case msg = <-chanRouterMessageIn:
			switch msg.Type {
			case 0:
				break
			}
			break
		}
	}
	return 0
}
func SendRouterEx(m *SThreadMessage) bool {
	select {
	case chanRouterMessageIn <- m:
	default:
		fmt.Println("no message sent")
		return false
	}
	return true
}

var gTemplates map[string]*template.Template

func templateBuild() bool {
	gTemplates = make(map[string]*template.Template)
	layouts, err := filepath.Glob(filepath.Join("public/pages/", "page/*.html"))
	if err != nil {
		log.Fatal(err)
	}
	includes, err := filepath.Glob(filepath.Join("public/pages/", "base/*.html"))
	if err != nil {
		log.Fatal(err)
	}
	//Generate our templates map from our layouts/ and includes/ directories
	for _, layout := range layouts {
		fmt.Printf("Layout: %s\n", layouts)
		files := append(includes, layout)
		gTemplates[filepath.Base(layout)] = template.Must(template.ParseFiles(files...))
		fmt.Printf("Built: %s\n", filepath.Base(layout))
	}
	return true
}

func responseRequestHandler(w http.ResponseWriter, r *http.Request) {}
func responseRequestDebug(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<p></p>"))
}
func responseRequestIndex(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte("<p></p>"))
	cookie1 := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
	http.SetCookie(w, cookie1)

	queryReload := r.URL.Query().Get("reload")
	if queryReload == "true" {
		fmt.Printf("Reloading pages\n")
		templateBuild()
	}
	tmpl, ok := gTemplates["index.html"]
	if !ok {
		return
	}

	tmpl.Execute(w, nil)
}

var flagAddr = flag.String("addr", "localhost:8080", "http service address")
var flagPairTimeout = flag.Int("ptimeout", 0, "default pair timeout")

func main() {
	fmt.Printf("server started\n")
	flag.Parse()

	templateBuild()
	MiddlwareSessionInit()

	fmt.Printf("Starting router\n")
	hub := NewRouter()
	go hub.run()

	fmt.Printf("Init HTTP server\n")
	rmux := mux.NewRouter()
	subrouter := rmux.PathPrefix("/").Subrouter()
	rmux.PathPrefix("/asset/").Handler(
		http.StripPrefix("/asset/",
			http.FileServer(http.Dir("public/web/"))))
	subrouter.HandleFunc("/api", responseRequestHandler).Methods("POST")
	subrouter.HandleFunc("/webtest", responseRequestIndex).Methods("GET")
	subrouter.HandleFunc("/debug", responseRequestDebug).Methods("GET")
	subrouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		startClient(hub, w, r)
	})
	subrouter.Use(HandlerMiddlewareSession)

	fmt.Printf("Starting HTTP server on %s\n", *flagAddr)
	srv := &http.Server{
		Handler: rmux,
		Addr:    *flagAddr,
		// Set timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	//go router()
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Printf("Server stopped\n")
	fmt.Printf("Server shutdown\n")
}
