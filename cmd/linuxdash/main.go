package main

import (
	"flag"
	"fmt"
	"github.com/bingoohuang/linux_dash"
	"net/http"
	"os"
)

func main() {
	http.Handle("/", http.FileServer(http.FS(linux_dash.DashStatic)))
	http.HandleFunc("/server/", linux_dash.DashServe)

	listen := flag.String("listen", ":8081", "Where the server listens for connections. [interface]:port")
	flag.Parse()

	fmt.Println("Starting http server at:", *listen)
	if err := http.ListenAndServe(*listen, nil); err != nil {
		fmt.Println("Error starting http server:", err)
		os.Exit(1)
	}
}
