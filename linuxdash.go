package linuxdash

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

//go:embed static
var assetsFs embed.FS
var DashStatic, _ = fs.Sub(assetsFs, "static")

//go:embed static/linux_json_api.sh
var LinuxJsonApiSh string

//go:embed static/ping_hosts
var PingHosts []byte

var jsFnEnd = regexp.MustCompile(`(?m)^\}$`)

const Shebang = `#!/bin/bash`

func ExtractShell(module string) string {
	re := fmt.Sprintf(`(?m)^%s\(\)\s*\{$`, regexp.QuoteMeta(module))
	fnStart := regexp.MustCompile(re)
	idx := fnStart.FindStringSubmatchIndex(LinuxJsonApiSh)
	if len(idx) == 0 {
		return ""
	}

	sub := LinuxJsonApiSh[idx[0]:]
	endIdx := jsFnEnd.FindStringSubmatchIndex(sub)
	commonEnd := jsFnEnd.FindStringSubmatchIndex(LinuxJsonApiSh)

	s := LinuxJsonApiSh[len(Shebang)+2:commonEnd[0]+2] + sub[:endIdx[0]+2] + module + "\n"

	return s
}

const invalidModule = `'{"success":false,"status":"Invalid module"}'`

func MakeDashServe(f func(module string) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { DashServe(w, r, f) }
}

func DashServe(w http.ResponseWriter, r *http.Request, f func(module string) ([]byte, error)) {
	module := r.URL.Query().Get("module")
	if module == "" {
		http.Error(w, "No module specified, or requested module doesn't exist.", 406)
		return
	}

	if out, err := f(module); err != nil {
		log.Printf("Error executing '%s': %s\n\tScript output: %s\n", module, err.Error(), string(out))
		http.Error(w, "Unable to execute module.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(out)
	}
}

func ExecuteShell(module string) ([]byte, error) {
	// Execute the command
	shell := ExtractShell(module)
	if shell == "" {
		return []byte(invalidModule), nil
	}

	if module == "ping" {
		os.WriteFile(".linuxdash_ping_hosts", PingHosts, os.ModePerm)
	}

	cmd := exec.Command("/bin/bash", "-c", shell)
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Run()
	return output.Bytes(), err
}
