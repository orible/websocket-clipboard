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
	"strings"
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
		fmt.Printf("[template] Layout: %s\n", layouts)
		files := append(includes, layout)
		gTemplates[filepath.Base(layout)] = template.Must(template.ParseFiles(files...))
		fmt.Printf("[template] Built: %s\n", filepath.Base(layout))
	}
	return true
}

func responseRequestHandler(w http.ResponseWriter, r *http.Request) {}
func responseRequestDebug(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<p></p>"))
}

type SPageInfo struct {
	PageTitle string
	NavTitle  string
	Theme     string
}

func responseRequestIndex(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte("<p></p>"))
	//cookie1 := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
	//http.SetCookie(w, cookie1)

	queryReload := r.URL.Query().Get("reload")
	if queryReload == "true" {
		fmt.Printf("Reloading pages\n")
		templateBuild()
	}
	tmpl, ok := gTemplates["index.html"]
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var info SPageInfo
	info.PageTitle = "go-clipboard"
	info.NavTitle = "go-clipboard"
	err := tmpl.Execute(w, &info)
	if err != nil {
		fmt.Print("responseRequestIndex -> failed to generate page\n", err)
	}
}

var flagAddr = flag.String("addr", "localhost:8080", "http service address")
var flagPairTimeout = flag.Int("ptimeout", 0, "default pair timeout")
var flagCert = flag.String("cert", "server", "certificate dir and name")

var flagInsecure = flag.Bool("https", true, "disable https security")
var flagCookieDomain = flag.String("cookiedomain", "localhost", "session cookie domain")

//var flagCookie flag.String("cookiehttp")

func main() {
	fmt.Printf("[init] go-websocket 0.1\n[init] server starting...\n")
	flag.Parse()

	fmt.Printf("[template] building templates\n")
	templateBuild()

	if *flagCookieDomain != "localhost" {
		fmt.Printf("[init] cookie domain: %s\n", *flagCookieDomain)
	}

	slice := *flagAddr
	MiddlwareSessionInit(slice[:strings.IndexByte(slice, ':')], false)

	fmt.Printf("[init] Certificate dir: %s\n", *flagCert)
	fmt.Printf("[init] Starting router\n")
	hub := NewRouter()
	go hub.run()

	fmt.Printf("[init] Init HTTP server\n")
	rmux := mux.NewRouter()
	subrouter := rmux.PathPrefix("/").Subrouter()
	rmux.PathPrefix("/asset/").Handler(
		http.StripPrefix("/asset/",
			http.FileServer(http.Dir("public/web/"))))

	//fix for serviceworker origin and scope
	rmux.PathPrefix("/").Handler(
		http.FileServer(http.Dir("public/web/chrome")))

	subrouter.HandleFunc("/api", responseRequestHandler).Methods("POST")
	subrouter.HandleFunc("/webtest", responseRequestIndex).Methods("GET")
	subrouter.HandleFunc("/debug", responseRequestDebug).Methods("GET")
	subrouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		startClient(hub, w, r)
	})
	subrouter.Use(HandlerMiddlewareSession)

	fmt.Printf("[init] Starting HTTP server on %s\n", *flagAddr)
	srv := &http.Server{
		Handler: rmux,
		Addr:    *flagAddr,
		// Set timeouts
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	//go router()
	go func() {
		if *flagInsecure == true {
			if err := srv.ListenAndServeTLS(*flagCert+".crt", *flagCert+".key"); err != nil {
				log.Println(err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil {
				log.Println(err)
			}
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Printf("[init] Server stopped\n")
	fmt.Printf("[init] Server shutdown\n")
}
