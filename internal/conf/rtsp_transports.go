package conf

import "strings"

type RtspTransports []string

func (t *RtspTransports) Marshal(tRtspTransports string) error {
	*t = strings.Split(tRtspTransports, ",")
	return nil
}
