package responses

import (
	"encoding/xml"
	"home_api/src/proto/problempb"
	"net/http"

	"github.com/goccy/go-json"
	"google.golang.org/protobuf/proto"
)

// -------------- Structs --------------

// problem -- Defined by https://www.rfc-editor.org/rfc/rfc9457.html#section-3
type problem struct {
	*problempb.Problem
}

// NewProblem -- Create a new Problem
func NewProblem(Type string, Status int, Title string, Detail string, Instance string) *problem {
	return &problem{
		&problempb.Problem{
			Type:     Type,
			Status:   int32(Status),
			Title:    Title,
			Detail:   Detail,
			Instance: Instance,
		},
	}
}

// SendProblem -- Send a Problem as JSON, XML or Protobuf
func (problem *problem) SendProblem(w http.ResponseWriter, r *http.Request) {
	var content string = "application/problem+"
	var structBytes []byte
	switch accept := r.Header.Get("Accept"); accept {
	case "application/x-protobuf":
		content += "x-protobuf"
		if pb, ok := any(problem).(proto.Message); ok {
			structBytes, _ = proto.Marshal(pb)
		}
		if encoder, ok := any(problem).(ProtoEncoder); ok {
			structBytes, _ = proto.Marshal(encoder.ToProto())
		}
	case "application/xml":
		content += "xml"
		structBytes, _ = xml.Marshal(problem)
	}
	if structBytes == nil {
		content += "json"
		structBytes, _ = json.Marshal(problem)
	}
	w.Header().Set("Content-Type", content)
	w.WriteHeader(int(problem.Status))
	w.Write(structBytes)
}
