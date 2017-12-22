package HTTPMUX

/*
the mutex struct
*/
import (
	"errors"
	"net/http"

	"github.com/use-go/websocket-streamserver/logger"
)

//one kind service one catalog Mutex
var portsServe map[string]*http.ServeMux

func init() {
	portsServe = make(map[string]*http.ServeMux)
}

// AddRoute To handle different data route for spec path.
func AddRoute(port, route string, handler func(w http.ResponseWriter, req *http.Request)) {
	mux, exist := portsServe[port]
	if false == exist {
		portsServe[port] = http.NewServeMux()
		mux = portsServe[port]
	}
	mux.HandleFunc(route, handler)
}

// GetPortServe retrive the *http.ServeMux for the spec port
func GetPortServe(port string) (serveMux *http.ServeMux, err error) {

	if len(port) < 1 {
		logger.LOGE("port param error in HttpMux")
		return nil, errors.New("port param Not Found")
	}

	mux, exist := portsServe[port]
	if exist {
		return mux, nil
	}

	return nil, errors.New("serveMux Not Found")
}

// // Start differnt http Service binded to ports
// func Start() {
// 	//for dash.js test
// 	//portsServe[":8080"].Handle("/dash_js/", http.StripPrefix("/dash_js/", http.FileServer(http.Dir("../test-websocket/"))))
// 	//for websocket-media-stream.ts test
// 	//portsServe[":8080"].Handle("/player/", http.StripPrefix("/player/", http.FileServer(http.Dir("../test-websocket/"))))
// 	for k, v := range portsServe {
// 		go func(addr string, handler http.Handler) {
// 			err := http.ListenAndServe(addr, handler)
// 			if err != nil {
// 				logger.LOGE(err.Error())
// 			}
// 		}(k, v)
// 	}
// }
