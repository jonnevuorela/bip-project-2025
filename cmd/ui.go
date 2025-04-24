package main

import (
	"image"

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
	var aspectRatio float32 = 16.0 / 9.0
	videoWidth := 1000
	app.VideoCanvas = canvas.NewRaster(func(w, h int) image.Image {
		if img := app.CurrentImage.Load(); img != nil {
			return img.(image.Image)
		}

		return image.NewRGBA(image.Rect(0, 0, w, int(float32(w)/aspectRatio)))
	})

	app.VideoCanvas.SetMinSize(fyne.NewSize(float32(videoWidth), float32(float32(videoWidth)/aspectRatio)))

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
	dataContainer := container.NewVBox(
		widget.NewLabel("Detections:"),
		app.DetailsText,
	)

	videoContainer := container.NewCenter(app.VideoCanvas)
	content := container.NewVSplit(videoContainer, dataContainer)

	split := container.NewHSplit(controls, content)
	split.Offset = 0.2

	app.Window.Resize(fyne.NewSize(1280, 720))
	app.Window.SetFixedSize(false)

	app.Window.SetContent(split)
}

func UpdateDeviceList(app *App) {
	options := make([]string, len(app.CameraDevices))

	for i := 0; i < len(app.CameraDevices); i++ {
		options[i] = app.CameraDevices[i].Name
	}
	app.DeviceSelect.Options = options
	app.DeviceSelect.OnChanged = func(selected string) {
		for i := 0; i < len(app.CameraDevices); i++ {
			option := app.CameraDevices[i].Name
			if selected == option {
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
