package download

import "net/http"

type RequestFieldGetter interface {
	GetHeaderMap() http.Header
}

func (h *httpRequest) GetHeaderMap() http.Header {
	return h.request.Header
}

type HttpClientSetter interface {
	SetHttpClient(client HttpClient)
}

func (h *httpRequest) SetHttpClient(client HttpClient) {
	h.client = client
}
