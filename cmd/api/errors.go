package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
)

func (app *Application) reportServerError(r *http.Request, err error) {
	var (
		message = err.Error()
		method  = r.Method
		url     = r.URL.String()
		trace   = string(debug.Stack())
	)

	requestAttrs := slog.Group("request", "method", method, "url", url)
	app.logger.Error(message, requestAttrs, "trace", trace)
}

func (app *Application) errorMessage(w http.ResponseWriter, r *http.Request, status int, message string, headers http.Header) {
	message = strings.ToUpper(message[:1]) + message[1:]

	err := response.JSONWithHeaders(w, response.JSONResponseParameters{
		Success: false,
		Status:  status,
		Msg:     message,
		ErrMsg:  message,
	}, headers)
	if err != nil {
		app.reportServerError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *Application) invalidCredentials(w http.ResponseWriter, r *http.Request) {
	message := "invalid credentials"
	app.errorMessage(w, r, http.StatusBadRequest, message, nil)
}

func (app *Application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	// log error to std out
	app.reportServerError(r, err)

	message := "The server encountered a problem and could not process your request"
	app.errorMessage(w, r, http.StatusInternalServerError, message, nil)
}

func (app *Application) notFound(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"
	app.errorMessage(w, r, http.StatusNotFound, message, nil)
}

func (app *Application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource", r.Method)
	app.errorMessage(w, r, http.StatusMethodNotAllowed, message, nil)
}

func (app *Application) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	app.errorMessage(w, r, http.StatusBadRequest, err.Error(), nil)
}

func (app *Application) basicAuthenticationRequired(w http.ResponseWriter, r *http.Request) {
	headers := make(http.Header)
	headers.Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

	message := "You must be authenticated to access this resource"
	app.errorMessage(w, r, http.StatusUnauthorized, message, headers)
}

func (app *Application) unauthorized(w http.ResponseWriter, r *http.Request) {
	message := "You must be authenticated to access this resource"
	app.errorMessage(w, r, http.StatusUnauthorized, message, nil)
}
