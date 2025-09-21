package defs

import (
	"XMedia/internal/auth"
	"net"

	"github.com/google/uuid"
)

// PathAccessRequest is a path access request.
type PathAccessRequest struct {
	Name     string
	Query    string
	Publish  bool
	SkipAuth bool

	// only if skipAuth = false
	Proto auth.Protocol
	ID    *uuid.UUID
	IP    net.IP
}
