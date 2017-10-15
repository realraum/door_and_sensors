// (c) Bernhard Tittelbach, 2016
package main

import (
	"net/http"

	"github.com/codegangsta/martini"
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

func goRunMartini() {
	m := martini.Classic()
	m.Get("/", webServeSpaceAPI)
	m.Get("/status.json", webServeSpaceAPI)
	m.Get("/spaceapi.json", webServeSpaceAPI)
	m.RunOnAddr(EnvironOrDefault("SPACEAPI_HTTP_INTERFACE", DEFAULT_SPACEAPI_HTTP_INTERFACE))
}
