package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
)

type Route struct {
	Prefix string
	Proxy  url.URL
}

type Config struct {
	Listen string            `json:"listen,omitempty"`
	Routes map[string]string `json:"routes,omitempty"`
	routes []Route
}

func readConfig(filename string) (Config, error) {
	config := Config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	routes := []Route{}
	for path := range config.Routes {
		proxy, err := url.Parse(config.Routes[path])
		if err != nil {
			return config, err
		}

		routes = append(routes, Route{
			Prefix: path,
			Proxy:  *proxy,
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		return len(routes[j].Prefix) < len(routes[i].Prefix)
	})

	config.routes = routes

	return config, nil
}

func director(routes []Route) func(req *http.Request) {
	return func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.URL.Scheme = "http"

		for i := range routes {
			if strings.HasPrefix(req.URL.Path, routes[i].Prefix) {
				req.Header.Add("X-Origin-Host", routes[i].Proxy.Host)
				req.URL.Host = routes[i].Proxy.Host
				break
			}
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func main() {
	configFile := flag.String("config", "/etc/stupid-proxy/config.json", "")
	flag.Parse()

	config, err := readConfig(*configFile)
	if err != nil {
		log.Println(err)
		return
	}
	proxy := &httputil.ReverseProxy{Director: director(config.routes)}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		// log.Println(r.Method, r.URL.Path)

		w := &responseWriter{rw, 0}
		proxy.ServeHTTP(w, r)

		if w.statusCode == http.StatusBadGateway || w.statusCode == http.StatusGatewayTimeout {
			rw.Write([]byte(fmt.Sprintf("%d %s", w.statusCode, http.StatusText(w.statusCode))))
		}
	})

	log.Printf("Listen %s...", config.Listen)
	log.Fatal(http.ListenAndServe(config.Listen, nil))
}
