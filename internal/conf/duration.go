package conf

import (
	"regexp"
	"strconv"
	"time"
)

var reDays = regexp.MustCompile("^(-?[0-9]+)d")

// Duration is a duration. It differs from the standard duration in these ways:
// - it is unmarshaled/marshaled from/to a string (instead of a number)
// - it supports days
type Duration time.Duration

func (d *Duration) Marshal(tDescription string) error {
	negative := false
	days := int64(0)

	m := reDays.FindStringSubmatch(string(tDescription))
	if m != nil {
		days, _ = strconv.ParseInt(m[1], 10, 64)
		if days < 0 {
			negative = true
			days = -days
		}

		tDescription = tDescription[len(m[0]):]
	}

	var nonDays time.Duration

	if len(tDescription) != 0 {
		var err error
		nonDays, err = time.ParseDuration(string(tDescription))
		if err != nil {
			return err
		}
	}

	nonDays += time.Duration(days) * 24 * time.Hour
	if negative {
		nonDays = -nonDays
	}
	//fmt.Println("Parsed Duration:", nonDays)
	*d = Duration(nonDays)
	return nil
}
