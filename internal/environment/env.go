package environment

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type env struct {
	GoBin       string `json:"GOBIN"`
	GoMod       string `json:"GOMOD"`
	GoModCache  string `json:"GOMODCACHE"`
	GoRoot      string `json:"GOROOT"`
	GoPath      string `json:"GOPATH"`
	GoVersion   string `json:"GOVERSION"`
	GoWorkspace string `json:"GOWORK"` // TODO implement module lookup by workspace
	ProjectRoot string
}

func init() {
	enironment = &env{}
	initArgs()
	initEnv()
}
func initArgs() {
	args := make([]string, len(os.Args))
	copy(args, os.Args)
	if len(os.Args) < 2 {
		args = append(args, ".")
	}

	enironment.ProjectRoot = args[1]
}

func initEnv() {
	cmd := exec.Command("go", "env", "-json")
	cmd.Dir = enironment.ProjectRoot
	data, err := cmd.Output()
	if err != nil {
		log.Fatal("unable to get environment variables: %w", err)
	} else {
		err = json.Unmarshal(data, enironment)
		if err != nil {
			log.Fatal("unable to unmarshal environment variables: %w", err)
		}
	}

	if len(enironment.GoRoot) > 0 {
		log.Println("env is preloaded %v", enironment)
		return
	}

	r := os.Getenv("GOROOT")
	if len(r) != 0 {
		enironment.GoRoot = r
	}

	p := os.Getenv("GOPATH")
	if len(p) != 0 {
		enironment.GoPath = p
		enironment.GoModCache = filepath.Join(p, modSubPath)
	}
}
