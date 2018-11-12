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
	dbPath := flag.String("dbpath", "temp.d", "Path to database file/folder")
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

	fields["url"] = u.String()
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
	tmpl, err := template.New("index").Parse(tepl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %v", err)
	}
	l.indexTmpl = tmpl
	return h, nil
}

var tepl = `
<!doctype html>
<html lang="en">

<head>
    <title>Link Shortner</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.1.2/css/bootstrap.min.css" integrity="sha384-Smlep5jCw/wG7hdkwQ/Z5nLIefveQRIY9nfy6xoR1uRYBtpZgI6339F5dgvm/e9B"
        crossorigin="anonymous">
</head>

<body>
    <nav class="navbar navbar-light bg-light">
        <a class="navbar-brand" href="#">Link Shortner</a>
    </nav>
    <div class="jumbotron jumbotron-fluid">
        <div class="container">
            <h1 class="display-4">Shrink your links</h1>
            {{if .error}}
            <div class="alert alert-warning" role="alert">
                A simple warning alert—check it out!
            </div>
            {{end}}

            <form method="POST">
                <div class="input-group mb-3">
                    {{.csrfField}}
                    <input type="url" name="url" required class="form-control" placeholder="https://google.com">
                    <div class="input-group-append">
                        <input class="btn btn-outline-secondary" type="submit" value="Shrink">
                    </div>
                </div>
            </form>

            {{if .url}}
            <div class="input-group mb-3">
                <div class="input-group-prepend">
                    <span class="input-group-text">Your shrinked link</span>
                </div>
                <input type="text" id="shilink" class="form-control" value="{{.url}}" readonly>
                <div class="input-group-append">
                    <button class="btn btn-outline-secondary" id="copier" data-clipboard-target="#shilink" type="button">Copy</button>
                </div>
            </div>
            {{end}}
        </div>
    </div>
    <!-- <script src="https://code.jquery.com/jquery-3.3.1.slim.min.js" integrity="sha384-q8i/X+965DzO0rT7abK41JStQIAqVgRVzpbzo5smXKp4YfRvH+8abtTE1Pi6jizo"
        crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.3/umd/popper.min.js" integrity="sha384-ZMP7rVo3mIykV+2+9J3UJ46jBk0WLaUAdn689aCwoqbBJiSnjAK/l8WvCWPIPm49"
        crossorigin="anonymous"></script>
    <script src="https://stackpath.bootstrapcdn.com/bootstrap/4.1.2/js/bootstrap.min.js" integrity="sha384-o+RDsa0aLu++PJvFqy8fFScvbHFLtbvScb8AjopnFD+iEQ7wo/CG0xlczd+2O/em"
        crossorigin="anonymous"></script> -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js"></script>
    <script>
    new ClipboardJS('#copier');
    </script>

</body>

</html>
`
