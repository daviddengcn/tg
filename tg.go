/*
  Timelog application
*/
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	//"github.com/daviddengcn/go-villa"
)

var (
	server string = ":8080"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func init() {
	flag.StringVar(&server, "server", ":8080", "Listening address [address]:port")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tg [flags] FILES...\n")
		flag.PrintDefaults()
	}
}

func main() {
	//	 fn := villa.Path(`/home/daviddeng/local/logs/hbase-hadoop-regionserver-hbase2798.frc3.facebook.com.log.2014-05-20-08`)
	//	fn := villa.Path(`/home/daviddeng/local/logs/hbase-hadoop-regionserver-hbase757.frc1.facebook.com.log.2014-07-08`)
	flag.Parse()

	//	fns := flag.Args()

	http.HandleFunc("/source", pageSource)
	http.HandleFunc("/", makeGzipHandler(pageRoot))

	fmt.Println("Start server at", server)
	log.Fatal(http.ListenAndServe(server, nil))
}
