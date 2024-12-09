package responses

import (
	"context"
	"encoding/xml"
	"github.com/a-h/templ"
	"log"
	"net/http"

	"github.com/goccy/go-json"
	"google.golang.org/protobuf/proto"
)

// -------------- Structs --------------

// ProtoEncoder Interface for encoding a struct as a protobuf message
// Useful for complex types that don't conform to auto-generated protobuf structs
type ProtoEncoder interface {
	ToProto() proto.Message
}

// -------------- Functions --------------

// SendStruct Send a struct as JSON, XML or Protobuf
func SendStruct[T any](w http.ResponseWriter, r *http.Request, statusCode int, data T) {
	var content = "application/"
	var structBytes []byte
	switch accept := r.Header.Get("Accept"); accept {
	case "application/x-protobuf":
		content += "x-protobuf"
		if pb, ok := any(data).(proto.Message); ok {
			structBytes, _ = proto.Marshal(pb)
		}
		if encoder, ok := any(data).(ProtoEncoder); ok {
			structBytes, _ = proto.Marshal(encoder.ToProto())
		}
	case "application/xml":
		content += "xml"
		structBytes, _ = xml.Marshal(data)
	}
	if structBytes == nil {
		content += "json"
		structBytes, _ = json.Marshal(data)
	}

	w.Header().Set("Content-Type", content)
	w.WriteHeader(statusCode)
	_, err := w.Write(structBytes)
	if err != nil {
		log.Println(err)
		InternalServerError(w, r, "Could not write struct")
	}
}

// DecodeStruct Decode a struct from JSON, XML or Protobuf
func DecodeStruct[T any](r *http.Request, data *T) error {
	var err error
	switch contentType := r.Header.Get("Content-Type"); contentType {
	case "application/x-protobuf":
		var b = make([]byte, r.ContentLength)
		_, err := r.Body.Read(b)
		if err != nil {
			return err
		}
		if pb, ok := any(*data).(proto.Message); ok {
			err = proto.Unmarshal(b, pb)
		}
		if encoder, ok := any(*data).(ProtoEncoder); ok {
			err = proto.Unmarshal(b, encoder.ToProto())
		}
	case "application/xml":
		err = xml.NewDecoder(r.Body).Decode(data)
	default:
		err = json.NewDecoder(r.Body).Decode(data)
	}
	return err
}

// Success Send a success response
func Success(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The request was successful."
	}
	w.Header().Set("Content-Type", "plain/text")
	_, err := w.Write([]byte(message))
	if err != nil {
		log.Println(err)
		InternalServerError(w, r, "Could not write text")
	}
}

// SuccessHTML Send a success response as HTML
func SuccessHTML(w http.ResponseWriter, r *http.Request, html string) {
	w.Header().Set("Content-Type", "text/html")
	_, err := w.Write([]byte(html))
	if err != nil {
		log.Println(err)
		InternalServerError(w, r, "Could not write HTML")
	}
}

// SendComponent Send a component as a success response
func SendComponent(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/component")
	err := component.Render(context.Background(), w)
	if err != nil {
		log.Println(err)
		InternalServerError(w, r, "Could not render component")
	}
}

// StructOK Send a struct as a success response
func StructOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	SendStruct(w, r, http.StatusOK, data)
}

// StructCreated Send a struct as a created response
func StructCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	SendStruct(w, r, http.StatusCreated, data)
}

// NoContent Send a no content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// BadRequest Send and encode an invalid input problem
func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The request body is invalid."
	}
	NewProblem(
		"about:blank",
		http.StatusBadRequest,
		"Bad Request",
		message,
		"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400",
	).SendProblem(w, r)
}

// Unauthorized Send an UnauthorizedResponse as JSON or XML
func Unauthorized(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "You must be logged in to access this resource."
	}
	NewProblem(
		"about:blank",
		http.StatusUnauthorized,
		"Unauthorized",
		message,
		"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/401",
	).SendProblem(w, r)
}

// Forbidden Send a ForbiddenResponse as JSON or XML
func Forbidden(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "You do not have permission to access this resource."
	}
	NewProblem(
		"about:blank",
		http.StatusForbidden,
		"Forbidden",
		message,
		"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403",
	).SendProblem(w, r)
}

// NotFound Send a NotFoundResponse as JSON or XML
func NotFound(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The requested resource could not be found."
	}
	NewProblem(
		"about:blank",
		http.StatusNotFound,
		"Not Found",
		message,
		"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/404",
	).SendProblem(w, r)
}

// InternalServerError -- Send an InternalServerErrorResponse as JSON or XML
func InternalServerError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "An internal server error occurred."
	}
	NewProblem(
		"about:blank",
		http.StatusInternalServerError,
		"Internal Server Error",
		message,
		"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500",
	).SendProblem(w, r)
}
