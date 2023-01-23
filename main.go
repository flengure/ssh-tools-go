package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/flengure/ssh-tools-go/tools"
)

// var myApp = app.New()

// var myWindow = myApp.NewWindow("ssh tools")

func main() {

	var ui = tools.NewTools()
	ui.Window.Resize(fyne.NewSize(660, 600))

	r, _ := tools.LoadResourceFromPath("icon.png")
	ui.Window.SetIcon(r)

	menuItem1 := fyne.NewMenuItem("New", nil)
	menuItem3 := fyne.NewMenuItem("Save", nil)
	// menuItem4 := fyne.NewMenuItem("Exit", nil)
	newMenu1 := fyne.NewMenu("File",
		menuItem1, ui.MenuOpen, menuItem3)
	menu := fyne.NewMainMenu(newMenu1)
	ui.Window.SetMainMenu(menu)

	gui := container.NewBorder(
		container.NewPadded(
			container.NewVBox(
				container.NewGridWithColumns(3,
					ui.HostEntry,
					ui.Password,
					ui.ConnectBtn,
				),
				container.NewMax(
					ui.HostDescLabel,
					ui.HostDesc,
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
										ui.Editor.EditConfig,
										ui.Editor.DelConfig,
										// ui.Editor.addConfig,
									),
									ui.Editor.Menu,
								),
								container.NewGridWithColumns(2,
									layout.NewSpacer(),
									ui.Editor.Save,
								),
								ui.Editor.Desc,
							),
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.Editor.Status,
								ui.Editor.Progress,
							),
						),
						nil,
						nil,
						ui.Editor.View,
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
										ui.Viewer.EditConfig,
										ui.Viewer.DelConfig,
										// ui.Viewer.addConfig,
									),
									ui.Viewer.Menu,
								),
								container.NewGridWithColumns(2,
									layout.NewSpacer(),
									layout.NewSpacer(),
								),
								ui.Viewer.Desc,
							),
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.Viewer.Status,
								ui.Viewer.Progress,
							),
						),
						nil,
						nil,
						ui.Viewer.View,
					),
				),
			),
			container.NewTabItem(
				"help",
				container.NewPadded(
					container.NewBorder(
						container.NewGridWithColumns(
							3,
							ui.HelpMenu,
							layout.NewSpacer(),
							ui.JsonSave,
						),
						container.NewGridWrap(
							fyne.NewSize(660, 33),
							container.NewMax(
								ui.HelpStatus,
								ui.HelpProgress,
							),
						),
						nil,
						nil,
						container.NewMax(
							container.NewVScroll(ui.HelpView),
							ui.JsonView,
						),
					),
				),
			),
		),
	)

	ui.Window.SetContent(gui)

	// ui.window = w

	ui.Window.ShowAndRun()
	// ui.window = myWindow

}
