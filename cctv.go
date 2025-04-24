package main

import (
	// "time"
	"context"
	"fmt"
	"github.com/greendrake/cctv/camera"
	"github.com/greendrake/cctv/webcast"
	"github.com/greendrake/server_client_hierarchy"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// The top node for holding and puppet-mastering all Camera nodes
type CCTV struct {
	server_client_hierarchy.Node
	webCastIDs []string
}

func New(ctx context.Context, camSet map[camera.CamName]*camera.Camera, baseDir string, WebCastPort string) *CCTV {
	cctv := &CCTV{}
	cctv.GetNode().ID = "CCTV"
	cctv.SetContextWaiter(ctx)
	for _, cam := range camSet {
		if cam.HasAnythingToDo() {
			for _, sId := range cam.WebCast {
				cctv.webCastIDs = append(cctv.webCastIDs, fmt.Sprintf("%v/%v", cam.Name, sId))
			}
			cam.Init(baseDir)
			cctv.AddClient(cam)
		}
	}
	if len(cctv.webCastIDs) > 0 {
		casterGetter := func(cam string, ssId string) *webcast.Caster {
			sId, _ := strconv.Atoi(ssId)
			return camSet[camera.CamName(cam)].GetStream(camera.StreamID(sId)).GetCaster()
		}
		go webcast.Run(ctx, WebCastPort, cctv.webCastIDs, casterGetter)
	}
	return cctv
}

func GetWorkDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dir := filepath.Dir(ex)
	// Helpful when developing:
	// when running `go run`, the executable is in a temporary directory.
	if strings.Contains(dir, "go-build") {
		return "."
	}
	return filepath.Dir(ex)
}

func main() {
	// Log to STDOUT in the standard manner
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.LUTC)

	err := os.Chdir(GetWorkDir())
	if err != nil {
		log.Fatal(err)
	}

	// Read camera configuration from YAML file "config.yaml"
	configFile := "config.yaml"
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open config file %s: %v", configFile, err)
	}
	defer f.Close()

	var config struct {
		BaseDir     string           `yaml:"BaseDir"`
		WebCastPort string           `yaml:"WebCastPort"`
		Cameras     []*camera.Camera `yaml:"Cameras"`
	}

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to parse YAML config: %v", err)
	}

	cams := config.Cameras
	camLen := len(cams)
	if camLen > 0 {
		log.Printf("Started with %v camera(s)", camLen)
		anythingToDo := false
		camSet := make(map[camera.CamName]*camera.Camera)
		// Verify that all camera names are unique
		for _, cam := range cams {
			_, exists := camSet[cam.Name]
			if exists {
				log.Printf("Duplicate camera name: %s\n", cam.Name)
			} else {
				camSet[cam.Name] = cam
				if !anythingToDo && cam.HasAnythingToDo() {
					anythingToDo = true
				}
			}
		}
		if anythingToDo {
			baseDir := config.BaseDir
			WebCastPort := config.WebCastPort
			// We've got some properly configured cameras, hence some real job to do.
			// Create a context that is responsive to signals:
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			cctv := New(ctx, camSet, baseDir, WebCastPort)
			defer func() {
				log.Println("All finished")
				stop()
			}()
			cctv.Wait()
		} else {
			log.Println("No cameras specify anything to do (Save or WebCast)")
		}
	} else {
		log.Println("No cameras configured")
	}
}
