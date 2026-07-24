package apperr

const (
	CodeOK           = 0
	CodeBadRequest   = 40000
	CodeUnsupported  = 40001
	CodeUnauthorized = 40100
	CodeLoginExpired = 40101
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeInternal     = 50000
	CodeDB           = 50020
	CodeRedis        = 50021
	CodeHA           = 50200
)
