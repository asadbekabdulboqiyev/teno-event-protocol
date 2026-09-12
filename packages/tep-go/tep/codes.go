package tep

import "net/http"

type TepCode string

const (
	CodeOK               TepCode = "OK"
	CodeEventAccepted    TepCode = "EVENT_ACCEPTED"
	CodeBadRequest       TepCode = "BAD_REQUEST"
	CodeSigInvalid       TepCode = "SIG_INVALID"
	CodeVersionUnsupported TepCode = "VERSION_UNSUPPORTED"
	CodeDuplicateEvent   TepCode = "DUPLICATE_EVENT"
	CodeEventNotFound    TepCode = "EVENT_NOT_FOUND"
	CodeUpstreamError    TepCode = "UPSTREAM_ERROR"
)

func httpStatus(code TepCode) int {
	switch code {
	case CodeOK, CodeDuplicateEvent:
		return http.StatusOK
	case CodeEventAccepted:
		return http.StatusAccepted
	case CodeSigInvalid:
		return http.StatusUnauthorized
	case CodeVersionUnsupported:
		return http.StatusUnsupportedMediaType
	case CodeEventNotFound:
		return http.StatusNotFound
	case CodeUpstreamError:
		return http.StatusBadGateway
	default:
		return http.StatusBadRequest
	}
}