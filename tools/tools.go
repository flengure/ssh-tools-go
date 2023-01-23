package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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
	Desc *widget.Entry
	file *widget.Entry
	cmd  *widget.Entry
}

func (ui *JobsEditor) GetJobs() Jobs {
	result := Jobs{}
	for k, v := range *ui {
		result[k] = map[string]string{
			"Desc": v.Desc.Text,
			"file": v.file.Text,
			"cmd":  v.cmd.Text,
		}
	}
	return result
}

type Editor struct {
	Menu            *widget.Select
	Save            *widget.Button
	View            *widget.Entry
	Status          *widget.Label
	Progress        *widget.ProgressBarInfinite
	EditorConfig    map[string]map[string]string
	conn            conn
	text            string
	err             error
	addConfig       *widget.Button
	DelConfig       *widget.Button
	EditConfig      *widget.Button
	writeable       bool
	editConfigPopup *widget.PopUp
	Desc            *widget.Label
}

func (ui *Editor) hasFile(s string) bool {
	val, ok := ui.EditorConfig[s]["file"]
	if ok && val != "" {
		return true
	}
	return false
}

func (ui *Editor) DisableMenu() {
	ui.Menu.Disable()
	ui.addConfig.Disable()
	ui.DelConfig.Disable()
	ui.EditConfig.Disable()
}

func (ui *Editor) EnableMenuControls() {
	ui.addConfig.Enable()
	ui.DelConfig.Enable()
	ui.EditConfig.Enable()
}

func (ui *Editor) DisableMenuControls() {
	ui.addConfig.Disable()
	ui.DelConfig.Disable()
	ui.EditConfig.Disable()
}

func (ui *Editor) showMessage(s string) {
	ui.Status.SetText(s)
}

func (ui *Editor) showError(s string) {
	ui.err = errors.New(s)
	ui.Status.SetText(s)
}

func (ui *Editor) showProgress(s string) {
	ui.showMessage(s)
	ui.Progress.Show()
}

func (ui *Editor) hideProgress(s string) {
	ui.showMessage(s)
	ui.Progress.Hide()
}

func (ui *Editor) help() string {
	h := "The Editor"
	h += "\n" + strings.Repeat("-", len(h)) + "\n\n"
	h += "Allows editing of the specified remote file\n"
	h += "and runs the specified command on successful save\n\n"
	if ui.EditorConfig != nil {
		f := "%-18v %-38v %-24v\n"
		h += fmt.Sprintf(f, "name", "file", "command")
		h += fmt.Sprintf(f, "----", "----", "-------")

		for k, v := range ui.EditorConfig {
			h += fmt.Sprintf(f, k, v["path"], v["cmd"])
		}
	}
	return h
}

func NewEditor() *Editor {

	ui := &Editor{
		Menu:       widget.NewSelect([]string{}, func(s string) {}),
		Save:       widget.NewButton("Save", func() {}),
		View:       widget.NewMultiLineEntry(),
		Status:     widget.NewLabel("Status..."),
		Progress:   widget.NewProgressBarInfinite(),
		conn:       conn{},
		addConfig:  widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {}),
		DelConfig:  widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
		EditConfig: widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
		writeable:  false,
		Desc:       widget.NewLabel(""),
	}

	ui.DisableMenu()
	ui.DisableMenuControls()
	ui.Menu.ClearSelected()
	ui.Save.Disable()
	ui.View.Disable()
	ui.View.TextStyle = fyne.TextStyle{Monospace: true, TabWidth: 4}
	ui.Progress.Hide()
	ui.Status.Show()
	ui.EditConfig.Disable()
	ui.DelConfig.Disable()

	ui.View.OnChanged = func(s string) {
		if s == ui.text {
			ui.Save.Disable()
			ui.EnableMenuControls()
		} else {
			ui.Save.Enable()
			ui.DisableMenuControls()
		}
	}

	return ui
}

type Tools struct {
	HostEntry     *widget.SelectEntry
	HostDesc      *widget.Entry
	HostDescLabel *widget.Label
	Password      *widget.Entry
	ConnectBtn    *widget.Button
	PrivateKey    *widget.Entry
	conn          conn
	config        *Config
	Editor        *Editor
	Viewer        *Editor
	HelpStatus    *widget.Label
	HelpProgress  *widget.ProgressBarInfinite
	JsonView      *widget.Entry
	jsonText      string
	JsonSave      *widget.Button
	HelpView      *widget.RichText
	HelpMenu      *widget.Select
	err           error
	delHost       *widget.Button
	MenuOpen      *fyne.MenuItem
	editHost      *widget.Button
	editHostPopup *widget.PopUp
	App           fyne.App
	Window        fyne.Window
}

func (ui *Tools) Connect() error {

	user, host := ParseHostSpecToUserHost(ui.HostEntry.Text)

	ui.conn.host = user + "@" + host
	ui.conn.password = ui.Password.Text
	ui.conn.key = ui.PrivateKey.Text

	ui.showProgress(fmt.Sprintf(
		"connecting to %s as %s...", host, user))

	err := ui.conn.Connect()
	if err != nil {
		ui.hideProgress(fmt.Sprintf(
			"could not dial out to %s as %s\n%s", host, user, err))
		return err
	}

	ui.HostEntry.SetText(ui.conn.host)
	ui.hideProgress(fmt.Sprintf(
		"success: connected to %s as %s", host, user))

	return nil

}

func (ui *Tools) showMessage(s string) {
	ui.Editor.Status.SetText(s)
	ui.Viewer.Status.SetText(s)
	ui.HelpStatus.SetText(s)
}

func (ui *Tools) showError(s string) {
	ui.Editor.showError(s)
	ui.Viewer.showError(s)
	ui.HelpStatus.SetText(s)
	ui.err = errors.New(s)
}

func (ui *Tools) showProgress(s string) {
	ui.Editor.showProgress(s)
	ui.Viewer.showProgress(s)
	ui.HelpStatus.SetText(s)
	ui.HelpProgress.Show()
}

func (ui *Tools) hideProgress(s string) {
	ui.Editor.hideProgress(s)
	ui.Viewer.hideProgress(s)
	ui.HelpStatus.SetText(s)
	ui.HelpProgress.Hide()
}

func (ui *Tools) SetHostConfig(host string) {

	ui.config.Host = host

	ui.Editor.EditorConfig = ui.config.Hosts[host].Editors
	ui.Viewer.EditorConfig = ui.config.Hosts[host].Viewers

	// is the new key typed in our map if yes
	// change the editor and View select options for new host

	if _, ok := ui.config.Hosts[host]; ok {

		ui.Editor.Menu.Options = maps.Keys(ui.Editor.EditorConfig)
		ui.Viewer.Menu.Options = maps.Keys(ui.Viewer.EditorConfig)
		ui.HostDesc.SetText(ui.config.Hosts[host].Desc)

	}

}

func (ui *Tools) SetHostConn(s string) {
	ui.conn.host = s
	ui.Editor.conn.host = s
	ui.Viewer.conn.host = s
}

// func (ui *Tools) SetHasSsh(s bool) {
// 	ui.conn.has_ssh = s
// 	ui.Editor.conn.has_ssh = s
// 	ui.Viewer.conn.has_ssh = s
// }

// func (ui *Tools) SetHasScp(s bool) {
// 	ui.conn.has_scp = s
// 	ui.Editor.conn.has_scp = s
// 	ui.Viewer.conn.has_scp = s
// }

func (ui *Tools) SetUiConnected() {
	ui.ConnectBtn.Disable()
	ui.Editor.Menu.Enable()
	ui.Viewer.Menu.Enable()
	ui.ConnectBtn.SetText(ui.conn.host)
}

func (ui *Tools) SetConnected(s string) {
	ui.SetHostConn(s)
	ui.SetUiConnected()
}

func (ui *Tools) SetUiNotConnected() {
	ui.ConnectBtn.Enable()
	ui.ConnectBtn.SetText("Connect")
	// ui.Editor.Menu.Disable()
	// ui.Viewer.Menu.Disable()
}

func (ui *Tools) SetNotConnected() {
	ui.SetHostConn("")
	ui.SetUiNotConnected()
}

// func (ui *Tools) SetSsh(s *ssh.Client) {
// 	ui.conn.ssh = s
// 	ui.Editor.conn.ssh = s
// 	ui.Viewer.conn.ssh = s
// 	// ui.SetHasSsh(true)
// }

// func (ui *Tools) SetScp(s *scp.Client) {
// 	ui.conn.scp = s
// 	ui.Editor.conn.scp = s
// 	ui.Viewer.conn.scp = s
// 	// ui.SetHasScp(true)
// }

func (ui *Tools) SetConfig(config *Config) {
	ui.config = config
	// ui.HostEntry = widget.NewSelectEntry(maps.Keys(ui.config.Hosts))
	ui.HostEntry.SetOptions(maps.Keys(ui.config.Hosts))
	ui.HostEntry.SetText(ui.config.DefaultHost())
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
		e.EditorConfig[value1.Text] = Job{
			"Desc": value2.Text,
			"file": value3.Text,
			"cmd":  value4.Text,
		}
		// Editors: ui.EditorConfig.Hosts[ui.HostEntry.Text].Editors,
		// Viewers: ui.EditorConfig.Hosts[ui.HostEntry.Text].Viewers,
		// }

		ui.config.Save()

		e.editConfigPopup.Hide()
	})
	cancelButton := widget.NewButton("Cancel", func() {
		e.editConfigPopup.Hide()
	})

	value1.SetText(e.Menu.Selected)
	value2.SetText(e.EditorConfig[e.Menu.Selected]["Desc"])
	value3.SetText(e.EditorConfig[e.Menu.Selected]["file"])
	value4.SetText(e.EditorConfig[e.Menu.Selected]["cmd"])
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

	e.editConfigPopup = widget.NewModalPopUp(cont, ui.Window.Canvas())
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
			e.Progress.Hide()
			return
		}

		e.text, err = ui.conn.get_content(e.EditorConfig[s]["file"])
		if err != nil {
			error_text := fmt.Sprintf(
				"fail: scp %s : %s", e.EditorConfig[s]["file"], err.Error())
			e.showError(error_text)
			e.Progress.Hide()
			return
		}

		e.View.SetText(e.text)
		// e.Desc.SetText(e.EditorConfig[s]["Desc"])
		e.View.Enable()
		e.EnableMenuControls()
		e.Save.Disable()

		e.hideProgress(fmt.Sprintf("success: scp %s", e.EditorConfig[s]["file"]))

	} else {

		e.showProgress("Attempting to run remote commands...")

		// No host we probably have no client remember to set this
		if e.conn.host == "" {
			e.showError("host not set, probably no client connection")
			return
		}

		result, err := ui.conn.output(e.EditorConfig[s]["cmd"])
		if err != nil {
			e.err = err
			err_text := fmt.Sprintf("failed: \"%s\"", e.EditorConfig[s]["cmd"])
			e.showError(err_text)
			e.hideProgress(err_text)
			e.View.SetText("")
			e.View.Disable()
			return
		}

		e.View.SetText(result)
		e.View.Enable()
		e.EnableMenuControls()
		e.hideProgress("command ran successfully")

	}

}

func (ui *Tools) saveJob(e *Editor) {

	// var err error

	if !(e.writeable && e.hasFile(e.Menu.Selected)) {
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

	err := e.conn.set_content(e.View.Text, e.EditorConfig[e.Menu.Selected]["file"])
	if err != nil {
		error_text := "failed: set_content: " + err.Error()
		e.showError(error_text)
		e.hideProgress(error_text)
		return
	}

	e.text = e.View.Text

	e.showMessage(fmt.Sprintf("successfully saved \"%s\"", e.EditorConfig[e.Menu.Selected]["file"]))

	if val, ok := e.EditorConfig[e.Menu.Selected]["cmd"]; ok {

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
			e.EditorConfig[e.Menu.Selected]["file"],
			val,
		))

	}

	e.hideProgress(fmt.Sprintf(
		"success: saved \"%s\"",
		e.EditorConfig[e.Menu.Selected]["file"],
	))

}

func NewTools() Tools {

	ui := Tools{
		App:           app.New(),
		HostEntry:     widget.NewSelectEntry([]string{}),
		HostDesc:      widget.NewEntry(),
		HostDescLabel: widget.NewLabel(""),
		Password:      widget.NewPasswordEntry(),
		ConnectBtn:    widget.NewButton("Connect", func() {}),
		PrivateKey:    widget.NewEntry(),
		HelpView:      widget.NewRichTextFromMarkdown("* Text"),
		HelpStatus:    widget.NewLabel("Status..."),
		HelpProgress:  widget.NewProgressBarInfinite(),
		JsonView:      widget.NewMultiLineEntry(),
		JsonSave:      widget.NewButton("Save", func() {}),
		config:        NewConfigAcl(),
		delHost:       widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
		MenuOpen:      fyne.NewMenuItem("Open", nil),
		editHost:      widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
	}

	// ui.App = app.New()
	ui.Window = ui.App.NewWindow("ssh tools")
	ui.Editor = NewEditor()
	ui.Viewer = NewEditor()

	ui.Editor.writeable = true

	config, err := LoadConfigFrom("config.json")
	if err != nil {
		ui.SetConfig(NewConfigNomic())
	} else {
		ui.SetConfig(&config)
	}

	// fmt.Println(config.host.Text)

	ui.SetHostConfig(config.DefaultHost())

	ui.HostDesc.SetText(ui.config.Hosts[ui.config.Host].Desc)

	// fmt.Println(ui.Editor.Menu.Options)

	ui.HostEntry.PlaceHolder = "user@host:port"
	ui.Password.PlaceHolder = "password"

	// ui.HelpView.TextStyle = fyne.TextStyle{Monospace: true}
	ui.JsonView.TextStyle = fyne.TextStyle{Monospace: true}

	ui.HelpProgress.Hide()
	ui.JsonSave.Hide()

	ui.SetNotConnected()

	ui.HostEntry.OnChanged = func(s string) {

		if ui.conn.host == "" || ui.conn.host != s {

			if ui.Editor.View.Text != "" {
				ui.Editor.text = ui.Editor.View.Text
			}

			if ui.Viewer.View.Text != "" {
				ui.Viewer.text = ui.Viewer.View.Text
			}

			ui.Viewer.View.SetText("")
			ui.Viewer.View.SetText("")
			ui.ConnectBtn.SetText("Connect")
			ui.ConnectBtn.Enable()

		} else {

			ui.Viewer.View.SetText(ui.Editor.text)
			ui.Viewer.View.SetText(ui.Viewer.text)
			ui.ConnectBtn.SetText(ui.conn.host)
			ui.ConnectBtn.Disable()

		}

		// ui.Editor.Menu.Disable()
		// ui.Editor.addConfig.Disable()
		// ui.Editor.DelConfig.Disable()
		// ui.Editor.EditConfig.Disable()
		// ui.Viewer.Menu.Disable()
		// ui.Viewer.addConfig.Disable()
		// ui.Viewer.DelConfig.Disable()
		// ui.Viewer.EditConfig.Disable()

		// fmt.Println(s)
		// fmt.Println(ui.config.Hosts[s])
		ui.SetHostConfig(s)

	}

	ui.HostDesc.OnChanged = func(s string) {
		ui.config.Hosts[ui.config.Host] = Host{
			Desc:    s,
			Editors: ui.config.Hosts[ui.config.Host].Editors,
			Viewers: ui.config.Hosts[ui.config.Host].Viewers,
		}
	}

	ui.MenuOpen.Action = func() {
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
			ui.SetHostConfig(ui.HostEntry.Text)

			// fmt.Println(ui.HostEntry.Text)

		}

		dialog.ShowFileOpen(onChosen, ui.Window)
	}

	ui.editHost.OnTapped = func() {

		label1 := widget.NewLabel("Host Specification")
		value1 := widget.NewLabel(ui.HostEntry.Text)
		label2 := widget.NewLabel("Description")
		value2 := widget.NewEntry()
		okButton := widget.NewButton("OK", func() {
			ui.config.Hosts[ui.HostEntry.Text] = Host{
				Desc:    value2.Text,
				Editors: ui.config.Hosts[ui.HostEntry.Text].Editors,
				Viewers: ui.config.Hosts[ui.HostEntry.Text].Viewers,
			}
			ui.editHostPopup.Hide()
		})
		cancelButton := widget.NewButton("Cancel", func() {
			ui.editHostPopup.Hide()
		})

		value2.SetText(ui.config.Hosts[ui.HostEntry.Text].Desc)
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

		ui.editHostPopup = widget.NewModalPopUp(cont, ui.Window.Canvas())
		ui.editHostPopup.Show()

	}

	ui.ConnectBtn.OnTapped = func() {

		_ = ui.Connect()
		user, host := ParseHostSpecToUserHost(ui.HostEntry.Text)

		// fmt.Println()
		ui.config.Hosts[ui.HostEntry.Text] = ui.config.Hosts[ui.config.Host]

		ui.SetHostConfig(ui.HostEntry.Text)
		ui.SetHostConn(ui.HostEntry.Text)

		ui.HostEntry.SetOptions(maps.Keys(ui.config.Hosts))

		// fmt.Println((ui.config))

		ui.Editor.Menu.ClearSelected()
		ui.Viewer.Menu.ClearSelected()

		ui.Editor.View.SetText("")
		ui.Viewer.View.SetText("")

		// ui.Editor.Menu.Disable()
		ui.Editor.addConfig.Disable()
		ui.Editor.DelConfig.Disable()
		ui.Editor.EditConfig.Disable()
		// ui.Viewer.Menu.Disable()
		ui.Viewer.addConfig.Disable()
		ui.Viewer.DelConfig.Disable()
		ui.Viewer.EditConfig.Disable()

		ui.config.Save()

		ui.SetConnected(ui.HostEntry.Text)
		ui.showMessage(fmt.Sprintf(
			"successfully connected as %s to %s", user, host))

		ui.hideProgress(fmt.Sprintf(
			"success: connected to %s as %s", host, user))
	}

	ui.HelpMenu = widget.NewSelect(
		[]string{"Editor", "Viewer", "config"},
		func(s string) {

			if s == "Editor" {
				ui.HelpView.ParseMarkdown(ui.Editor.help())
				ui.JsonView.Hide()
				ui.HelpView.Show()
				ui.JsonSave.Hide()
				return
			}
			if s == "Viewer" {
				ui.HelpView.Wrapping = fyne.TextWrapBreak
				// ui.HelpView.Scroll = container.ScrollVerticalOnly
				ui.HelpView.ParseMarkdown(ui.Viewer.help())
				ui.JsonView.Hide()
				ui.HelpView.Show()
				ui.JsonSave.Hide()
				return
			}
			if s == "config" {
				ui.jsonText, _ = ui.config.Json()
				ui.JsonView.SetText(ui.jsonText)
				ui.HelpView.Hide()
				ui.JsonView.Show()
				ui.JsonSave.Show()
				return
			}

		},
	)

	ui.JsonView.OnChanged = func(s string) {

		// Disable the Save button if the text has not changed
		// Enable if it has changed

		if s == ui.jsonText {

			ui.JsonSave.Disable()

		} else {

			ui.JsonSave.Enable()

		}

	}

	ui.JsonSave.OnTapped = func() {

		// Save the json config settings to file
		// ui.jsonEntry.Text is the text to be saved
		// ui.jsonText is the text before editing started
		// ui.config.File is our output file

		// ToDo: Check if the file corresponds with our data structure
		// before saving

		// convert the string to byte stream after appending a newline
		b := []byte(fmt.Sprintf("%s\n", ui.JsonView.Text))

		// write the byte stream to file
		err := os.WriteFile(ui.config.File, b, 0644)
		if err != nil {

			ui.showError(fmt.Sprintf(
				"%s: %s", "fail: saving configuration", err.Error()))

			return
		}

		ui.jsonText = ui.JsonView.Text
		ui.JsonView.OnChanged(ui.JsonView.Text)

	}

	ui.Editor.EditConfig.OnTapped = func() { ui.editJob(ui.Editor) }
	ui.Viewer.EditConfig.OnTapped = func() { ui.editJob(ui.Viewer) }

	ui.Editor.Menu.OnChanged = func(s string) {
		ui.Editor.Desc.SetText(ui.Editor.EditorConfig[s]["desc"])
		ui.runJob(ui.Editor, s)
	}
	ui.Viewer.Menu.OnChanged = func(s string) {
		ui.Viewer.Desc.SetText(ui.Viewer.EditorConfig[s]["desc"])
		ui.runJob(ui.Viewer, s)
	}

	ui.Editor.Save.OnTapped = func() { ui.saveJob(ui.Editor) }

	return ui
}
