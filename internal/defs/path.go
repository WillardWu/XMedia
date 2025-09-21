package defs

import (
	"XMedia/internal/stream"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
)

type Path interface {
	StartPublisher(req PathStartPublisherReq) (*stream.Stream, error)
}

// PathAddPublisherRes contains the response of AddPublisher().
type PathAddPublisherRes struct {
	Path Path
	Err  error
}

// PathAddPublisherReq contains arguments of AddPublisher().
type PathAddPublisherReq struct {
	Author        Publisher
	AccessRequest PathAccessRequest
	Res           chan PathAddPublisherRes
}

// PathStartPublisherRes contains the response of StartPublisher().
type PathStartPublisherRes struct {
	Stream *stream.Stream
	Err    error
}

// PathStartPublisherReq contains arguments of StartPublisher().
type PathStartPublisherReq struct {
	Author             Publisher
	Desc               *description.Session
	GenerateRTPPackets bool
	Res                chan PathStartPublisherRes
}
