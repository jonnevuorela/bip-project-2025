package main

import (
	"fmt"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/yalue/onnxruntime_go"
	"gocv.io/x/gocv"
)

/**
 * The main app struct.
 * If you add any kind of functionality or ui elements into app,
 * you should probably start from here.
 */
type App struct {
	// UI
	Window        fyne.Window
	MainContent   fyne.CanvasObject
	ContentCanvas fyne.CanvasObject
	ControlPanel  fyne.CanvasObject
	VideoCanvas   *canvas.Raster
	StatusLabel   *widget.Label
	DeviceSelect  *widget.Select
	DataLabel     *widget.Label
	DataBody      *widget.TextGrid

	// Video
	CurrentImage  *atomic.Value
	StopCurrent   chan bool
	CameraDevices []CameraDevice
	Video         *gocv.VideoCapture

	// Detection
	Detector      *onnxruntime_go.Session[float32]
	InputTensors  []*onnxruntime_go.Tensor[float32]
	OutputTensors []*onnxruntime_go.Tensor[float32]
	Detections    []Detection
}

func main() {
	envErr := onnxruntime_go.InitializeEnvironment()
	if envErr != nil {
		fmt.Printf("Error initializing onnx environment: %v", envErr)
	}
	defer onnxruntime_go.DestroyEnvironment()

	a := app.New()
	w := a.NewWindow("SmartSign™")

	accidentDetector, inputTensors, outputTensors := LoadDetectionModel()
	defer accidentDetector.Destroy()

	app := &App{
		Window:        w,
		CurrentImage:  &atomic.Value{},
		StopCurrent:   make(chan bool),
		Detector:      accidentDetector,
		InputTensors:  inputTensors,
		OutputTensors: outputTensors,
	}

	detErr := app.Detector.Run()
	if detErr != nil {
		fmt.Printf("Error starting ONNX session: %v", detErr)
	}

	SetupUI(app)
	w.Resize(fyne.NewSize(1280, 720))
	w.Show()
	go DetectCameras(app)
	a.Run()
}
