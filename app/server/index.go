package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

var (
	listenAddress = flag.String("listen", ":8081", "Where the server listens for connections. [interface]:port")
)

func init() {
	flag.Parse()
}

//go:embed static
var assetsFs embed.FS

//go:embed linux_json_api.sh
var linuxJsonApiSh string

//go:embed ping_hosts
var pingHosts []byte

var jsFnEnd = regexp.MustCompile(`(?m)^\}$`)

const Shebang = `#!/bin/bash`

func extractShell(name string) string {
	re := fmt.Sprintf(`(?m)^%s\(\)\s*\{$`, regexp.QuoteMeta(name))
	fnStart := regexp.MustCompile(re)
	idx := fnStart.FindStringSubmatchIndex(linuxJsonApiSh)
	if len(idx) == 0 {
		return ""
	}

	sub := linuxJsonApiSh[idx[0]:]
	endIdx := jsFnEnd.FindStringSubmatchIndex(sub)
	commonEnd := jsFnEnd.FindStringSubmatchIndex(linuxJsonApiSh)

	return linuxJsonApiSh[len(Shebang)+2:commonEnd[0]+2] + sub[:endIdx[0]+2] + name + "\n"
}

const invalidModule = `'{"success":false,"status":"Invalid module"}'`

func main() {
	assets, _ := fs.Sub(assetsFs, "static")

	http.Handle("/", http.FileServer(http.FS(assets)))
	http.HandleFunc("/server/", func(w http.ResponseWriter, r *http.Request) {
		module := r.URL.Query().Get("module")
		if module == "" {
			http.Error(w, "No module specified, or requested module doesn't exist.", 406)
			return
		}

		if module == "ping" {
			os.WriteFile("/tmp/ping_hosts", pingHosts, os.ModePerm)
		}

		// Execute the command
		shell := extractShell(module)
		if shell == "" {
			w.Write([]byte(invalidModule))
			return
		}
		cmd := exec.Command("/bin/bash", "-c", shell)
		var output bytes.Buffer
		cmd.Stdout = &output
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error executing '%s': %s\n\tScript output: %s\n", module, err.Error(), output.String())
			http.Error(w, "Unable to execute module.", http.StatusInternalServerError)
			return
		}

		w.Write(output.Bytes())
	})

	fmt.Println("Starting http server at:", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		fmt.Println("Error starting http server:", err)
		os.Exit(1)
	}
}
