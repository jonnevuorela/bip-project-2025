package main

import (
	"fmt"
	"image"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

/**
 * UI setup function called externally from main function.
 * Any UI modifications should be currently done here.
 * @param *app
 */
func SetupUI(app *App) {
	app.VideoCanvas = canvas.NewRaster(func(w, h int) image.Image {
		if img := app.CurrentImage.Load(); img != nil {
			return img.(image.Image)
		}
		return image.NewRGBA(image.Rect(0, 0, w, h))
	})

	app.StatusLabel = widget.NewLabel("Ready")
	app.DeviceSelect = widget.NewSelect(nil, nil)

	refreshBtn := widget.NewButton("Refresh Cameras", func() {
		go DetectCameras(app)
	})

	controls := container.NewVBox(
		widget.NewLabel("Select Camera:"),
		app.DeviceSelect,
		refreshBtn,
		app.StatusLabel,
	)

	split := container.NewHSplit(controls, app.VideoCanvas)
	split.Offset = 0.2
	app.Window.SetContent(split)
}

func UpdateDeviceList(app *App) {
	options := make([]string, len(app.CameraDevices))
	for i, cam := range app.CameraDevices {
		options[i] = fmt.Sprintf("%s (%dx%d)", cam.Name, cam.Width, cam.Height)
	}

	app.DeviceSelect.Options = options
	app.DeviceSelect.OnChanged = func(selected string) {
		for i, cam := range app.CameraDevices {
			if strings.HasPrefix(selected, cam.Name) {
				startStream(app, i)
				break
			}
		}
	}
	app.DeviceSelect.Refresh()
}

func RefreshCanvas(app *App) {
	app.VideoCanvas.Refresh()
}

/**
 * The layout on App.MainContent takes content like this as parameters
 *
 * @return The CanvasObject returned should be assigned to the corresponding field on the App struct.
 */
func LeftPanel() fyne.CanvasObject {
	content := widget.NewLabel("Here is the left panel.")

	return content
}

/**
 * The layout on App.MainContent takes content like this as parameters
 *
 * @return The CanvasObject returned should be assigned to the corresponding field on the App struct.
 */
func HelloWorld() fyne.CanvasObject {
	hello := widget.NewLabel("Hello Fyne!")
	content := container.NewVBox(
		hello,
		widget.NewButton("Hi!", func() {
			hello.SetText("Welcome :)")
		}),
	)
	return content

}
