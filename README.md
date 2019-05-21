# mugyu
`mugyu` is an onomatopoeia when pushing into a narrow place in Japanese.  
This is a proxy that compresses packets at the front end.  
Support encoding type is gzip and deflate, brotli.

It is supported quic.


# usage

```
$ go run main.go -b https://backend -c certfile -k keyfile -p 443 -enable_http3
```