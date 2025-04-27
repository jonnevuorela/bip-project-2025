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
 * @param *app
 */
func DetectCameras(app *App) {
	app.StatusLabel.SetText("Scanning for cameras...")
	app.StatusLabel.Refresh()

	devices := FindVideoDevices()
	var cameras []CameraDevice

	for i := 0; i < len(devices); i++ {
		fmt.Println(i)
		fmt.Println(devices[i])

		// probe the device with ffmpeg to exit early.
		// we have to do this, beacause gocv will hang
		// upon trying to handle errors from v4l2-ctl output.
		fmt.Printf("Probing device %d: %s\n", i, devices[i])
		if err := probeDeviceWithFFmpeg(devices[i]); err != nil {
			fmt.Println(err)
			continue
		}

		cmd := exec.Command("v4l2-ctl", "-d", devices[i], "--info")
		output, err := cmd.Output()
		if err != nil {
			fmt.Println("Error retrieving info for device %s: %v\n", devices[i], err)
			continue
		}

		var name string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, "Card type") {
				name = strings.TrimSpace(strings.Split(line, ":")[1])
				break
			}
		}

		// try to open device
		// if problems with opening video, try different backend. V4L2 works for now.
		cam, err := gocv.VideoCaptureFileWithAPI(devices[i], gocv.VideoCaptureV4L2)
		if err != nil {
			fmt.Printf("Error opening video device %s: %v\n", devices[i], err)
			continue
		}

		// get just single frame to confirm that device works
		mat := gocv.NewMat()
		if ok := cam.Read(&mat); !ok || mat.Empty() {
			fmt.Printf("Device %s is not providing frames or is incompatible.\n", devices[i])
			cam.Close()
			continue
		}

		cameras = append(cameras, CameraDevice{
			ID:     i,
			Path:   devices[i],
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
	app.StatusLabel.SetText(fmt.Sprintf("Camera detection completed"))
	time.Sleep(1000 * time.Millisecond)
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
		app.StopCurrent = nil // reset the kill signal
		time.Sleep(100 * time.Millisecond)
	}
	app.StopCurrent = make(chan bool)
	stopChan := app.StopCurrent

	// if problems with opening video, try different backend. V4L2 works for now.
	cam, err := gocv.VideoCaptureFileWithAPI(app.CameraDevices[deviceID].Path, gocv.VideoCaptureV4L2)

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
			case <-stopChan:
				app.StatusLabel.SetText("Stopping the video stream.")
				return
			default:
				if ok := cam.Read(&frame); ok && !frame.Empty() {
					app.StatusLabel.SetText("jamming")
					img, _ := frame.ToImage()

					if app.Detector != nil {
						pImg := processVideoFeed(img, app)
						app.DataLabel.SetText("jamming")
						app.CurrentImage.Store(pImg)
					} else {
						app.DataLabel.SetText("No detection instance.")
						app.CurrentImage.Store(img)
					}

					RefreshCanvas(app)
				}
				time.Sleep(33 * time.Millisecond) // ~30 FPS
			}
		}
	}()
}

/**
 * Probe devices stdout with ffmpeg without getting actual output from device.
 * @param device string
 * @return error
 */
func probeDeviceWithFFmpeg(device string) error {
	cmd := exec.Command("ffmpeg", "-f", "v4l2", "-i", device, "-t", "1", "-vframes", "1", "-f", "null", "-")
	cmd.Stdout = nil // suppress stdout
	cmd.Stderr = nil // suppress stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Device %s failed probe with ffmpeg: %v", device, err)
	}
	return nil
}
