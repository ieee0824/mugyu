package main

import (
	"bytes"
	"compress/flate"
	gz "compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

type encodeMode int

func (e encodeMode) String() string {
	switch e {
	case gzip:
		return "gzip"
	case deflate:
		return "deflate"
	case brotil:
		return "br"
	}
	return ""
}

const (
	none encodeMode = iota << 1
	gzip
	deflate
	brotil
)

func parseEncodeMode(s string) encodeMode {
	if strings.Contains(s, "br") {
		return brotil
	} else if strings.Contains(s, "gzip") {
		return gzip
	} else if strings.Contains(s, "deflate") {
		return deflate
	}
	return none
}

func compressDeflate(r io.Reader) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	zw, err := flate.NewWriter(buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	defer zw.Close()

	if _, err := io.Copy(zw, r); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(buf), err
}

func compressGzip(r io.Reader) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	zw, err := gz.NewWriterLevel(buf, gz.BestCompression)
	if err != nil {
		return nil, err
	}
	defer zw.Close()

	if _, err := io.Copy(zw, r); err != nil {
		return nil, err
	}

	return ioutil.NopCloser(buf), err
}

func main() {
	director := func(request *http.Request) {
		request.URL.Scheme = "http"
		request.URL.Host = ":9000"
	}

	modifier := func(res *http.Response) error {
		m := parseEncodeMode(res.Request.Header.Get("Accept-Encoding"))
		fmt.Println(m)
		buf := new(bytes.Buffer)
		io.Copy(buf, res.Body)
		res.Body = ioutil.NopCloser(buf)
		io.Copy(res.Body, buf)

		switch m {
		case gzip:
			r, err := compressGzip(buf)
			if err != nil {
				return err
			}
			//res.Body = r
			//res.Header.Set("Content-Encoding", m.String())
			_ = r
		case deflate:
			r, err := compressDeflate(buf)
			if err != nil {
				fmt.Println(err)
				return err
			}
			//res.Body = r
			//res.Header.Set("Content-Encoding", m.String())
			_ = r
		case brotil:
			res.Header.Set("Content-Encoding", m.String())
		default:
		}

		return nil
	}

	rp := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifier,
	}
	server := http.Server{
		Addr:    ":8080",
		Handler: rp,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
