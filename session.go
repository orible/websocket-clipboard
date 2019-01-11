package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
)

var gSessionStore = sessions.NewFilesystemStore("store/", []byte("mA9WrDP5EaL3DnvCnAaHzpjTm6n5RLrqMcK3DPtZ"))

type SSession struct {
	m_session *sessions.Session //Session store from "github.com/gorilla/sessions"
}

const (
	SESSION_LAST_TIME     = "ltime"
	SESSION_REQUEST_COUNT = "rcount"
	SESSION_UUID          = "uuid"
)

func Ct_session(ref *sessions.Session) *SSession {
	e := new(SSession)
	e.m_session = ref
	return e
}
func (s *SSession) GetUUID() int {
	if intUUID, ok := s.m_session.Values[SESSION_UUID].(int); ok {
		return intUUID
	}
	panic(0)
}
func (s *SSession) GetLastRequestTime() int {
	if intIP, ok := s.m_session.Values[SESSION_LAST_TIME].(int); ok {
		return intIP
	}
	panic(0)
}
func (s *SSession) RequestCount() int {
	if intCount, ok := s.m_session.Values[SESSION_REQUEST_COUNT].(int); ok {
		return intCount
	}
	panic(0)
}

//GetTimeUnixMilliseconds ...
//time since unix epoch in milliseconds
func GetTimeUnixMilliseconds() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func HandlerMiddlewareSession(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var l = new(bytes.Buffer)

		fmt.Fprintf(l, "Reading session store...[")
		session, serr := gSessionStore.Get(r, "session-key")
		if serr != nil {
			fmt.Fprintf(l, "error] failed to read session storage!\n")
			http.Error(w, serr.Error(), http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to read session storage!\n")
			fmt.Println("[session] Failed to read session storage\n", serr)
			return
		}
		fmt.Printf("[session] Reading session store...")
		qflagCreateNew := r.URL.Query().Get("session_cnew")
		fmt.Fprintf(l, "OK]\n")
		if session.IsNew {
			fmt.Fprintf(l, "Created new session cookie\n")
			fmt.Printf("created new session\n")
			session.Values[SESSION_REQUEST_COUNT] = 0
			session.Values[SESSION_LAST_TIME] = GetTimeUnixMilliseconds()
			session.Values[SESSION_UUID] = -1
		} else {
			fmt.Printf("session exists\n")
		}

		if qflagCreateNew == "true" && !session.IsNew {
			fmt.Fprintf(l, "session_cnew: true\n\tdestroyed session\n")
			session.Options.MaxAge = -1
		}

		rw := Ct_session(session)
		session.Values[SESSION_REQUEST_COUNT] = rw.RequestCount() + 1
		fmt.Fprintf(l, "Request count: [%d]\n", Ct_session(session).RequestCount())
		fmt.Fprintf(l, "Socket UUID: [%d]\n", Ct_session(session).GetUUID())

		/* Save cookie state, write buffer */
		err := session.Save(r, w)
		if err != nil {
			fmt.Fprintf(l, "save error: %s\n", err)
			log.Fatal(err)
		}

		ctx := context.WithValue(r.Context(), "sessionstore", session)
		h.ServeHTTP(w, r.WithContext(ctx))

		if r.URL.Path == "/debug" {
			w.Write([]byte("<pre>"))
			l.WriteTo(w)
			w.Write([]byte("</pre>"))
		}
	})
}
func MiddlwareSessionInit(domain string, httpOnly bool) {
	gSessionStore.Options = &sessions.Options{
		Domain: domain,
		//Domain: "localhost",
		Path:   "/",
		MaxAge: 3600 * 8, // 8 hours
		//HttpOnly: true,
		HttpOnly: false,
	}
}
