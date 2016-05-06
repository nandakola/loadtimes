// webapp: a standalone example Negroni / Gorilla based webapp.
//
// This example demonstrates basic usage of Appdash in a Negroni / Gorilla
// based web application. The entire application is ran locally (i.e. on the
// same server) -- even the Appdash web UI.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"sourcegraph.com/sourcegraph/appdash"
	"sourcegraph.com/sourcegraph/appdash/httptrace"
	"sourcegraph.com/sourcegraph/appdash/traceapp"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// Used to  store the CtxSpanID in a request's context (see gorilla/context docs
// for more information).
const CtxSpanID = 0

// ClientCallInfo to fetch the values
type ClientCallInfo struct {
	Name          string
	EntryType     string
	StartTime     float64
	EndTime       float64
	InitiatorType string
}

// NewServerEvent returns an event which records various aspects of an
// HTTP response. It takes an HTTP request, not response, as input
// because the information it records is derived from the request, and
// HTTP handlers don't have access to the response struct (only
// http.ResponseWriter, which requires wrapping or buffering to
// introspect).
//
// The returned value is incomplete and should have its Response and
// ServerRecv/ServerSend values set before being logged.

// RequestInfo describes an HTTP request.
type RequestInfo struct {
	Method        string
	URI           string
	Proto         string
	Headers       map[string]string
	Host          string
	RemoteAddr    string
	ContentLength int64
}

// ResponseInfo describes an HTTP response.
type ResponseInfo struct {
	Headers       map[string]string
	ContentLength int64
	StatusCode    int
}

// NewServerEvent describes event to be stored.
func NewServerEvent() *ServerEvent {
	return &ServerEvent{}
}

// ServerEvent records an HTTP server request handling event.
type ServerEvent struct {
	Request    RequestInfo  `trace:"Server.Request"`
	Response   ResponseInfo `trace:"Server.Response"`
	Route      string       `trace:"Server.Route"`
	User       string       `trace:"Server.User"`
	ServerRecv time.Time    `trace:"Server.Recv"`
	ServerSend time.Time    `trace:"Server.Send"`
}

// Schema returns the constant "HTTPServer".
func (ServerEvent) Schema() string { return "HTTPServer" }

// Important implements the appdash ImportantEvent.
func (ServerEvent) Important() []string {
	return []string{"Server.Response.StatusCode"}
}

// Start implements the appdash TimespanEvent interface.
func (e ServerEvent) Start() time.Time { return e.ServerRecv }

// End implements the appdash TimespanEvent interface.
func (e ServerEvent) End() time.Time { return e.ServerSend }

// We want to create HTTP clients recording to this collector inside our Home
// handler below, so we use a global variable (for simplicity sake) to store
// the collector in use. We could also use gorilla/context to store it.
var collector appdash.Collector

func main() {
	// Create a recent in-memory store, evicting data after 20s.
	//
	// The store defines where information about traces (i.e. spans and
	// annotations) will be stored during the lifetime of the application. This
	// application uses a MemoryStore store wrapped by a RecentStore with an
	// eviction time of 20s (i.e. all data after 20s is deleted from memory).
	memStore := appdash.NewMemoryStore()
	store := &appdash.RecentStore{
		MinEvictAge: 300 * time.Second,
		DeleteStore: memStore,
	}

	// Start the Appdash web UI on port 8700.
	//
	// This is the actual Appdash web UI -- usable as a Go package itself, We
	// embed it directly into our application such that visiting the web server
	// on HTTP port 8700 will bring us to the web UI, displaying information
	// about this specific web-server (another alternative would be to connect
	// to a centralized Appdash collection server).
	tapp := traceapp.New(nil)
	tapp.Store = store
	tapp.Queryer = memStore
	log.Println("Appdash web UI running on HTTP :8700")
	go func() {
		log.Fatal(http.ListenAndServe(":8700", tapp))
	}()

	// We will use a local collector (as we are running the Appdash web UI
	// embedded within our app).
	//
	// A collector is responsible for collecting the information about traces
	// (i.e. spans and annotations) and placing them into a store. In this app
	// we use a local collector (we could also use a remote collector, sending
	// the information to a remote Appdash collection server).
	collector = appdash.NewLocalCollector(store)

	// Create the appdash/httptrace middleware.
	//
	// Here we initialize the appdash/httptrace middleware. It is a Negroni
	// compliant HTTP middleware that will generate HTTP events for Appdash to
	// display. We could also instruct Appdash with events manually, if we
	// wanted to.
	tracemw := httptrace.Middleware(collector, &httptrace.MiddlewareConfig{
		RouteName: func(r *http.Request) string { return r.URL.Path },
		SetContextSpan: func(r *http.Request, spanID appdash.SpanID) {
			context.Set(r, CtxSpanID, spanID)
		},
	})

	// Setup our router (for information, see the gorilla/mux docs):
	router := mux.NewRouter()
	router.HandleFunc("/", Home)
	router.HandleFunc("/endpoint", Endpoint)

	// Setup Negroni for our app (for information, see the negroni docs):
	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(tracemw)) // Register appdash's HTTP middleware.
	n.UseHandler(router)
	n.Run(":8699")
}

// Home is the homepage handler for our app.
func Home(w http.ResponseWriter, r *http.Request) { // Grab the span from the gorilla context. We do this so that we can grab
	// the span.Trace ID and link directly to the trace on the web-page itself!
	span := context.Get(r, CtxSpanID).(appdash.SpanID)

	// We're going to make some API requests, so we create a HTTP client using
	// a appdash/httptrace transport here. The transport will inform Appdash of
	// the HTTP events occuring.
	httpClient := &http.Client{
		Transport: &httptrace.Transport{
			Recorder: appdash.NewRecorder(span, collector),
			SetName:  true,
		},
	}

	// Make three API requests using our HTTP client.
	for i := 0; i < 3; i++ {
		resp, err := httpClient.Get("/endpoint")
		if err != nil {
			log.Println("/endpoint:", err)
			continue
		}
		resp.Body.Close()
	}

	// Render the page.
	fmt.Fprintf(w, `<!DOCTYPE html>
										<html>
										<head>

										  <!-- Basic Page Needs
										  –––––––––––––––––––––––––––––––––––––––––––––––––– -->
										  <meta charset="utf-8">
										  <title>Test load</title>
										  <meta name="description" content="">
										  <meta name="author" content="">

										  <!-- Mobile Specific Metas
										  –––––––––––––––––––––––––––––––––––––––––––––––––– -->
										<meta name="viewport" content="width=device-width, initial-scale=1">
										<link href="https://bedbathandbeyond.qa.nrdecor.com/js/lib/jqwidgets/styles/jqx.base.css" rel="stylesheet" type="text/css" id="jqw-styles">
										<link href="https://bedbathandbeyond.qa.nrdecor.com/js/lib/jqwidgets/styles/jqx.office.css" rel="stylesheet" type="text/css" id="jqw-styles">
										<link href="https://bedbathandbeyond.qa.nrdecor.com/js/lib/justified-gallery/styles/justifiedGallery.css" type="text/css" rel="stylesheet" >


										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/enterprise/default/css/styles.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/enterprise/default/css/widgets.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/bbb2/common/css/common.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/bbb2/common/css/jquery-ui-1.9.2.custom.min.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/bbb2/web/css/web.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/bbb2/web/css/fonts.css" media="all" />
										<link rel="stylesheet" type="text/css" href="https://bedbathandbeyond.qa.nrdecor.com/skin/frontend/enterprise/default/css/print.css" media="print" />
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/prototype/prototype.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/lib/ccard.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/prototype/validation.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/scriptaculous/builder.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/scriptaculous/effects.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/scriptaculous/dragdrop.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/scriptaculous/controls.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/scriptaculous/slider.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/varien/js.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/varien/form.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/varien/menu.js"></script>
										<script type="text/javascript" src="https://bedbathandbeyond.qa.nrdecor.com/js/mage/translate.js"></script>
										<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.0.0-alpha1/jquery.min.js"></script>

										  <!-- Scripts
										  –––––––––––––––––––––––––––––––––––––––––––––––––– -->

										<script type="text/javascript">
										     $(document).ready(function () {
										    console.log(window.performance)//.getEntries())
										    var arr = window.performance.getEntriesByType("resource")
										    jsonObj = [];
										     console.log(jsonObj);
										       $.each( arr, function( i, val ) {
										         var name = val.name;
										         var entryType = val.entryType;
										         var startTime = val.fetchStart;
										         var endTime = val.duration;
										         var initiatorType = val.initiatorType;

										         item = {}
										         item ["name"] = name;
										         item ["entryType"] = entryType;
										         item ["startTime"] = startTime;
										         item ["endTime"] = endTime;
										         item ["initiatorType"] = initiatorType;

										         jsonObj.push(item);
										        });
										        jsonString = JSON.stringify(jsonObj);
										        console.log(jsonString);
										        $.ajax({
										            type: "POST",
										            url: "http://192.168.70.1:8699/endpoint",
										            data: jsonString
										            // success: success,
										            // dataType: dataType
										          });
										     });
										   </script>
										</head>
										<body>
										<!-- Primary Page Layout
										  –––––––––––––––––––––––––––––––––––––––––––––––––– -->
										 <img id="image0"  src="http://flex.madebymufffin.com/images/inacup_donut.jpg" alt="Smiley face" height="42" width="42">
										<!-- End Document
										  –––––––––––––––––––––––––––––––––––––––––––––––––– -->
										</body>
										</html>
									`)
	fmt.Fprintf(w, `<p><a href="http://localhost:8700/traces" target="_">View all traces</a></p>`)
}

// Endpoint is an example API endpoint. In a real application, the backend of
// your service would be contacting several external and internal API endpoints
// which may be the bottleneck of your application.
//
// For example purposes we just sleep for 200ms before responding to simulate a
// slow API endpoint as the bottleneck of your application.
func Endpoint(w http.ResponseWriter, r *http.Request) {
	traceID := appdash.NewRootSpanID()
	decoder := json.NewDecoder(r.Body)
	var t []ClientCallInfo
	err := decoder.Decode(&t)
	if err != nil {
		log.Println("erooror", err)
	}
	startTime := time.Now()
	for i := 0; i < len(t); i++ {
		e := NewServerEvent()
		e.ServerRecv = startTime
		e.Route = t[i].InitiatorType
		e.User = "u"
		e.Response = ResponseInfo{
			StatusCode: 200,
			//Headers:    map[string]string{"Span-Id": "0000000000000001/0000000000000002/0000000000000003"},
		}
		e.Request = RequestInfo{
			Method:  "GET",
			Proto:   "HTTP/1.1",
			URI:     t[i].Name,
			Host:    "example.com",
			Headers: map[string]string{"X-Req-Header": "a"},
		}
		duration := t[i].EndTime
		c := int64(duration)
		e.ServerSend = time.Unix(0, ((startTime.UnixNano()/1000000)+c)*1000000)
		traceIDto := appdash.NewSpanID(traceID)
		rec := appdash.NewRecorder(traceIDto, collector)
		rec.Name(t[i].Name)
		rec.Event(e)
		rec.Finish()
	}
	//	time.Now() + time.Duration(194.15)*time.Millisecond
	// log.Println("I am inside Endpoint", startTime)
	// log.Println("I am inside Endpoint", endTime)
}
