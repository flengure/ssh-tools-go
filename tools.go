package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/exp/maps"
)

func NewStaticResource(name string, content []byte) *fyne.StaticResource {
	return &fyne.StaticResource{
		StaticName:    name,
		StaticContent: content,
	}
}

func LoadResourceFromPath(path string) (*fyne.StaticResource, error) {
	bytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	name := filepath.Base(path)
	return NewStaticResource(name, bytes), nil
}

type JobsEditor map[string]struct {
	desc *widget.Entry
	file *widget.Entry
	cmd  *widget.Entry
}

func (ui *JobsEditor) GetJobs() Jobs {
	result := Jobs{}
	for k, v := range *ui {
		result[k] = map[string]string{
			"desc": v.desc.Text,
			"file": v.file.Text,
			"cmd":  v.cmd.Text,
		}
	}
	return result
}

type Editor struct {
	menu            *widget.Select
	save            *widget.Button
	view            *widget.Entry
	status          *widget.Label
	progress        *widget.ProgressBarInfinite
	editorConfig    map[string]map[string]string
	conn            conn
	text            string
	err             error
	addConfig       *widget.Button
	delConfig       *widget.Button
	editConfig      *widget.Button
	writeable       bool
	editConfigPopup *widget.PopUp
	desc            *widget.Label
}

func (ui *Editor) hasFile(s string) bool {
	val, ok := ui.editorConfig[s]["file"]
	if ok && val != "" {
		return true
	}
	return false
}

func (ui *Editor) DisableMenu() {
	ui.menu.Disable()
	ui.addConfig.Disable()
	ui.delConfig.Disable()
	ui.editConfig.Disable()
}

func (ui *Editor) EnableMenuControls() {
	ui.addConfig.Enable()
	ui.delConfig.Enable()
	ui.editConfig.Enable()
}

func (ui *Editor) DisableMenuControls() {
	ui.addConfig.Disable()
	ui.delConfig.Disable()
	ui.editConfig.Disable()
}

func (ui *Editor) showMessage(s string) {
	ui.status.SetText(s)
}

func (ui *Editor) showError(s string) {
	ui.err = errors.New(s)
	ui.status.SetText(s)
}

func (ui *Editor) showProgress(s string) {
	ui.showMessage(s)
	ui.progress.Show()
}

func (ui *Editor) hideProgress(s string) {
	ui.showMessage(s)
	ui.progress.Hide()
}

func (ui *Editor) help() string {
	h := "The Editor"
	h += "\n" + strings.Repeat("-", len(h)) + "\n\n"
	h += "Allows editing of the specified remote file\n"
	h += "and runs the specified command on successful save\n\n"
	if ui.editorConfig != nil {
		f := "%-18v %-38v %-24v\n"
		h += fmt.Sprintf(f, "name", "file", "command")
		h += fmt.Sprintf(f, "----", "----", "-------")

		for k, v := range ui.editorConfig {
			h += fmt.Sprintf(f, k, v["path"], v["cmd"])
		}
	}
	return h
}

func NewEditor() *Editor {

	ui := &Editor{
		menu:       widget.NewSelect([]string{}, func(s string) {}),
		save:       widget.NewButton("Save", func() {}),
		view:       widget.NewMultiLineEntry(),
		status:     widget.NewLabel("status..."),
		progress:   widget.NewProgressBarInfinite(),
		conn:       conn{},
		addConfig:  widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {}),
		delConfig:  widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
		editConfig: widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
		writeable:  false,
		desc:       widget.NewLabel(""),
	}

	ui.DisableMenu()
	ui.DisableMenuControls()
	ui.menu.ClearSelected()
	ui.save.Disable()
	ui.view.Disable()
	ui.view.TextStyle = fyne.TextStyle{Monospace: true, TabWidth: 4}
	ui.progress.Hide()
	ui.status.Show()
	ui.editConfig.Disable()
	ui.delConfig.Disable()

	ui.view.OnChanged = func(s string) {
		if s == ui.text {
			ui.save.Disable()
			ui.EnableMenuControls()
		} else {
			ui.save.Enable()
			ui.DisableMenuControls()
		}
	}

	return ui
}

type Tools struct {
	hostEntry     *widget.SelectEntry
	hostDesc      *widget.Entry
	hostDescLabel *widget.Label
	password      *widget.Entry
	connectBtn    *widget.Button
	privateKey    *widget.Entry
	conn          conn
	config        *Config
	editor        *Editor
	viewer        *Editor
	helpStatus    *widget.Label
	helpProgress  *widget.ProgressBarInfinite
	jsonView      *widget.Entry
	jsonText      string
	jsonSave      *widget.Button
	helpView      *widget.RichText
	helpMenu      *widget.Select
	err           error
	delHost       *widget.Button
	menuOpen      *fyne.MenuItem
	editHost      *widget.Button
	editHostPopup *widget.PopUp
}

func (ui *Tools) Connect() error {

	user, host := ParseHostSpecToUserHost(ui.hostEntry.Text)

	ui.conn.host = user + "@" + host
	ui.conn.password = ui.password.Text
	ui.conn.key = ui.privateKey.Text

	ui.showProgress(fmt.Sprintf(
		"connecting to %s as %s...", host, user))

	err := ui.conn.Connect()
	if err != nil {
		ui.hideProgress(fmt.Sprintf(
			"could not dial out to %s as %s\n%s", host, user, err))
		return err
	}

	ui.hostEntry.SetText(ui.conn.host)
	ui.hideProgress(fmt.Sprintf(
		"success: connected to %s as %s", host, user))

	return nil

}

func (ui *Tools) showMessage(s string) {
	ui.editor.status.SetText(s)
	ui.viewer.status.SetText(s)
	ui.helpStatus.SetText(s)
}

func (ui *Tools) showError(s string) {
	ui.editor.showError(s)
	ui.viewer.showError(s)
	ui.helpStatus.SetText(s)
	ui.err = errors.New(s)
}

func (ui *Tools) showProgress(s string) {
	ui.editor.showProgress(s)
	ui.viewer.showProgress(s)
	ui.helpStatus.SetText(s)
	ui.helpProgress.Show()
}

func (ui *Tools) hideProgress(s string) {
	ui.editor.hideProgress(s)
	ui.viewer.hideProgress(s)
	ui.helpStatus.SetText(s)
	ui.helpProgress.Hide()
}

func (ui *Tools) SetHostConfig(host string) {

	ui.config.Host = host

	ui.editor.editorConfig = ui.config.Hosts[host].Editors
	ui.viewer.editorConfig = ui.config.Hosts[host].Viewers

	// is the new key typed in our map if yes
	// change the editor and view select options for new host

	if _, ok := ui.config.Hosts[host]; ok {

		ui.editor.menu.Options = maps.Keys(ui.editor.editorConfig)
		ui.viewer.menu.Options = maps.Keys(ui.viewer.editorConfig)
		ui.hostDesc.SetText(ui.config.Hosts[host].Desc)

	}

}

func (ui *Tools) SetHostConn(s string) {
	ui.conn.host = s
	ui.editor.conn.host = s
	ui.viewer.conn.host = s
}

// func (ui *Tools) SetHasSsh(s bool) {
// 	ui.conn.has_ssh = s
// 	ui.editor.conn.has_ssh = s
// 	ui.viewer.conn.has_ssh = s
// }

// func (ui *Tools) SetHasScp(s bool) {
// 	ui.conn.has_scp = s
// 	ui.editor.conn.has_scp = s
// 	ui.viewer.conn.has_scp = s
// }

func (ui *Tools) SetUiConnected() {
	ui.connectBtn.Disable()
	ui.editor.menu.Enable()
	ui.viewer.menu.Enable()
	ui.connectBtn.SetText(ui.conn.host)
}

func (ui *Tools) SetConnected(s string) {
	ui.SetHostConn(s)
	ui.SetUiConnected()
}

func (ui *Tools) SetUiNotConnected() {
	ui.connectBtn.Enable()
	ui.connectBtn.SetText("Connect")
	// ui.editor.menu.Disable()
	// ui.viewer.menu.Disable()
}

func (ui *Tools) SetNotConnected() {
	ui.SetHostConn("")
	ui.SetUiNotConnected()
}

// func (ui *Tools) SetSsh(s *ssh.Client) {
// 	ui.conn.ssh = s
// 	ui.editor.conn.ssh = s
// 	ui.viewer.conn.ssh = s
// 	// ui.SetHasSsh(true)
// }

// func (ui *Tools) SetScp(s *scp.Client) {
// 	ui.conn.scp = s
// 	ui.editor.conn.scp = s
// 	ui.viewer.conn.scp = s
// 	// ui.SetHasScp(true)
// }

func (ui *Tools) SetConfig(config *Config) {
	ui.config = config
	// ui.hostEntry = widget.NewSelectEntry(maps.Keys(ui.config.Hosts))
	ui.hostEntry.SetOptions(maps.Keys(ui.config.Hosts))
	ui.hostEntry.SetText(ui.config.DefaultHost())
}

func (ui *Tools) editJob(e *Editor) {

	label1 := widget.NewLabel("Name")
	value1 := widget.NewEntry()
	label2 := widget.NewLabel("Description")
	value2 := widget.NewEntry()
	label3 := widget.NewLabel("File")
	value3 := widget.NewEntry()
	label4 := widget.NewLabel("Command")
	value4 := widget.NewEntry()
	okButton := widget.NewButton("OK", func() {
		e.editorConfig[value1.Text] = Job{
			"desc": value2.Text,
			"file": value3.Text,
			"cmd":  value4.Text,
		}
		// Editors: ui.editorConfig.Hosts[ui.hostEntry.Text].Editors,
		// Viewers: ui.editorConfig.Hosts[ui.hostEntry.Text].Viewers,
		// }

		ui.config.Save()

		e.editConfigPopup.Hide()
	})
	cancelButton := widget.NewButton("Cancel", func() {
		e.editConfigPopup.Hide()
	})

	value1.SetText(e.menu.Selected)
	value2.SetText(e.editorConfig[e.menu.Selected]["desc"])
	value3.SetText(e.editorConfig[e.menu.Selected]["file"])
	value4.SetText(e.editorConfig[e.menu.Selected]["cmd"])
	value2.MultiLine = true
	value2.Wrapping = fyne.TextWrapBreak
	value4.MultiLine = true
	value4.Wrapping = fyne.TextWrapBreak

	var form1 fyne.CanvasObject

	if e.writeable {
		form1 = container.New(layout.NewFormLayout(),
			label1, value1, label2, value2, label3, value3)
	} else {
		form1 = container.New(layout.NewFormLayout(),
			label1, value1, label2, value2)
	}

	form2 := container.NewBorder(label4, nil, nil, nil,
		value4)

	buttons := container.NewGridWithColumns(2, cancelButton, okButton)
	cont := container.NewBorder(form1, buttons, nil, nil, form2)

	e.editConfigPopup = widget.NewModalPopUp(cont, myWindow.Canvas())
	e.editConfigPopup.Resize(fyne.NewSize(400, 300))
	e.editConfigPopup.Show()

}

func (ui *Tools) runJob(e *Editor, s string) {

	if e.hasFile(s) {

		var err error

		e.showProgress("Attempting to load remote file...")

		// No host we probably have no client remember to set this
		if e.conn.host == "" {
			error_text := "fail: No ssh client"
			e.showError(error_text)
			e.progress.Hide()
			return
		}

		e.text, err = ui.conn.get_content(e.editorConfig[s]["file"])
		if err != nil {
			error_text := fmt.Sprintf(
				"fail: scp %s : %s", e.editorConfig[s]["file"], err.Error())
			e.showError(error_text)
			e.progress.Hide()
			return
		}

		e.view.SetText(e.text)
		// e.desc.SetText(e.editorConfig[s]["desc"])
		e.view.Enable()
		e.EnableMenuControls()
		e.save.Disable()

		e.hideProgress(fmt.Sprintf("success: scp %s", e.editorConfig[s]["file"]))

	} else {

		e.showProgress("Attempting to run remote commands...")

		// No host we probably have no client remember to set this
		if e.conn.host == "" {
			e.showError("host not set, probably no client connection")
			return
		}

		result, err := ui.conn.output(e.editorConfig[s]["cmd"])
		if err != nil {
			e.err = err
			err_text := fmt.Sprintf("failed: \"%s\"", e.editorConfig[s]["cmd"])
			e.showError(err_text)
			e.hideProgress(err_text)
			e.view.SetText("")
			e.view.Disable()
			return
		}

		e.view.SetText(result)
		e.view.Enable()
		e.EnableMenuControls()
		e.hideProgress("command ran successfully")

	}

}

func (ui *Tools) saveJob(e *Editor) {

	// var err error

	if !(e.writeable && e.hasFile(e.menu.Selected)) {
		return
	}

	e.showProgress("Attempting to save remote file...")

	// No host we probably have no client remember to set this
	if e.conn.host == "" {
		error_text := "fail: No ssh client"
		e.showError(error_text)
		e.hideProgress(error_text)
		return
	}

	err := e.conn.set_content(e.view.Text, e.editorConfig[e.menu.Selected]["file"])
	if err != nil {
		error_text := "failed: set_content: " + err.Error()
		e.showError(error_text)
		e.hideProgress(error_text)
		return
	}

	e.text = e.view.Text

	e.showMessage(fmt.Sprintf("successfully saved \"%s\"", e.editorConfig[e.menu.Selected]["file"]))

	if val, ok := e.editorConfig[e.menu.Selected]["cmd"]; ok {

		// run command associated with saving file
		err = e.conn.run(val)
		if err != nil {
			error_text := fmt.Sprintf(
				"failed running \"%s\"", val)
			e.showError(error_text)
			e.hideProgress(error_text)
		}
		e.hideProgress(fmt.Sprintf(
			"success: saved \"%s\" and ran \"%s\"",
			e.editorConfig[e.menu.Selected]["file"],
			val,
		))

	}

	e.hideProgress(fmt.Sprintf(
		"success: saved \"%s\"",
		e.editorConfig[e.menu.Selected]["file"],
	))

}

func NewTools() Tools {

	ui := Tools{
		hostEntry:     widget.NewSelectEntry([]string{}),
		hostDesc:      widget.NewEntry(),
		hostDescLabel: widget.NewLabel(""),
		password:      widget.NewPasswordEntry(),
		connectBtn:    widget.NewButton("Connect", func() {}),
		privateKey:    widget.NewEntry(),
		editor:        NewEditor(),
		viewer:        NewEditor(),
		helpView:      widget.NewRichTextFromMarkdown("* Text"),
		helpStatus:    widget.NewLabel("status..."),
		helpProgress:  widget.NewProgressBarInfinite(),
		jsonView:      widget.NewMultiLineEntry(),
		jsonSave:      widget.NewButton("Save", func() {}),
		config:        NewConfigAcl(),
		delHost:       widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
		menuOpen:      fyne.NewMenuItem("Open", nil),
		editHost:      widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
	}

	ui.editor.writeable = true

	config, err := LoadConfigFrom("config.json")
	if err != nil {
		ui.SetConfig(NewConfigNomic())
	} else {
		ui.SetConfig(&config)
	}

	// fmt.Println(config.host.Text)

	ui.SetHostConfig(config.DefaultHost())

	ui.hostDesc.SetText(ui.config.Hosts[ui.config.Host].Desc)

	// fmt.Println(ui.editor.menu.Options)

	ui.hostEntry.PlaceHolder = "user@host:port"
	ui.password.PlaceHolder = "password"

	// ui.helpView.TextStyle = fyne.TextStyle{Monospace: true}
	ui.jsonView.TextStyle = fyne.TextStyle{Monospace: true}

	ui.helpProgress.Hide()
	ui.jsonSave.Hide()

	ui.SetNotConnected()

	ui.hostEntry.OnChanged = func(s string) {

		if ui.conn.host == "" || ui.conn.host != s {

			if ui.editor.view.Text != "" {
				ui.editor.text = ui.editor.view.Text
			}

			if ui.viewer.view.Text != "" {
				ui.viewer.text = ui.viewer.view.Text
			}

			ui.viewer.view.SetText("")
			ui.viewer.view.SetText("")
			ui.connectBtn.SetText("Connect")
			ui.connectBtn.Enable()

		} else {

			ui.viewer.view.SetText(ui.editor.text)
			ui.viewer.view.SetText(ui.viewer.text)
			ui.connectBtn.SetText(ui.conn.host)
			ui.connectBtn.Disable()

		}

		// ui.editor.menu.Disable()
		// ui.editor.addConfig.Disable()
		// ui.editor.delConfig.Disable()
		// ui.editor.editConfig.Disable()
		// ui.viewer.menu.Disable()
		// ui.viewer.addConfig.Disable()
		// ui.viewer.delConfig.Disable()
		// ui.viewer.editConfig.Disable()

		// fmt.Println(s)
		// fmt.Println(ui.config.Hosts[s])
		ui.SetHostConfig(s)

	}

	ui.hostDesc.OnChanged = func(s string) {
		ui.config.Hosts[ui.config.Host] = Host{
			Desc:    s,
			Editors: ui.config.Hosts[ui.config.Host].Editors,
			Viewers: ui.config.Hosts[ui.config.Host].Viewers,
		}
	}

	ui.menuOpen.Action = func() {
		// fmt.Println("button tapped")
		onChosen := func(f fyne.URIReadCloser, err error) {
			if err != nil {
				fmt.Println(err)
				return
			}
			if f == nil {
				return
			}
			file := f.URI().Path()
			// fmt.Printf("chosen: %s\n", f.URI().Path())
			conf, err := LoadConfigFrom(file)
			if err != nil {
				fmt.Println(err)
				return
			}

			ui.SetConfig(&conf)
			ui.config.File = file
			ui.SetHostConfig(ui.hostEntry.Text)

			// fmt.Println(ui.hostEntry.Text)

		}

		dialog.ShowFileOpen(onChosen, myWindow)
	}

	ui.editHost.OnTapped = func() {

		label1 := widget.NewLabel("Host Specification")
		value1 := widget.NewLabel(ui.hostEntry.Text)
		label2 := widget.NewLabel("Description")
		value2 := widget.NewEntry()
		okButton := widget.NewButton("OK", func() {
			ui.config.Hosts[ui.hostEntry.Text] = Host{
				Desc:    value2.Text,
				Editors: ui.config.Hosts[ui.hostEntry.Text].Editors,
				Viewers: ui.config.Hosts[ui.hostEntry.Text].Viewers,
			}
			ui.editHostPopup.Hide()
		})
		cancelButton := widget.NewButton("Cancel", func() {
			ui.editHostPopup.Hide()
		})

		value2.SetText(ui.config.Hosts[ui.hostEntry.Text].Desc)
		grid := container.New(layout.NewFormLayout(), label1, value1, label2, value2)
		cont := container.NewVBox(
			grid,
			container.NewGridWithColumns(2,
				layout.NewSpacer(),
				container.NewGridWithColumns(2,
					cancelButton,
					okButton,
				),
			),
		)

		ui.editHostPopup = widget.NewModalPopUp(cont, myWindow.Canvas())
		ui.editHostPopup.Show()

	}

	ui.connectBtn.OnTapped = func() {

		_ = ui.Connect()
		user, host := ParseHostSpecToUserHost(ui.hostEntry.Text)

		// fmt.Println()
		ui.config.Hosts[ui.hostEntry.Text] = ui.config.Hosts[ui.config.Host]

		ui.SetHostConfig(ui.hostEntry.Text)
		ui.SetHostConn(ui.hostEntry.Text)

		ui.hostEntry.SetOptions(maps.Keys(ui.config.Hosts))

		// fmt.Println((ui.config))

		ui.editor.menu.ClearSelected()
		ui.viewer.menu.ClearSelected()

		ui.editor.view.SetText("")
		ui.viewer.view.SetText("")

		// ui.editor.menu.Disable()
		ui.editor.addConfig.Disable()
		ui.editor.delConfig.Disable()
		ui.editor.editConfig.Disable()
		// ui.viewer.menu.Disable()
		ui.viewer.addConfig.Disable()
		ui.viewer.delConfig.Disable()
		ui.viewer.editConfig.Disable()

		ui.config.Save()

		ui.SetConnected(ui.hostEntry.Text)
		ui.showMessage(fmt.Sprintf(
			"successfully connected as %s to %s", user, host))

		ui.hideProgress(fmt.Sprintf(
			"success: connected to %s as %s", host, user))
	}

	ui.helpMenu = widget.NewSelect(
		[]string{"editor", "viewer", "config"},
		func(s string) {

			if s == "editor" {
				ui.helpView.ParseMarkdown(ui.editor.help())
				ui.jsonView.Hide()
				ui.helpView.Show()
				ui.jsonSave.Hide()
				return
			}
			if s == "viewer" {
				ui.helpView.Wrapping = fyne.TextWrapBreak
				// ui.helpView.Scroll = container.ScrollVerticalOnly
				ui.helpView.ParseMarkdown(ui.viewer.help())
				ui.jsonView.Hide()
				ui.helpView.Show()
				ui.jsonSave.Hide()
				return
			}
			if s == "config" {
				ui.jsonText, _ = ui.config.Json()
				ui.jsonView.SetText(ui.jsonText)
				ui.helpView.Hide()
				ui.jsonView.Show()
				ui.jsonSave.Show()
				return
			}

		},
	)

	ui.jsonView.OnChanged = func(s string) {

		// Disable the Save button if the text has not changed
		// Enable if it has changed

		if s == ui.jsonText {

			ui.jsonSave.Disable()

		} else {

			ui.jsonSave.Enable()

		}

	}

	ui.jsonSave.OnTapped = func() {

		// Save the json config settings to file
		// ui.jsonEntry.Text is the text to be saved
		// ui.jsonText is the text before editing started
		// ui.config.File is our output file

		// ToDo: Check if the file corresponds with our data structure
		// before saving

		// convert the string to byte stream after appending a newline
		b := []byte(fmt.Sprintf("%s\n", ui.jsonView.Text))

		// write the byte stream to file
		err := os.WriteFile(ui.config.File, b, 0644)
		if err != nil {

			ui.showError(fmt.Sprintf(
				"%s: %s", "fail: saving configuration", err.Error()))

			return
		}

		ui.jsonText = ui.jsonView.Text
		ui.jsonView.OnChanged(ui.jsonView.Text)

	}

	ui.editor.editConfig.OnTapped = func() { ui.editJob(ui.editor) }
	ui.viewer.editConfig.OnTapped = func() { ui.editJob(ui.viewer) }

	ui.editor.menu.OnChanged = func(s string) {
		ui.editor.desc.SetText(ui.editor.editorConfig[s]["desc"])
		ui.runJob(ui.editor, s)
	}
	ui.viewer.menu.OnChanged = func(s string) {
		ui.viewer.desc.SetText(ui.viewer.editorConfig[s]["desc"])
		ui.runJob(ui.viewer, s)
	}

	ui.editor.save.OnTapped = func() { ui.saveJob(ui.editor) }

	return ui
}
