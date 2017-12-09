package react_go_gae_example

import (
	"encoding/json"
	"html/template"
	"log"
	"sync"
	"net/http"

	"github.com/robertkrimen/otto"

	"appengine"
	"appengine/datastore"
)

var templates = template.Must(template.ParseFiles("index.html"))

const jsFile = "server.js"

////////////////////////////////////////////////////////////////////////////////

type State struct {
	Value int `datastore:"value,noindex" json:"value"`
}

func getStateFromDB(ctx appengine.Context) (*State, error) {
	var state State
	key := datastore.NewKey(ctx, "State", "currentState", 0, nil)
	if err := datastore.Get(ctx, key, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

////////////////////////////////////////////////////////////////////////////////

func compileRenderJS(vm *otto.Otto) (otto.Value, error) {
	log.Printf("Compiling %s", jsFile)
	var v otto.Value
	script, err := vm.Compile(jsFile, nil)
	if err != nil { return v, err }
	v, err = vm.Run(script)
	if err != nil { return v, err }
	v, err = vm.Get("server")
	if err != nil { return v, err }
	v, err = v.Object().Get("render")
	if err != nil { return v, err }
	log.Printf("Compiled %s", jsFile)
	return v, nil
}

var callRenderJS = func(stateJSON string) (otto.Value, error) {
	vm := otto.New()
	renderJS, err := compileRenderJS(vm)
	if err != nil { return otto.NullValue(), err }
	return renderJS.Call(otto.NullValue(), stateJSON)
}

func render(state *State) (string, string, error) {
	stateJSON, err := json.Marshal(state)
	if err != nil { return "", "", err }
	var renderResult otto.Value
	renderResult, err = callRenderJS(string(stateJSON))
	if err != nil {
		log.Printf("Error rendering. To get more information, run:\n")
		log.Printf("  ./renderComponent '%s'\n", string(stateJSON))
		return "", "", err
	}
	var renderedHTML, renderedState otto.Value
	renderedHTML, err = renderResult.Object().Get("html")
	if err != nil { return "", "", err }
	renderedState, err = renderResult.Object().Get("state")
	if err != nil { return "", "", err }
	return renderedHTML.String(), renderedState.String(), nil
}

////////////////////////////////////////////////////////////////////////////////

func handleInitializeDB(w http.ResponseWriter, r *http.Request) error {
	initialState := State{
		Value: 42,
	}
	ctx := appengine.NewContext(r)
	key := datastore.NewKey(ctx, "State", "currentState", 0, nil)
	_, err := datastore.Put(ctx, key, &initialState)
	return err
}

func handleState(w http.ResponseWriter, r *http.Request) error {
	state, err := getStateFromDB(appengine.NewContext(r))
	if err != nil { return err }
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(state)
}

func handleIndex(w http.ResponseWriter, r *http.Request) error {
	type IndexPage struct {
		HTML  template.HTML
		State template.JS
	}
	if len(r.Header["X-Devserver"]) > 0 {
		// When accessing through the dev server, 
		// don't pre-render anything
		return templates.ExecuteTemplate(w, "index.html", IndexPage {
			HTML:  "",
			State: template.JS("null"),
		})
	} else {
		// Prerender
		state, err := getStateFromDB(appengine.NewContext(r))
		if err != nil { return err }
		var renderedHTML, renderedState string
		renderedHTML, renderedState, err = render(state)
		if err != nil { return err }
		return templates.ExecuteTemplate(w, "index.html", IndexPage {
			HTML:  template.HTML(renderedHTML),
			State: template.JS(renderedState),
		})
	}
}

////////////////////////////////////////////////////////////////////////////////

type handler func(http.ResponseWriter, *http.Request) error

func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

////////////////////////////////////////////////////////////////////////////////

func init() {
	if !appengine.IsDevAppServer() {
		// Compile server.js only once in production
		var renderJS otto.Value
		var once sync.Once
		callRenderJS = func(stateJSON string) (otto.Value, error) {
			once.Do(func () { 
				vm := otto.New()
				var err error
				renderJS, err = compileRenderJS(vm)
				if err != nil { log.Fatal(err) }
			})
			return renderJS.Call(otto.NullValue(), stateJSON)
		}
	}

	http.Handle("/api/state", handler(handleState))
	http.Handle("/api/initialize-db", handler(handleInitializeDB))
	http.Handle("/", handler(handleIndex))
}
