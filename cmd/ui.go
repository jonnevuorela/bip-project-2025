package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

/**
 * Define a Layout
 * @param *App pointer to the app struct
 * @return CanvasObject to the App.MainContent. This sets just the layout
 */
func CreateLayout(app *App) fyne.CanvasObject {

	leftPanel := container.NewVBox(
		app.LeftCanvas,
	)
	mainContent := container.NewVBox(
		app.ContentCanvas,
	)

	layout := container.NewBorder(
		nil,         // top
		nil,         // bottom
		leftPanel,   // left
		nil,         // right
		mainContent, // center
	)
	return layout
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
