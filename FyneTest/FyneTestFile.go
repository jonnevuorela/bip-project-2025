package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Smart Street Sign")

	speedSign := canvas.NewImageFromFile("100Speed.png")
	speedSign.FillMode = canvas.ImageFillContain

	WarningSign := canvas.NewImageFromFile("Blank.png")
	WarningSign.FillMode = canvas.ImageFillContain

	tabs := container.NewAppTabs(
		container.NewTabItem("Signs", container.New(layout.NewGridLayout(3), speedSign, WarningSign)),
		container.NewTabItem("Debug", widget.NewLabel(":D")),
	)

	tabs.SetTabLocation(container.TabLocationLeading)
	myWindow.SetContent(tabs)
	myWindow.Show()

	// Start goroutine to swap the image inside the tab
	go func() {
		for {
			time.Sleep(5 * time.Second)
			fyne.Do(func() {
				speedSign.File = "50Speed.png" // Switch to new image
				WarningSign.File = "WarningSlipRoad.png"
				speedSign.Refresh() // Force UI refresh
				WarningSign.Refresh()
			})

			time.Sleep(5 * time.Second)
			fyne.Do(func() {
				speedSign.File = "100Speed.png" // Switch back
				WarningSign.File = "Blank.png"
				speedSign.Refresh()
				WarningSign.Refresh()
			})
		}
	}()

	myApp.Run()
}
