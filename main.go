package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

var myApp = app.New()
var myWindow = myApp.NewWindow("ssh tools")

func main() {

	var ui = NewTools()
	// var modal *widget.PopUp

	// ui.app = &a

	//ui.window = ui.app.NewWindow("ssh tools")
	myWindow.Resize(fyne.NewSize(660, 600))

	r, _ := LoadResourceFromPath("icon.png")
	myWindow.SetIcon(r)
	// ui.window.Resize(fyne.NewSize(660, 600))

	// ui.app = &myApp
	// ui.window = &myWindow

	// ui.connectBtn.Resize(fyne.NewSize(
	// 	ui.window.Canvas().Size().Width/6,
	// 	ui.connectBtn.MinSize().Height,
	// ))
	fmt.Println(myApp.Metadata().ID)

	menuItem1 := fyne.NewMenuItem("New", nil)
	menuItem3 := fyne.NewMenuItem("Save", nil)
	// menuItem4 := fyne.NewMenuItem("Exit", nil)
	newMenu1 := fyne.NewMenu("File",
		menuItem1, ui.menuOpen, menuItem3)
	menu := fyne.NewMainMenu(newMenu1)
	myWindow.SetMainMenu(menu)

	// hdr := container.NewHBox(
	// 	ui.editHost,
	// 	ui.delHost,
	// )
	gui := container.NewBorder(
		container.NewPadded(
			container.NewVBox(
				container.NewGridWithColumns(3,
					ui.hostEntry,
					ui.password,
					ui.connectBtn,
				),
				container.NewMax(
					ui.hostDescLabel,
					ui.hostDesc,
				),
			),
		),
		nil,
		nil,
		nil,
		container.NewAppTabs(
			container.NewTabItem(
				"Editor",
				container.NewPadded(
					container.NewBorder(
						container.NewVBox(
							container.NewGridWithColumns(2,
								container.NewBorder(nil, nil, nil,
									container.NewHBox(
										ui.editor.editConfig,
										ui.editor.delConfig,
										// ui.editor.addConfig,
									),
									ui.editor.menu,
								),
								container.NewGridWithColumns(2,
									layout.NewSpacer(),
									ui.editor.save,
								),
								ui.editor.desc,
							),
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.editor.status,
								ui.editor.progress,
							),
						),
						nil,
						nil,
						ui.editor.view,
					),
				),
			),
			container.NewTabItem(
				"Viewer",
				container.NewPadded(
					container.NewBorder(
						container.NewVBox(
							container.NewGridWithColumns(2,
								container.NewBorder(nil, nil, nil,
									container.NewHBox(
										ui.viewer.editConfig,
										ui.viewer.delConfig,
										// ui.viewer.addConfig,
									),
									ui.viewer.menu,
								),
								container.NewGridWithColumns(2,
									layout.NewSpacer(),
									layout.NewSpacer(),
								),
								ui.viewer.desc,
							),
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.viewer.status,
								ui.viewer.progress,
							),
						),
						nil,
						nil,
						ui.viewer.view,
					),
				),
			),
			container.NewTabItem(
				"help",
				container.NewPadded(
					container.NewBorder(
						container.NewGridWithColumns(
							3,
							ui.helpMenu,
							layout.NewSpacer(),
							ui.jsonSave,
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.helpStatus,
								ui.helpProgress,
							),
						),
						nil,
						nil,
						container.NewMax(
							container.NewVScroll(ui.helpView),
							ui.jsonView,
						),
					),
				),
			),
		),
	)

	myWindow.SetContent(gui)

	// ui.window = w

	myWindow.ShowAndRun()
	// ui.window = myWindow

}
