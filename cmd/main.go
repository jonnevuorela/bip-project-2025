package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

/**
 * The main app struct.
 * If you add any kind of functionality or ui elements into app,
 * you should probably start from here.
 */
type App struct {
	Window      fyne.Window
	MainContent fyne.CanvasObject

	ContentCanvas fyne.CanvasObject
	LeftCanvas    fyne.CanvasObject
}

func main() {
	a := app.New()
	w := a.NewWindow("SmartSign™")

	app := &App{
		Window: w,
	}

	app.ContentCanvas = HelloWorld() // add content to centercanvas
	app.LeftCanvas = LeftPanel()     // add content to leftcanvas

	content := CreateLayout(app) // define the layout (we probably want couple of these. (one for debugging and one for sign.))
	app.MainContent = content

	w.SetContent(content)

	w.Show()

	a.Run()
}
