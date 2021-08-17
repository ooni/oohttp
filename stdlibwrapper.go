// This is an adapter for integrating net/http dependend code.

package http

import "net/http"

type StdlibTransport struct {
	Transport
}

func (txp *StdlibTransport) RoundTrip(stdReq *http.Request) (*http.Response, error) {
	req, err := NewRequest(stdReq.Method, stdReq.URL.String(), stdReq.Body)
	if err != nil {
		return nil, err
	}
	req.Header = Header(stdReq.Header)
	resp, err := txp.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	stdResp := &http.Response{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMinor:       resp.ProtoMinor,
		ProtoMajor:       resp.ProtoMajor,
		Header:           http.Header(resp.Header),
		Body:             resp.Body,
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Close:            resp.Close,
		Uncompressed:     resp.Uncompressed,
		Trailer:          http.Header(resp.Trailer),
		Request:          stdReq, // TODO(kelmenhorst,bassosimone): is this ok?
		TLS:              resp.TLS,
	}
	return stdResp, nil
}
