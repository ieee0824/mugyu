package main

import (
	"compress/flate"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/itchio/go-brotli/enc"
	"github.com/lucas-clemente/quic-go/http3"
)

type compressResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w compressResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w compressResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func gzipHandler(fn http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Encoding", "gzip")
	gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		fmt.Printf("Error closing gzip: %+v\n", err)
	}
	defer func() {
		err := gz.Close()
		if err != nil {
			fmt.Printf("Error closing gzip: %+v\n", err)
		}
	}()
	gzr := compressResponseWriter{Writer: gz, ResponseWriter: w}
	fn(gzr, r)
}

func deflateHandler(fn http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Encoding", "deflate")
	df, err := flate.NewWriter(w, flate.BestSpeed)
	if err != nil {
		fmt.Printf("Error closing deflate: %+v\n", err)
	}
	defer func() {
		err := df.Close()
		if err != nil {
			fmt.Printf("Error closing deflate: %+v\n", err)
		}
	}()
	dfr := compressResponseWriter{Writer: df, ResponseWriter: w}
	fn(dfr, r)
}

func brotliHandler(fn http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Encoding", "br")

	op := &enc.BrotliWriterOptions{
		Quality: 4,
		LGWin:   10,
	}
	br := enc.NewBrotliWriter(w, op)
	defer func() {
		err := br.Close()
		if err != nil {
			fmt.Printf("Error closing brotli: %+v\n", err)
		}
	}()
	brr := compressResponseWriter{Writer: br, ResponseWriter: w}
	fn(brr, r)
}

func makeCompressionHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Assept-Encoding")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
			brotliHandler(fn, w, r)
			return
		} else if strings.Contains(r.Header.Get("Accept-Encoding"), "deflate") {
			deflateHandler(fn, w, r)
			return
		} else if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gzipHandler(fn, w, r)
			return
		}

		fn(w, r)
	}
}

func reverseProxy(backEndUrl string) func(http.ResponseWriter, *http.Request) {
	url, err := url.Parse(backEndUrl)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(url).ServeHTTP
}

func main() {
	var (
		bg       = flag.String("b", "http://localhost:9000", "background")
		port     = flag.Uint("p", 8080, "front port")
		useQuic  = flag.Bool("enable_quic", false, "use quic")
		certPath = flag.String("c", "", "cert file path")
		keyPath  = flag.String("k", "", "key file path")
	)
	flag.Parse()

	fmt.Println(*certPath)
	fmt.Println(*keyPath)

	proxyServer := http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: makeCompressionHandler(reverseProxy(*bg)),
	}

	if *useQuic {
		if err := http3.ListenAndServeQUIC(proxyServer.Addr, *certPath, *keyPath, proxyServer.Handler); err != nil {
			log.Fatalln(err)
		}
	} else {
		if err := proxyServer.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}
}
