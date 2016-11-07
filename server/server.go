package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"scout/data"

	_ "github.com/go-sql-driver/mysql" // the init function registers a sql driver
)

var (
	// HandlerLog is used by handlers to register information not displayed to the client
	HandlerLog *log.Logger

	errorHandlers map[int]ErrorHandler
	server        *http.Server
	errorLog      *log.Logger // used for internal logging
	cache         map[string]cacheObject
	db            *sql.DB
)

func init() {
	var err error // prevent shadowing of the db variable on the next line with :=
	db, err = sql.Open("mysql", "scout@/scouting2016")
	if err != nil {
		panic("unable to open database: " + err.Error())
	}

	server = &http.Server{
		Addr:    ":80", // 80 is the standard http port
		Handler: http.NewServeMux(),
	}
	errorLog = log.New(os.Stderr, "Server: ", log.Ldate|log.Ltime|log.Lshortfile)
	HandlerLog = log.New(os.Stderr, "Handler: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorHandlers = map[int]ErrorHandler{}
	cache = map[string]cacheObject{}
}

// Close frees resources used by the server
func Close() error {
	return db.Close()
}

// ListenAndServe waits on the server's address and hands out request to handlers
func ListenAndServe() error {
	return server.ListenAndServe()
}

// ServeErrorCode has the server respond to an error using the designated
// ErrorHandler.  If no ErrorHandler is available, returns false
func ServeErrorCode(code int, writer http.ResponseWriter, request *http.Request) {
	handler, found := errorHandlers[code]
	if found {
		writer.WriteHeader(code)
		handler.ServeErrorHTTP(code, writer, request)
	} else {
		defaultErrorHandler(code, writer, request)
	}
}

func serveError(err error, writer http.ResponseWriter, request *http.Request) {
	if err == nil {
		return
	}

	switch err.(type) {
	case *os.PathError, data.HTTPNotFoundError:
		ServeErrorCode(http.StatusNotFound, writer, request)
	case data.HTTPMethodError:
		ServeErrorCode(http.StatusMethodNotAllowed, writer, request)
	default:
		switch err {
		case data.ErrAccess:
			ServeErrorCode(http.StatusUnauthorized, writer, request)
		default:
			errorLog.Printf("Unhandled error to %s (%s)", request.URL.String(), err.Error())
			defaultErrorHandler(http.StatusInternalServerError, writer, request)
		}
	}
}

// HandleError sets the ErrorHandler to be used when the corresponding code
// shows up while serving.
func HandleError(code int, handler ErrorHandler) {
	errorHandlers[code] = handler
}

// Handle sets the handler to be used when the specified path is requested
// from the server.  Errors originating from the UnsafeHandler are automatically
// handled by the Server.  See HandleError
func Handle(pattern string, handler UnsafeHandler) {
	server.Handler.(*http.ServeMux).Handle(pattern, safeHandler{unsafe: handler})
}

// Cache adds data to the server cache with a name that specifies the request path
// that triggers sending the data.  contentType is the value to put in the http
// Content-Type header.  Cached objects must have a handler in their path to be
// used.  Returns false if the name was already taken, preventing the addition
// to the cache.  Returns true on success.
func Cache(name string, data []byte, contentType string) bool {
	_, taken := cache[name]
	if taken {
		return false
	}

	cache[name] = cacheObject{data: data, contentType: contentType}
	return true
}

type cacheObject struct {
	data        []byte
	contentType string
}

// ServeHTTPUnsafe implements the UnsafeHandler interface
func (obj cacheObject) ServeHTTPUnsafe(writer http.ResponseWriter, _ *http.Request) error {
	writer.Header().Set("Content-Type", obj.contentType)
	_, err := writer.Write(obj.data)
	return err
}

// UnsafeHandler defines something that reponds to an http request but may error
// out.  Any errors that are returned are handled by the Server.  See
// Server.AddErrorHandler.  Implements http.Handler
type UnsafeHandler interface {
	ServeHTTPUnsafe(http.ResponseWriter, *http.Request) error
}

// HandlerFunc wraps a function to be used as an UnsafeHandler
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// ServeHTTPUnsafe implements the UnsafeHandler interface; simply calls the
// function used as a HandlerFunc
func (fn HandlerFunc) ServeHTTPUnsafe(writer http.ResponseWriter, request *http.Request) error {
	return fn(writer, request)
}

type safeHandler struct {
	unsafe UnsafeHandler
}

// ServeHTTP takes an http.Request and writes an appropriate response, using
// the server cache and the UnsafeHandler to process the output.
func (handler safeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			errorLog.Println("caught panic in safeHandler: ", r)
			ServeErrorCode(http.StatusInternalServerError, writer, request)
		}
	}()

	if writer == nil || request == nil {
		errorLog.Println("safeHandler passed uninitialized ResponseWriter or Request")
		ServeErrorCode(http.StatusInternalServerError, writer, request)
		return
	}

	var response UnsafeHandler
	response, cached := cache[request.URL.Path]
	if !cached {
		response = handler.unsafe
	}

	if response == nil {
		errorLog.Println("registered handler was unimplemented: " + request.URL.Path)
		ServeErrorCode(http.StatusNotFound, writer, request)
		return
	}

	serveError(response.ServeHTTPUnsafe(writer, request), writer, request)
}

// ErrorHandler takes a specified http error code and writes a response.
type ErrorHandler interface {
	ServeErrorHTTP(int, http.ResponseWriter, *http.Request)
}

// ErrorHandlerFunc allows functions to easily be used as ErrorHandlers.
type ErrorHandlerFunc func(int, http.ResponseWriter, *http.Request)

// ServeErrorHTTP implements ErrorHandler
func (fn ErrorHandlerFunc) ServeErrorHTTP(code int, writer http.ResponseWriter, request *http.Request) {
	fn(code, writer, request)
}

func defaultErrorHandler(code int, writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
	<head>
		<title>Error %d</title>
	</head>
	<body>
		Sorry, an unknown error occured internal to the server (%d).
	</body>
</html>`, code, code)))
}
