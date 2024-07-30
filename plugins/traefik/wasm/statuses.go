package main

type headerStatus string

const (
	REQUEST_HEADER_NAME = "X-Souin-Error-Location"

	STATUS_OK                      headerStatus = "OK"
	STATUS_EXCLUDED                headerStatus = "EXCLUDED"
	UNSUPPORTED_METHOD             headerStatus = "UNSUPPORTED_METHOD"
	CACHE_CONTROL_EXTRACTION_ERROR headerStatus = "CACHE_CONTROL_EXTRACTION_ERROR"
	IS_MUTATION_REQUEST            headerStatus = "IS_MUTATION_REQUEST"
	NOT_MODIFIED                   headerStatus = "NOT_MODIFIED"
	NEED_REVALIDATION              headerStatus = "NEED_REVALIDATION"
	STATUS_SWR                     headerStatus = "STATUS_SWR"
	STATUS_MUST_REVALIDATE         headerStatus = "STATUS_MUST_REVALIDATE"
	STATUS_STALE                   headerStatus = "STATUS_STALE"
	STATUS_TO_STORE                headerStatus = "STATUS_TO_STORE"
)
