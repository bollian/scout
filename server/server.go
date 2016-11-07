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

// Server provides a safe way by which to run a webserver.  Wraps http.Server
type Server struct {
	server        *http.Server
	errorLog      *log.Logger // used for internal logging
	HandlerLog    *log.Logger // used by handlers
	errorHandlers map[int]ErrorHandler
	cache         map[string]cacheObject
	credStore     *sql.DB
}

// New creates a Server and default initializes its values
func New(addr string) (*Server, error) {
	store, err := sql.Open("mysql", "scout@/scouting2016")
	if err != nil {
		return nil, err
	}

	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: http.NewServeMux(),
		},
		errorLog:      log.New(os.Stderr, "Server: ", log.Ldate|log.Ltime|log.Lshortfile),
		HandlerLog:    log.New(os.Stderr, "Handler: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorHandlers: map[int]ErrorHandler{},
		cache:         map[string]cacheObject{},
		credStore:     store,
	}, nil
}

// Close frees resources used by the server
func (server *Server) Close() error {
	if server == nil {
		return nil
	}
	return server.credStore.Close()
}

// ListenAndServe waits on the server's address and hands out request to handlers
func (server *Server) ListenAndServe() error {
	return server.server.ListenAndServe()
}

// ServeErrorCode has the server respond to an error using the designated
// ErrorHandler.  If no ErrorHandler is available, returns false
func (server *Server) ServeErrorCode(code int, writer http.ResponseWriter, request *http.Request) {
	handler, found := server.errorHandlers[code]
	if found {
		writer.WriteHeader(code)
		handler.ServeErrorHTTP(code, writer, request)
	} else {
		defaultErrorHandler(code, writer, request)
	}
}

func (server *Server) serveError(err error, writer http.ResponseWriter, request *http.Request) {
	if err == nil {
		return
	}

	switch err.(type) {
	case *os.PathError, data.HTTPNotFoundError:
		server.ServeErrorCode(http.StatusNotFound, writer, request)
	case data.HTTPMethodError:
		server.ServeErrorCode(http.StatusMethodNotAllowed, writer, request)
	default:
		switch err {
		case data.ErrAccess:
			server.ServeErrorCode(http.StatusUnauthorized, writer, request)
		default:
			server.errorLog.Printf("Unhandled error to %s (%s)", request.URL.String(), err.Error())
			defaultErrorHandler(http.StatusInternalServerError, writer, request)
		}
	}
}

// HandleError sets the ErrorHandler to be used when the corresponding code
// shows up while serving.
func (server *Server) HandleError(code int, handler ErrorHandler) {
	server.errorHandlers[code] = handler
}

// Handle sets the handler to be used when the specified path is requested
// from the server.  Errors originating from the UnsafeHandler are automatically
// handled by the Server.  See HandleError
func (server *Server) Handle(pattern string, handler UnsafeHandler) {
	switch mux := server.server.Handler.(type) {
	case *http.ServeMux:
		mux.Handle(pattern, &safeHandler{
			server:  server,
			handler: handler,
		})
	}
}

// Cache adds data to the server cache with a name that specifies the request path
// that triggers sending the data.  contentType is the value to put in the http
// Content-Type header.  Cached objects must have a handler in their path to be
// used.  Returns false if the name was already taken, preventing the addition
// to the cache.  Returns true on success.
func (server *Server) Cache(name string, data []byte, contentType string) bool {
	_, taken := server.cache[name]
	if taken {
		return false
	}

	server.cache[name] = cacheObject{
		data:        data,
		contentType: contentType,
	}
	return true
}

type cacheObject struct {
	data        []byte
	contentType string
}

// ServeHTTPUnsafe implements the UnsafeHandler interface
func (obj cacheObject) ServeHTTPUnsafe(server *Server, writer http.ResponseWriter, _ *http.Request) error {
	writer.Header().Set("Content-Type", obj.contentType)
	_, err := writer.Write(obj.data)
	return err
}

// UnsafeHandler defines something that reponds to an http request but may error
// out.  Any errors that are returned are handled by the Server.  See
// Server.AddErrorHandler.
type UnsafeHandler interface {
	ServeHTTPUnsafe(*Server, http.ResponseWriter, *http.Request) error
}

// HandlerFunc wraps a function to be used as an UnsafeHandler
type HandlerFunc func(*Server, http.ResponseWriter, *http.Request) error

// ServeHTTPUnsafe implements the UnsafeHandler interface; simply calls the
// function used as a HandlerFunc
func (fn HandlerFunc) ServeHTTPUnsafe(server *Server, writer http.ResponseWriter, request *http.Request) error {
	return fn(server, writer, request)
}

type safeHandler struct {
	server  *Server
	handler UnsafeHandler
}

// ServeHTTP takes an http.Request and writes an appropriate response, using
// the server cache and the UnsafeHandler to process the output.
func (handler safeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			handler.server.errorLog.Println("caught panic in safeHandler: ", r)
			handler.server.ServeErrorCode(http.StatusInternalServerError, writer, request)
		}
	}()

	if handler.handler == nil {
		handler.server.errorLog.Println("safeHandler wrapping uninitialized handler")
		handler.server.ServeErrorCode(http.StatusNotFound, writer, request)
		return
	}

	if writer == nil || request == nil {
		handler.server.errorLog.Println("safeHandler passed uninitialized ResponseWriter or Request")
		handler.server.ServeErrorCode(http.StatusInternalServerError, writer, request)
		return
	}

	var response UnsafeHandler
	response, cached := handler.server.cache[request.URL.Path]
	if !cached {
		response = handler.handler
	}
	handler.server.serveError(response.ServeHTTPUnsafe(handler.server, writer, request), writer, request)
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
