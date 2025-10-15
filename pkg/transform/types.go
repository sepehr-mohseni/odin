package transform

// RequestTransform defines transformations to apply to requests
type RequestTransform struct {
Headers     map[string]string
QueryParams map[string]string
Body        string
}

// ResponseTransform defines transformations to apply to responses
type ResponseTransform struct {
Headers map[string]string
Body    string
}
