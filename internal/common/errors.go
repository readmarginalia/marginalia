package common

import "strconv"

type ServiceError struct {
	Reason string `json:"reason"`
	Code   int    `json:"-"`
}

func (e ServiceError) Error() string {
	return e.Reason + " (code: " + strconv.Itoa(e.Code) + ")"
}
