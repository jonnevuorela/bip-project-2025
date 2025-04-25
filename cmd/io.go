package main

import (
	"fmt"
	"image"
	"image/color"
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
					// Process frame with YOLO
					processedFrame, detections := ProcessFrame(frame)

					// Update status with detection information
					if len(detections) > 0 {
						app.StatusLabel.SetText(fmt.Sprintf("Detected %d objects", len(detections)))
					} else {
						app.StatusLabel.SetText("Streaming")
					}

					// Convert to image for display
					img, _ := processedFrame.ToImage()
					app.CurrentImage.Store(img)

					// Clean up
					processedFrame.Close()
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

func processFrameForDetection(frame gocv.Mat, app *App) {
	if app.Detector == nil {
		return
	}

	detector, ok := app.Detector.(*AccidentModel)
	if !ok {
		fmt.Println("Error: Detector is not correct type")
		return
	}
	frameCopy := frame.Clone()
	defer frameCopy.Close()

	blob := gocv.BlobFromImage(
		frameCopy,
		1.0/255.0,
		// should always match the training dimensions
		image.Pt(640, 640),
		gocv.NewScalar(0, 0, 0, 0),
		true,
		false,
	)
	defer blob.Close()

	detector.Net.SetInput(blob, "images")

	output := detector.Net.Forward("output0")
	defer output.Close()

	boxes, confidences, classIds := processOutput(output, frame.Cols(), frame.Rows())

	indices := performNMS(boxes, confidences)

	for _, idx := range indices {
		if idx >= len(boxes) || idx >= len(confidences) || idx >= len(classIds) {
			fmt.Print("Warning: Invalid detection index: %d", idx)
			continue
		}

		box := boxes[idx]
		classId := classIds[idx]
		confidence := confidences[idx]

		className := "unknown"
		if classId >= 0 && classId < len(detector.ClassNames) {
			className = detector.ClassNames[classId]
		} else {
			fmt.Print("Warning: ClassId %d is out of bounds for array with %d elements", classId, len(detector.ClassNames))
		}

		gocv.Rectangle(&frame, box, color.RGBA{0, 255, 0, 255}, 2)
		label := fmt.Sprintf("%s: %.2f%%", className, confidence*100)

		yPos := box.Min.Y - 10
		if yPos < 10 {
			yPos = box.Min.Y + 20
		}
		gocv.PutText(&frame, label,
			image.Pt(box.Min.X, yPos),
			gocv.FontHersheyPlain, 0.5,
			color.RGBA{0, 255, 0, 255}, 2)
	}

}

func processOutput(out gocv.Mat, frameWidth, frameHeight int) ([]image.Rectangle, []float32, []int) {
	var boxes []image.Rectangle
	var confidences []float32
	var classIds []int

	confThreshold := float32(0.4)

	numRows := out.Rows()
	numCols := out.Cols()
	fmt.Printf("YOLOv5 output shape: %d x %d\n", numRows, numCols)

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered: %v\n", r)
			}
		}()

		numClasses := 1
		//boxSize := 5 + numClasses

		for i := 0; i < numRows; i++ {
			confidence := float32(out.GetFloatAt(i, 4))

			if confidence >= confThreshold {
				maxScore := float32(0)
				classId := 0

				for c := 0; c < numClasses; c++ {
					score := float32(out.GetFloatAt(i, 5+c))
					if score > maxScore {
						maxScore = score
						classId = c
					}
				}

				if maxScore >= confThreshold {
					x := float32(out.GetFloatAt(i, 0))
					y := float32(out.GetFloatAt(i, 1))
					w := float32(out.GetFloatAt(i, 2))
					h := float32(out.GetFloatAt(i, 3))

					left := int((x - w/2) * float32(frameWidth))
					top := int((y - h/2) * float32(frameHeight))
					right := int((x + w/2) * float32(frameWidth))
					bottom := int((y + h/2) * float32(frameHeight))

					boxes = append(boxes, image.Rect(left, top, right, bottom))
					confidences = append(confidences, confidence*maxScore)
					classIds = append(classIds, classId)
				}
			}
		}
	}()

	fmt.Printf("Found %d detections\n", len(boxes))
	return boxes, confidences, classIds
}
func performNMS(boxes []image.Rectangle, confidences []float32) []int {
	if len(boxes) == 0 {
		return []int{}
	}

	indices := make([]int, len(boxes))
	for i := range indices {
		indices[i] = i
	}

	for i := 0; i < len(indices)-1; i++ {
		for j := i + 1; j < len(indices); j++ {
			if confidences[indices[i]] < confidences[indices[j]] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	nmsThreshold := 0.4

	kept := make([]int, 0)
	for len(indices) > 0 {
		idx := indices[0]
		kept = append(kept, idx)

		remainder := make([]int, 0)
		for _, otherIdx := range indices[1:] {
			if calculateIoU(boxes[idx], boxes[otherIdx]) <= nmsThreshold {
				remainder = append(remainder, otherIdx)
			}
		}

		indices = remainder
	}

	return kept
}

func calculateIoU(boxA, boxB image.Rectangle) float64 {
	intersection := boxA.Intersect(boxB)
	intersectionArea := intersection.Dx() * intersection.Dy()

	if intersectionArea <= 0 {
		return 0.0
	}

	boxAArea := boxA.Dx() * boxA.Dy()
	boxBArea := boxB.Dx() * boxB.Dy()
	unionArea := boxAArea + boxBArea - intersectionArea

	return float64(intersectionArea) / float64(unionArea)
}
