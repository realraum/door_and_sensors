// (c) Bernhard Tittelbach, 2016
package main

import (
	"net/http"

	"github.com/codegangsta/negroni"
)

func webServeSpaceAPI(w http.ResponseWriter, r *http.Request) {
	defer recover() //don't crash just exit goroutine
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Content-Type", "application/json")
	spaceapijsonRWMutex.RLock()
	w.Write(spaceapijsonbytes)
	spaceapijsonRWMutex.RUnlock()
	return
}

func goRunWebserver() {
	n := negroni.Classic()
	mux := http.NewServeMux()
	mux.HandleFunc("/", webServeSpaceAPI)
	mux.HandleFunc("/status.json", webServeSpaceAPI)
	mux.HandleFunc("/spaceapi.json", webServeSpaceAPI)
	n.UseHandler(mux)
	http.ListenAndServe(EnvironOrDefault("SPACEAPI_HTTP_INTERFACE", DEFAULT_SPACEAPI_HTTP_INTERFACE), n)
}
