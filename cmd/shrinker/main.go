package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/abdullah2993/lshort"
	"github.com/dgraph-io/badger"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

func main() {

	addr := flag.String("addr", "localhost:8080", "Address to listen on")
	dbPath := flag.String("dbpath", "temp.d", "Path to database filr/folder")
	dbType := flag.String("db", "bolt", "Which Database to use, bolt or badger")
	csrfSecret := flag.String("csrf", "", "CSRF secret to use")
	csrfSecure := flag.Bool("csrf.cookies", false, "Use csrf in cookies")

	flag.Parse()

	var db lshort.LinkShortner
	var err error

	switch *dbType {
	case "bolt":
		db, err = lshort.NewLinkShortnerBolt(*dbPath, nil)
		if err != nil {
			log.Fatalf("unable to create shrinker: %v", err)
		}
	case "badger":
		db, err = lshort.NewLinkShortnerBadger(*dbPath, badger.DefaultOptions)
		if err != nil {
			log.Fatalf("unable to create shrinker: %v", err)
		}
	default:
		log.Fatalln("invalid database type")
	}

	h, err := NewLinkShrinkHandler(db, []byte(*csrfSecret), csrf.Secure(*csrfSecure))
	if err != nil {
		log.Fatalf("unable to create handler: %v", err)
	}

	log.Fatal(http.ListenAndServe(*addr, h))
}

type linkShrinkHandler struct {
	shrinker  lshort.LinkShortner
	indexTmpl *template.Template
}

func (l *linkShrinkHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	l.indexTmpl.Execute(w, map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	})
}

func (l *linkShrinkHandler) handleShrink(w http.ResponseWriter, r *http.Request) {
	link := r.FormValue("url")
	fields := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	}
	defer l.indexTmpl.Execute(w, fields)
	if link == "" {
		fields["error"] = "unable to shrink url"
		return
	}

	key, err := l.shrinker.Shrink(link)
	if err != nil {
		fields["error"] = "unable to shrink url"
		return
	}

	var u url.URL
	if r.URL.IsAbs() {
		u.Scheme = r.URL.Scheme
		u.Host = r.URL.Host
	} else {
		u.Scheme = "http"
		if r.TLS != nil {
			u.Scheme = "https"
		}
		u.Host = r.Host
	}
	u.Path = key

	fields["url"] = u.String() + "ddddddddddddd"
}

func (l *linkShrinkHandler) handleExpand(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["id"]
	if key == "" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	url, err := l.shrinker.Expand(key)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, url, http.StatusPermanentRedirect)
}

func NewLinkShrinkHandler(shrinker lshort.LinkShortner, csrfSecret []byte, csrfOpts ...csrf.Option) (http.Handler, error) {
	l := &linkShrinkHandler{shrinker: shrinker}
	r := mux.NewRouter()
	r.HandleFunc("/", l.handleIndex).Methods("GET")
	r.HandleFunc("/", l.handleShrink).Methods("POST")
	r.HandleFunc("/{id:[rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz]+}", l.handleExpand).Methods("GET")
	var h http.Handler = r
	if csrfSecret != nil && len(csrfSecret) > 0 {
		h = csrf.Protect(csrfSecret, csrfOpts...)(h)
	}
	tmpl, err := template.ParseFiles("templates/index.gohtml")
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %v", err)
	}
	l.indexTmpl = tmpl
	return h, nil
}
