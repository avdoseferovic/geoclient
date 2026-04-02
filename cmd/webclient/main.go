package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	webDir := flag.String("web-dir", "web", "directory containing index.html, wasm_exec.js, and eoclient.wasm")
	assetsDir := flag.String("assets-dir", ".", "directory containing gfx, maps, and pub subdirectories")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(*webDir)))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(*assetsDir))))

	log.Printf("serving web client on http://127.0.0.1%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
