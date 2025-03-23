package main

import (
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"gocv.io/x/gocv"
)

/**
 * The main app struct.
 * If you add any kind of functionality or ui elements into app,
 * you should probably start from here.
 */
type App struct {
	Window        fyne.Window
	MainContent   fyne.CanvasObject
	ContentCanvas fyne.CanvasObject
	ControlPanel  fyne.CanvasObject
	VideoCanvas   *canvas.Raster
	StatusLabel   *widget.Label
	DeviceSelect  *widget.Select
	DetailsText   *widget.Label

	Video *gocv.VideoCapture

	CurrentImage  *atomic.Value
	StopCurrent   chan bool
	CameraDevices []CameraDevice
}

func main() {
	a := app.New()
	w := a.NewWindow("SmartSign™")

	app := &App{
		Window:       w,
		CurrentImage: &atomic.Value{},
		StopCurrent:  make(chan bool),
	}

	SetupUI(app)
	w.Resize(fyne.NewSize(1280, 720))
	w.Show()
	go DetectCameras(app)
	a.Run()
}
