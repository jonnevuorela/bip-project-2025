package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

type CameraDevice struct {
	ID     int
	Name   string
	Path   string
	Width  int
	Height int
}

/**
 * Find video devices on /dev/ dir.
 * this shouldnt work on windows machines.
 * @return devices []string
 */
func FindVideoDevices() []string {
	var devices []string
	cmd := exec.Command("v4l2-ctl", "--list-devices")
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/dev/video") {
			path := strings.TrimSpace(line)
			if _, err := os.Stat(path); err == nil {
				devices = append(devices, path)
			}
		}
	}
	return devices
}

/**
 * DetectCameras and save to the app state.
 * (or it should, but it gets stuck for now, even if it gets entries in devices)
 * @param *app
 */
func DetectCameras(app *App) {
	app.StatusLabel.SetText("Scanning for cameras...")
	app.StatusLabel.Refresh()

	devices := FindVideoDevices()
	var cameras []CameraDevice

	for idx, path := range devices {
		name := fmt.Sprintf("Camera %d", idx)
		cmd := exec.Command("v4l2-ctl", "-d", path, "-D")
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), "Camera") {
				name = strings.TrimPrefix(string(output), "Driver Info:\n\t")
			}
		}

		cam, err := gocv.VideoCaptureFile(path)
		if err != nil {
			continue
		}

		mat := gocv.NewMat()
		if ok := cam.Read(&mat); !ok {
			cam.Close()
			continue
		}

		cameras = append(cameras, CameraDevice{
			ID:     idx,
			Path:   path,
			Name:   name,
			Width:  mat.Cols(),
			Height: mat.Rows(),
		})

		mat.Close()
		cam.Close()
		time.Sleep(100 * time.Millisecond)
	}

	app.CameraDevices = cameras
	UpdateDeviceList(app)
	app.StatusLabel.SetText(fmt.Sprintf("Found %d cameras", len(cameras)))
	app.StatusLabel.Refresh()
}

/**
 * Start streaming goroutine with selected device
 * @param *app, deviceID
 */
func startStream(app *App, deviceID int) {
	if deviceID >= len(app.CameraDevices) {
		return
	}

	if app.StopCurrent != nil {
		close(app.StopCurrent)
	}
	app.StopCurrent = make(chan bool)

	cam, err := gocv.VideoCaptureFile(app.CameraDevices[deviceID].Path)
	if err != nil {
		app.StatusLabel.SetText("Error opening device")
		return
	}

	go func() {
		defer cam.Close()
		frame := gocv.NewMat()
		defer frame.Close()

		for {
			select {
			case <-app.StopCurrent:
				return
			default:
				if ok := cam.Read(&frame); ok && !frame.Empty() {
					img, _ := frame.ToImage()
					app.CurrentImage.Store(img)
					RefreshCanvas(app)
				}
				time.Sleep(33 * time.Millisecond) // ~30 FPS
			}
		}
	}()
}
