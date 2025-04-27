package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/yalue/onnxruntime_go"
	"gocv.io/x/gocv"
)

type CameraDevice struct {
	ID     int
	Name   string
	Path   string
	Width  int
	Height int
}

type Detection struct {
	X1, Y1, X2, Y2 float32
	Confidence     float32
	Class          int
}

type ClassificationResult struct {
	ClassID    int
	ClassName  string
	Confidence float32
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

func processVideoFeed(img image.Image, app *App) image.Image {
	if app.Detector == nil || len(app.InputTensors) == 0 || len(app.OutputTensors) == 0 {
		return img
	}

	err := updateInputTensorWithImage(app.InputTensors[0], img)
	if err != nil {
		app.DataLabel.SetText(fmt.Sprintf("Error updating tensor: %v", err))
		return img
	}

	err = app.Detector.Run()
	if err != nil {
		app.DataLabel.SetText(fmt.Sprintf("Error running model: %v", err))
		return img
	}

	results := parseOutputTensor(app.OutputTensors[0])

	annotatedImg := drawClassificationResults(img, results)

	updateClassificationUI(app, results)

	return annotatedImg
}

func updateInputTensorWithImage(tensor *onnxruntime_go.Tensor[float32], img image.Image) error {
	size := 640
	mat, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return fmt.Errorf("failed to convert image to Mat: %v", err)
	}
	defer mat.Close()

	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(mat, &resized, image.Point{X: size, Y: size}, 0, 0, gocv.InterpolationLinear)

	tensorData := tensor.GetData()

	means := []float32{0.485, 0.456, 0.406}
	stds := []float32{0.229, 0.224, 0.225}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			pixel := resized.GetVecbAt(y, x)
			idx := (y*size + x) * 3

			r := float32(pixel[2]) / 255.0
			g := float32(pixel[1]) / 255.0
			b := float32(pixel[0]) / 255.0

			tensorData[idx] = (r - means[0]) / stds[0]   // R
			tensorData[idx+1] = (g - means[1]) / stds[1] // G
			tensorData[idx+2] = (b - means[2]) / stds[2] // B
		}
	}

	return nil
}

func parseOutputTensor(tensor *onnxruntime_go.Tensor[float32]) []ClassificationResult {
	outputData := tensor.GetData()
	results := []ClassificationResult{}
	shape := tensor.GetShape()

	fmt.Printf("Output tensor shape: %v\n", shape)
	type IndexedScore struct {
		index int
		score float32
	}

	scores := make([]IndexedScore, len(outputData))
	for i, score := range outputData {
		scores[i] = IndexedScore{index: i, score: score}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	for i := 0; i < 5 && i < len(scores); i++ {
		results = append(results, ClassificationResult{
			ClassID:    scores[i].index,
			ClassName:  getClassName(scores[i].index),
			Confidence: scores[i].score,
		})
	}

	return results
}

func drawClassificationResults(img image.Image, results []ClassificationResult) image.Image {
	mat, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return img
	}
	defer mat.Close()

	gocv.Rectangle(&mat, image.Rect(10, 10, 350, 30+len(results)*20),
		color.RGBA{0, 0, 0, 180}, -1)

	gocv.PutText(&mat, "Classification Results:",
		image.Point{15, 30}, gocv.FontHersheySimplex,
		0.5, color.RGBA{255, 255, 255, 255}, 1)

	for i, res := range results {
		text := fmt.Sprintf("%s: %.2f%%", res.ClassName, res.Confidence*100)
		gocv.PutText(&mat, text,
			image.Point{15, 50 + i*20},
			gocv.FontHersheySimplex, 0.5,
			color.RGBA{255, 255, 255, 255}, 1)
	}
	imgDet, err := mat.ToImage()
	if err != nil {
		return img
	} else {

		return imgDet
	}
}

func updateClassificationUI(app *App, results []ClassificationResult) {
	var body strings.Builder

	body.WriteString("Classification Results:\n\n")
	for i, res := range results {
		body.WriteString(fmt.Sprintf("%d. %s: %.2f%%\n",
			i+1, res.ClassName, res.Confidence*100))
	}

	app.DataBody.SetText(body.String())
	app.DataBody.Refresh()
}
func getClassName(classID int) string {
	labels := map[int]string{
		0: "accident",
	}

	if name, ok := labels[classID]; ok {
		return name
	}
	return fmt.Sprintf("Class %d", classID)
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
