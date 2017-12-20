package HTTPMUX

/*
the mutex struct
*/
import (
	"net/http"

	"github.com/use-go/websocket-streamserver/logger"
)

var ports map[string]*http.ServeMux

func init() {
	ports = make(map[string]*http.ServeMux)
}

// AddRoute To handle route for spec path.
func AddRoute(port, route string, handler func(w http.ResponseWriter, req *http.Request)) {
	mux, exist := ports[port]
	if false == exist {
		ports[port] = http.NewServeMux()
		mux = ports[port]
	}
	mux.HandleFunc(route, handler)
}

// Start differnt Service binded to ports
func Start() {
	//for dash.js test
	ports[":8080"].Handle("/dash_js/", http.StripPrefix("/dash_js/", http.FileServer(http.Dir("../test-websocket/"))))
	ports[":8080"].Handle("/playease/", http.StripPrefix("/playease/", http.FileServer(http.Dir("../test-websocket/"))))
	for k, v := range ports {
		go func(addr string, handler http.Handler) {
			err := http.ListenAndServe(addr, handler)
			if err != nil {
				logger.LOGE(err.Error())
			}
		}(k, v)
	}
}
