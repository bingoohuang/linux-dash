package linux_dash

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

//go:embed static
var assetsFs embed.FS
var DashStatic, _ = fs.Sub(assetsFs, "static")

//go:embed static/linux_json_api.sh
var linuxJsonApiSh string

//go:embed static/ping_hosts
var pingHosts string

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

func DashServe(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	if module == "" {
		http.Error(w, "No module specified, or requested module doesn't exist.", 406)
		return
	}

	// Execute the command
	shell := extractShell(module)
	if shell == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(invalidModule))
		return
	}
	if module == "ping" {
		shell = strings.ReplaceAll(shell, "PING_HOSTS", pingHosts)
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(output.Bytes())
}
