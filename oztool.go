package main

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/pango"
	"log"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"os/user"
	"strconv"
	"reflect"
	"strings"
	"path"
	"bufio"
	"regexp"
	"path/filepath"
//	"errors"
)


type slPreferences struct {
	Winheight uint
	Winwidth uint
	Wintop uint
	Winleft uint
}

const (
	DataTypeNone = iota
	DataTypeString
	DataTypeBool
	DataTypeInt
	DataTypeMultiInt
	DataTypeUInt
	DataTypeStrArray
	DataTypeStrMulti
	DataTypeStruct
	DataTypeStructArray
)

type menuVal struct {
	Name string
	Description string
	Function interface{}
	AccelStr string
}

const (
	ConfigOptionNone = iota
	ConfigOptionImage
	ConfigOptionFilePicker
)

const (
	ConfigVerifierNone = 0
	ConfigVerifierFileExists = 1
	ConfigVerifierFileReadable = 2
	ConfigVerifierFileExec = 4
	ConfigVerifierFileCanBeNull = 8
)

type ConfigOption struct {
	Flag uint
	Option interface{}
	Verification uint
}

type configVal struct {
	Name string
	Description string
	Type int
	Value interface{}
	WidgetAssoc interface{}
	Possibilities []interface{}
	Option ConfigOption
}


var userPrefs slPreferences
var mainWin *gtk.Window
var globalLS *gtk.ListStore
var profileBox *gtk.Box = nil
var Notebook *gtk.Notebook = nil
var notebookPages map[string]*gtk.Box
var CurProfile map[string]*json.RawMessage
var ProfileNames []string
var alertProvider *gtk.CssProvider


var extforwarder_config_data = []configVal {
	{ "name", "Name", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "dynamic", "Dynamic", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "multi", "Multi", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "ext_proto", "External Protocol", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "proto", "Protocol", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "addr", "Address", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "target_host", "Target Host", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "target_port", "Target Port", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "socket_owner", "Socket Owner", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
}

var whitelist_config_template = []configVal {
	{ "path", "Path", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, 0} },
	{ "target", "Target", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "read_only", "Read Only", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "can_create", "Can Create", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "ignore", "Ignore", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "force", "Force", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "no_follow", "No Follow", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "allow_suid", "Allow Setuid", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
}

var blacklist_config_template = []configVal {
	{ "path", "Path", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, 0} },
	{ "no_follow", "No Follow", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
}

var envvar_config_template = []configVal {
	{ "name", "Name", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "value", "Value", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
}

var general_config_template = []configVal {
	{ "name", "Name", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "path", "Path", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists} },
	{ "paths", "Paths", DataTypeStrArray, []string{}, nil, nil, ConfigOption{0, nil, 0} },
	{ "profile_path", "Profile Path", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "OZ Profile Configs (*.json)": {"*.json"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull} },
	{ "default_params", "Default Parameters", DataTypeStrArray, []string{}, nil, nil, ConfigOption{0, nil, 0} },
	{ "reject_user_args", "Reject User Arguments", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "auto_shutdown", "Auto Shutdown", DataTypeStrMulti, "yes", nil, []interface{}{ "no", "yes", "soft" }, ConfigOption{0, nil, 0} },
	{ "watchdog", "Watchdog", DataTypeStrArray, []string{}, nil, nil, ConfigOption{0, nil, 0} },
	{ "wrapper", "Wrapper", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull} },
	{ "multi", "One sandbox per instance", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "no_sys_proc", "Disable sandbox mounting of /sys and /proc", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "no_defaults", "Disable default directory mounts", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "allow_files", "Allow bind mounting of files as args inside the sandbox", DataTypeBool, false, nil, nil, ConfigOption{0, nil, 0} },
	{ "allowed_groups", "Allowed Groups", DataTypeStrArray, "", nil, nil, ConfigOption{0, nil, 0} },
}

var X11_config_template = []configVal {
	{ "enabled", "Enabled", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "tray_icon", "Tray Icon", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionImage, nil, 0} },
	{ "window_icon", "Window Icon", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionImage, nil, 0} },
	{ "enable_tray", "Enable Tray", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "enable_notifications", "Enable Notifications", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "disable_clipboard", "Disable Clipboard", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "audio_mode", "Audio Mode", DataTypeStrMulti, "none", nil, []interface{}{ "none", "speaker", "full", "pulseaudio"}, ConfigOption{0, nil, 0} },
	{ "pulseaudio", "Enable PulseAudio", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "border", "Border", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
}

var network_config_template = []configVal {
	{ "type", "Network Type", DataTypeStrMulti, "none", nil, []interface{}{ "none", "host", "empty", "bridge" }, ConfigOption{0, nil, 0} },
	{ "bridge", "Bridge", DataTypeString, "", nil, nil, ConfigOption{0, nil, 0} },
	{ "dns_mode", "DNS Mode", DataTypeStrMulti, "none", nil, []interface{}{ "none", "pass", "dhcp" }, ConfigOption{0, nil, 0} },
}

var seccomp_config_template = []configVal {
	{ "mode", "Mode", DataTypeStrMulti, "disabled", nil, []interface{}{ "train", "whitelist", "blacklist", "disabled" }, ConfigOption{0, nil, 0} },
	{ "enforce", "Enforce", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "debug", "Debug Mode", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "train", "Training Mode", DataTypeBool, true, nil, nil, ConfigOption{0, nil, 0} },
	{ "train_output", "Training Data Output", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, 0} },
	{ "whitelist", "seccomp Syscall Whitelist", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "Seccomp Configs (*.seccomp)": {"*.seccomp"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull} },
	{ "blacklist", "seccomp Syscall Blacklist", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "Seccomp Configs (*.seccomp)": {"*.seccomp"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull} },
	{ "extradefs", "Extra Definitions", DataTypeStrArray, []string{}, nil, nil, ConfigOption{0, nil, 0} },
}

/*var whitelist_config_template = []configVal {
	{ "", "Whitelist Entry", DataTypeStructArray, nil, nil, nil, ConfigOption{0, nil, 0} },
} */

var allTabs = map[string]*[]configVal { "general": &general_config, "x11": &X11_config, "network": &network_config, "seccomp": &seccomp_config, "whitelist": &whitelist_config }
var allTabsA = map[string]*[][]configVal { "whitelist": nil, "blacklist": nil, "environment": nil }

var allTabsOrdered = []string{ "general", "x11", "network", "whitelist", "blacklist", "seccomp", "environment" }


var file_menu = []menuVal {
	{ "Open Profile", "Open oz profile JSON configuration file", menu_Open, "<Ctrl>o" },
	{ "Save As", "Save current oz profile to JSON configuration file", menu_Save, "<Ctrl>s" },
	{ "Exit", "Quit oztool", menu_Quit, "<Ctrl>q" },
}

var action_menu = []menuVal {
	{ "Run application", "Run the application in its oz sandbox", nil, "<Shift><Alt>F1" },
}

var general_config, X11_config, network_config, seccomp_config, whitelist_config, blacklist_config, environment_config []configVal

var allMenus = map[string][]menuVal { "File": file_menu, "Action": action_menu }
var allMenusOrdered = []string{ "File", "Action" }


func getConfigPath() string {
	usr, err := user.Current()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine location of user preferences file:", err, "\n");
		return ""
	}

	prefPath := usr.HomeDir + "/.oztool.json"
	return prefPath
}

func savePreferences() bool {
	usr, err := user.Current()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine location of user preferences file:", err, "\n");
		return false
	}

	prefPath := usr.HomeDir + "/.oztool.json"

	jsonPrefs, err := json.Marshal(userPrefs)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not generate user preferences data:", err, "\n")
		return false
	}

	err = ioutil.WriteFile(prefPath, jsonPrefs, 0644)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save user preferences data:", err, "\n")
		return false
	}

	return true
}

func loadPreferences() bool {
	usr, err := user.Current()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine location of user preferences file:", err, "\n");
		return false
	}

	prefPath := usr.HomeDir + "/.oztool.json"
	fmt.Println("xxxxxxxxxxxxxxxxxxxxxx preferences path = ", prefPath)

	jfile, err := ioutil.ReadFile(prefPath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not read preference data from file:", err, "\n")
		return false
	}

	err = json.Unmarshal(jfile, &userPrefs)

	if err != nil {
                fmt.Fprintf(os.Stderr, "Error: could not load preferences data from file:", err, "\n")
		return false
	}

	fmt.Println(userPrefs)
	return true
}

func setAlerted(widget *gtk.Entry) {

	if alertProvider == nil {
		var err error
		alertProvider, err = gtk.CssProviderNew()

		if err != nil {
			log.Fatal("Unable to create CSS provider:", err)
		}

		alertProvider.LoadFromData("entry { background-color: #ffa500; }")
	}

	alertContext, err := widget.GetStyleContext()

	if err != nil {
		log.Fatal("Unable to create alert context:", err)
	}

	alertContext.AddProvider(alertProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func serializeConfigToJSON(config []configVal, secname string, fmtlevel int) (string, error) {
	result := "{\n"
	first := true
	padding := 0

	if secname != "general" {
		result = ", \"" + secname + "\": {\n"
	}

	if fmtlevel > 0 {

		for i := 0; i < len(config); i++ {
			if len(config[i].Name) > padding {
				padding = len(config[i].Name)
			}
		}

	}

	for i := 0; i < len(config); i++ {

		if secname == "general" {
			result += " "
		} else {
			result += "     "
		}

		if !first {
			result += ","
		} else {
			first = false
			result += " "
		}

		result += "\"" + config[i].Name + "\": "

		if fmtlevel > 0 {

			for s := 0; s < padding - len(config[i].Name); s++ {
				result += " "
			}

		}

		if config[i].Type == DataTypeString {
			widget := config[i].WidgetAssoc.(*gtk.Entry)
			estr, err := widget.GetText()

			if err != nil {
				log.Fatal("Unable to get value from entry value: ", err)
			}

			if config[i].Option.Verification > 0 {
			err = verifyConfig(config[i].Option.Verification, estr)

			if err != nil {
				rgb := gdk.NewRGBA()
				rgb.Parse("#0000ff")
				setAlerted(config[i].WidgetAssoc.(*gtk.Entry))
//				config[i].WidgetAssoc.(*gtk.Entry).GrabFocus()
				return "", err
			} else {
				ctx, err := config[i].WidgetAssoc.(*gtk.Entry).GetStyleContext()

				if err != nil {
					log.Fatal("Unable to get style context for widget:", err)
				}

				if alertProvider != nil {
					ctx.RemoveProvider(alertProvider)
				}

			}
}

			result += "\"" + estr + "\""
		} else if config[i].Type == DataTypeStrMulti {
			ropts := config[i].WidgetAssoc.([]*gtk.RadioButton)

			for r := 0; r < len(ropts); r++ {

				if ropts[r].GetActive() {
					rstr := config[i].Possibilities[r].(string)
					result += "\"" + rstr + "\""
					break
				}

			}
		} else if config[i].Type == DataTypeBool {
			widget := config[i].WidgetAssoc.(*gtk.CheckButton)

			if widget.GetActive() {
				result += "true"
			} else {
				result += "false"
			}
		} else if config[i].Type == DataTypeInt || config[i].Type == DataTypeUInt {
			result += fmt.Sprintf("%v", config[i].Value)
		} else if config[i].Type == DataTypeStrArray {
			result += "[]"
		} else {
			result += "\"unsupported\""
		}

		result += "\n"
	}

	if secname == "general" {
		result += "\n"
	} else {
		result += "}\n"
	}

	return result, nil
}

func get_radiobutton(group *gtk.RadioButton, label string, activated bool) *gtk.RadioButton {

	if group == nil {
		radiobutton, err := gtk.RadioButtonNewWithLabel(nil, label)

		if err != nil {
			log.Fatal("Unable to create radio button:", err)
		}

		radiobutton.SetActive(activated)
		return radiobutton
	}

	radiobutton, err := gtk.RadioButtonNewWithLabelFromWidget(group, label)

	if err != nil {
		log.Fatal("Unable to create radio button in group:", err)
	}

	radiobutton.SetActive(activated)
	return radiobutton
}

func get_checkbox(text string, activated bool) *gtk.CheckButton {
	cb, err := gtk.CheckButtonNewWithLabel(text)

	if err != nil {
		log.Fatal("Unable to create new checkbox:", err)
	}

	cb.SetActive(activated)
	return cb
}

func get_entry(text string) *gtk.Entry {
	entry, err := gtk.EntryNew()

	if err != nil {
		log.Fatal("Unable to create text entry:", err)
	}

	entry.SetText(text)
	return entry
}

func get_label(text string) *gtk.Label {
	label, err := gtk.LabelNew(text)

	if err != nil {
		log.Fatal("Unable to create label in GUI:", err)
		return nil
	}

	return label
}

func fillNotebookPages(notebook *gtk.Notebook) {

	var pages = []string{ "General", "Whitelist", "Blacklist", "X11", "Environment", "Network", "seccomp", "Forwarders" }

	for n := range pages {

		box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

                if err != nil {
                        log.Fatal("Unable to create notebook page:", err)
                }

		notebook.AppendPage(box, get_label(pages[n]))
		notebookPages[strings.ToLower(pages[n])] = box
	}

}

func createNotebook() *gtk.Notebook {
	notebook, err := gtk.NotebookNew()

        if err != nil {
                log.Fatal("Unable to create new notebook:", err)
        }

	fillNotebookPages(notebook)
	return notebook
}

func createColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, err := gtk.CellRendererTextNew()

	if err != nil {
		log.Fatal("Unable to create text cell renderer:", err)
	}

	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)

	if err != nil {
		log.Fatal("Unable to create cell column:", err)
	}

	return column
}

func createListStore(nadded int) *gtk.ListStore {
	colData := []glib.Type{glib.TYPE_STRING, glib.TYPE_STRING}

	for n := 0; n < nadded; n++ {
		colData = append(colData, glib.TYPE_STRING)
	}

	listStore, err := gtk.ListStoreNew(colData...)

	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}

	return listStore
}

func menu_Open() {
	fmt.Println("OPEN!")
}

func menu_Save() {
	fmt.Println("SAVE!")

	jstr := ""

	for i := 0; i < len(allTabsOrdered); i++ {
		jappend, err := serializeConfigToJSON(*allTabs[allTabsOrdered[i]], allTabsOrdered[i], 1)

		if err != nil {
			promptError(err.Error())
			return
		}

		jstr += jappend
	}

	jstr += "}"

	fmt.Println(jstr)
}

func menu_Quit() {
	fmt.Println("Quitting on user instruction.")
	gtk.MainQuit()
}

func accelDispatch() {
	fmt.Println("ACCELERATOR!!!")
}

func createMenu(box*gtk.Box) {
	menuBar, err := gtk.MenuBarNew()

	if err != nil {
		log.Fatal("Unable to create menu bar:", err)
	}

	for m := 0; m < len(allMenusOrdered); m++ {
		mname := allMenusOrdered[m]

		menu, err := gtk.MenuNew()

		if err != nil {
			log.Fatal("Unable to create menu:", err)
		}

		mi, err := gtk.MenuItemNewWithLabel(mname)

		if err != nil {
			log.Fatal("Unable to create menu item:", err)
		}

		mi.SetSubmenu(menu)
		var ag *gtk.AccelGroup = nil

		for i := 0; i < len(allMenus[mname]); i++ {
			mi2, err := gtk.MenuItemNewWithLabel(allMenus[mname][i].Name)

			if err != nil {
				log.Fatal("Unable to create menu item:", err)
			}

			menu.Append(mi2)

			if allMenus[mname][i].AccelStr != "" {

				if allMenus[mname][i].Function == nil {
					fmt.Fprintf(os.Stderr, "Skipping over empty menu item creation: %v\n", allMenus[mname][i].Name)
					continue
				}

				key, mods := gtk.AcceleratorParse(allMenus[mname][i].AccelStr)

				if ag == nil {
					ag, err = gtk.AccelGroupNew()

					if err != nil {
						log.Fatal("Unable to create accelerator group:", err)
					}

					mainWin.AddAccelGroup(ag)
				}

				ag.Connect(key, mods, gtk.ACCEL_VISIBLE, allMenus[mname][i].Function)
				menu.SetAccelGroup(ag)
			}

			mi2.Connect("activate", allMenus[mname][i].Function)
		}

		menuBar.Append(mi)
	}

	box.PackStart(menuBar, false, false, 0)
}

func promptError(msg string) {
        dialog := gtk.MessageDialogNew(mainWin, 0, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, "Error: %s", msg)
        dialog.Run()
        dialog.Destroy()
}

func verifyConfig(vflags uint, param string) error {

	if vflags == 0 {
		return nil
	}

	if uint64(vflags) & ^uint64((ConfigVerifierFileExists | ConfigVerifierFileReadable | ConfigVerifierFileExec | ConfigVerifierFileCanBeNull)) > 0 {
		fmt.Fprintf(os.Stderr, "Error: unrecognized verify configuration specified - %v\n", vflags)
	}

	if vflags & (ConfigVerifierFileExists | ConfigVerifierFileReadable | ConfigVerifierFileExec) > 0 {
		fmt.Println("Attempting to verify: ", param)

		if (vflags & ConfigVerifierFileCanBeNull == ConfigVerifierFileCanBeNull) && strings.TrimSpace(param) == "" {
			return nil
		}

		_, err := os.Open(param)

		if err != nil {
			return err
		}

	}

	return nil
}

func addRow(listStore *gtk.ListStore, name, data string) {
	iter := listStore.Append()

	colVals := make([]interface{}, 2)
	colVals[0] = name
	colVals[1] = data

	colNums := make([]int, 2)

	for n := 0; n < 2; n++ {
		colNums[n] = n
	}

	err := listStore.Set(iter, colNums, colVals)

	if err != nil {
		log.Fatal("Unable to add row:", err)
	}

}

func tv_click(tv *gtk.TreeView, listStore *gtk.ListStore) {
	sel, err := tv.GetSelection()

	if err == nil {
		rows := sel.GetSelectedRows(listStore)

		fmt.Println("RETURNED ROWS: ", rows.Length())
		if rows.Length() > 0 {
			rdata := rows.NthData(0)

			lIndex, err := strconv.Atoi(rdata.(*gtk.TreePath).String())

			if err == nil {
				fmt.Println("LIST INDEX: ", lIndex)
				fmt.Println("PROFILE BOX = ", profileBox)

				fmt.Println("XXX indexing ", lIndex, " into ", len(ProfileNames))

				Notebook.Destroy()
				showProfileByPath(ProfileNames[lIndex])
/*				CurProfile, err = loadProfileFile(ProfileNames[lIndex])

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading profile data:", err)
				}

				Notebook.Destroy()
				Notebook = createNotebook()
				profileBox.Add(Notebook)

				for tname := range allTabs {
					tbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

					if err != nil {
						log.Fatal("Unable to create box:", err)
					}

					populate_profile_tab(tbox, allTabs[tname])
					notebookPages[tname].Add(tbox)
				}

				Notebook.ShowAll()
				profileBox.ShowAll() */




//				clear_container(notebookPages["general"], true)
//				notebookPages["general"].Add(get_label("OK"))
			} else {
				fmt.Fprintf(os.Stderr, "Error loading profile from selected index:", err)
			}

			fmt.Println("DATAI: ", rdata.(*gtk.TreePath).String())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Could not read profile selection:%v\n", err)
	}

}

func setup_profiles_list(plist []string) *gtk.Box {
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	if err != nil {
		log.Fatal("Unable to create settings box:", err)
	}

	scrollbox, err := gtk.ScrolledWindowNew(nil, nil)

	if err != nil {
		log.Fatal("Unable to create settings scrolled window:", err)
	}

	box.Add(scrollbox)
	scrollbox.SetSizeRequest(500, 200)

	tv, err := gtk.TreeViewNew()

	if err != nil {
		log.Fatal("Unable to create treeview:", err)
	}

	scrollbox.Add(tv)

	sel, err := tv.GetSelection()

	if err == nil {
		sel.SetMode(gtk.SELECTION_SINGLE)
	}

	tv.AppendColumn(createColumn("Name", 0))
	tv.AppendColumn(createColumn("Description", 1))

	listStore := createListStore(2)
	globalLS = listStore

	tv.SetModel(listStore)

	for n := 0; n < len(plist); n++ {
		addRow(listStore, plist[n], "XXX")
	}

	tv.Connect("row-activated", func() {
		tv_click(tv, listStore)
	})

	return box
}

func get_bold_texttag() *gtk.TextTag {
	boldTT, err := gtk.TextTagNew("bold")

	if err != nil {
		log.Fatal("Unable to create text tag for boldface:", err)
	}

	boldTT.SetProperty("weight", pango.WEIGHT_ULTRABOLD)
	return boldTT
}

func clear_container(container *gtk.Box, descend bool) {
	children := container.GetChildren()
	fmt.Println("RETURNED CHILDREN: ", children.Length())

	if descend {
		fmt.Println("REFLECT = ", reflect.TypeOf(children.NthData(0)))
		nchild := children.NthData(0).(*gtk.Box)
		children = nchild.GetChildren()
	}


	i := 0

	children.Foreach(func (item interface{}) {
		i++

		if i > 0 {
			fmt.Println("DELETING: ", reflect.TypeOf(item))
			item.(*gtk.Widget).Destroy()
		}
	})
}

func get_vbox() *gtk.Box {
	vbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	if err != nil {
		log.Fatal("Unable to create vertical box:", err)
	}

	return vbox
}

func get_hbox() *gtk.Box {
	hbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	if err != nil {
		log.Fatal("Unable to create horizontal box:", err)
	}

	return hbox
}

func str_in_array(needle string, haystack[]string, nocase bool) bool {

    for _, i := range haystack {

	if nocase && strings.ToLower(needle) == strings.ToLower(i) {
		return true
	} else if needle == i {
            return true
        }

    }

    return false
}

func populate_profile_tabA(container *gtk.Box, valConfigs [][]configVal) {

	for i := 0; i < len(valConfigs); i++ {
		h := get_vbox()
		h.SetMarginTop(5)
		populate_profile_tab(h, valConfigs[i])
		container.PackStart(h, false, true, 10)
	}

}

func populate_profile_tab(container *gtk.Box, valConfig []configVal) {

	for i := 0; i < len(valConfig); i++ {
//fmt.Println("XXX: current one is: ", valConfig[i].Name)
		h := get_hbox()
		h.SetMarginTop(5)

		if valConfig[i].Type == DataTypeString {
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			val := get_entry(valConfig[i].Value.(string))
			valConfig[i].WidgetAssoc = val
			h.PackStart(val, true, true, 10)

			if valConfig[i].Option.Flag == ConfigOptionImage {
				img, err := gtk.ImageNewFromFile("baba")

				if err != nil {
					fmt.Println("Error: could not load image from file:", err)
				} else {
					h.PackStart(img, false, true, 10)

					pb, err := gdk.PixbufNewFromFileAtScale(valConfig[i].Value.(string), 48, 48, true)

					if err != nil {
						fmt.Println("Error: could not load pixel buf from file:", err)
					} else {
						img.SetFromPixbuf(pb)
					}

				}

			} else if valConfig[i].Option.Flag == ConfigOptionFilePicker {
				fcb, err := gtk.FileChooserButtonNew("Select a file", gtk.FILE_CHOOSER_ACTION_OPEN)

				if err != nil {
					log.Fatal("Unable to create file choose button:", err)
				}

				fcb.Connect("file-set", func() {
					val.SetText(fcb.GetFilename())
				})

				fcb.SetCurrentName(valConfig[i].Value.(string))
				fcb.SetCurrentFolder(filepath.Dir(valConfig[i].Value.(string)))

				if valConfig[i].Option.Option != nil {
					filters := valConfig[i].Option.Option.(map[string][]string)

					for fname := range filters {

						ff, err := gtk.FileFilterNew()

						if err != nil {
							log.Fatal("Unable to create file filter:", err)
						}

						ff.SetName(fname)

						for g := 0; g < len(filters[fname]); g++ {
							ff.AddPattern(filters[fname][g])
						}

						fcb.AddFilter(ff)
					}

				}

				h.PackStart(fcb, false, true, 10)
			}

		} else if valConfig[i].Type == DataTypeBool {
			wcheck := get_checkbox(valConfig[i].Description, valConfig[i].Value.(bool))
			valConfig[i].WidgetAssoc = wcheck
			h.PackStart(wcheck, false, true, 10)
		} else if valConfig[i].Type == DataTypeStrMulti {
			radios := make([]*gtk.RadioButton, 0)
			sval := valConfig[i].Value.(string)
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			r1 := get_radiobutton(nil, valConfig[i].Possibilities[0].(string), sval==valConfig[i].Possibilities[0].(string))
			radios = append(radios, r1)
			h.PackStart(r1, false, true, 10)

			for j := 1; j < len(valConfig[i].Possibilities); j++ {
				rx := get_radiobutton(r1, valConfig[i].Possibilities[j].(string), sval==valConfig[i].Possibilities[j].(string))
				radios = append(radios, rx)
				h.PackStart(rx, false, true, 10)
			}

			valConfig[i].WidgetAssoc = radios
		} else if valConfig[i].Type == DataTypeStrArray {
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			h.PackStart(get_label("[Unsupported]"), false, true, 10)
		} else if valConfig[i].Type == DataTypeStructArray {
			fmt.Println("!!!! struct array")
			fmt.Println("typeof = ", reflect.TypeOf(valConfig[i].Value))
			fmt.Println("val = ", valConfig[i].Value)
		} else {
			fmt.Println("***** UNSUPPORTED -> " + valConfig[i].Name + " / " + valConfig[i].Description)
		}

		container.Add(h)
	}

}

func getChildAsRMA(jdata map[string]*json.RawMessage, field string) []map[string]*json.RawMessage {
	var newm []map[string]*json.RawMessage

	if jdata[field] == nil {
		return nil
	}

	err := json.Unmarshal(*jdata[field], &newm)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing field %s: %v\n", field, err)
		return nil
	}

	return newm
}

func getChildAsRM(jdata map[string]*json.RawMessage, field string) map[string]*json.RawMessage {
	var newm map[string]*json.RawMessage

	if jdata[field] == nil {
		return nil
	}

	err := json.Unmarshal(*jdata[field], &newm)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in parsing field %s: %v\n", field, err)
		return nil
	}

	return newm
}

func populateValuesA(config []configVal, jdata []map[string]*json.RawMessage) [][]configVal {
	config_copy := make([]configVal, len(config))
	copy(config_copy, config)
	allVals := make([][]configVal, 0)
	fmt.Println("ATTEMPTING WITH JLEN # = ", len(jdata), " / CLEN = ", len(config))

	for i := 0; i < len(jdata); i++ {
		val := populateValues(config, jdata[i])
		val_copy := make([]configVal, len(val))
		copy(val_copy, val)
		allVals = append(allVals, val_copy)
		copy(config, config_copy)
	}

	return allVals
}

func populateValues(config []configVal, jdata map[string]*json.RawMessage) []configVal {

//fmt.Println("--- config len was ", len(config))
	for c := 0; c < len(config); c++ {
		jname := config[c].Name
//		fmt.Println("XXX: Attempting to merge: ", jname)

		_, ex := jdata[jname]

		if !ex {
//			fmt.Println("Error: skipping over variable: ", jname)
			continue
		}

//		fmt.Println("Jval = ", jval)

		if config[c].Type == DataTypeBool {
			bval := true
			err := json.Unmarshal(*jdata[jname], &bval)

			if err != nil {
				fmt.Println("Error reading in JSON data as boolean:", err)
			}

			config[c].Value = bval
			fmt.Println("--- deserialized bool = ", bval)
		} else if config[c].Type == DataTypeString {
			sval := ""
			err := json.Unmarshal(*jdata[jname], &sval)

			if err != nil {
				fmt.Println("Error reading in JSON data as string:", err)
			}

			config[c].Value = sval
			fmt.Println("--- deserialized string = ", sval)
		} else if config[c].Type == DataTypeStrMulti {
			sval := ""
			err := json.Unmarshal(*jdata[jname], &sval)

			if err != nil {
				fmt.Println("Error reading in JSON data as string:", err)
			}

			sarray := make([]string, len(config[c].Possibilities))

			for s := 0; s < len(config[c].Possibilities); s++ {
				sarray[s] = config[c].Possibilities[s].(string)
			}

			if !str_in_array(sval, sarray, false) {
				log.Fatal("Error: bad value in JSON combo.")
			}


			config[c].Value = sval
			fmt.Println("--- deserialized string/multi = ", sval)
		} else if config[c].Type == DataTypeStructArray {
			fmt.Println("\n\n\n\nUNSUPPORTEDMULTI!!")
		} else {
			fmt.Println("UNSUPPORTED: ", jname)
		}
//	DataTypeInt DataTypeMultiInt DataTypeUInt DataTypeStrArray 

	}

	return config
}

func reset_configs() {
	general_config = make([]configVal, len(general_config_template))
	X11_config = make([]configVal, len(X11_config_template))
	network_config = make([]configVal, len(network_config_template))
	seccomp_config = make([]configVal, len(seccomp_config_template))
	whitelist_config = make([]configVal, len(whitelist_config_template))
	blacklist_config = make([]configVal, len(blacklist_config_template))
	environment_config = make([]configVal, len(envvar_config_template))
	copy(general_config, general_config_template)
	copy(X11_config, X11_config_template)
	copy(network_config, network_config_template)
	copy(seccomp_config, seccomp_config_template)
	copy(whitelist_config, whitelist_config_template)
	copy(environment_config, envvar_config_template)
}

func showProfileByPath(path string) {
	reset_configs()
	var err error
	CurProfile, err = loadProfileFile(path)

	if err != nil {
		fmt.Println("Unexpected error reading profile from file: ", path)
		return
	}

	jseccomp := getChildAsRM(CurProfile, "seccomp")

	if jseccomp == nil {
		log.Fatal("Error: could not parse seccomp values")
	}

	seccomp_config = populateValues(seccomp_config, jseccomp)

	jx11 := getChildAsRM(CurProfile, "xserver")

	if jx11 == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse X11 values\n")
	}

	X11_config = populateValues(X11_config, jx11)

	jwhitelist := getChildAsRMA(CurProfile, "whitelist")

	if jwhitelist == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse whitelist values\n")
	}

	whitelist_config_array := populateValuesA(whitelist_config, jwhitelist)
	allTabsA["whitelist"] = &whitelist_config_array

//	fmt.Println("FINAL WHITELIST_CONFIG:")
//	fmt.Println(whitelist_config_array)

	jblacklist := getChildAsRMA(CurProfile, "blacklist")

	if jblacklist == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse blacklist values\n")
	}

	blacklist_config_array := populateValuesA(blacklist_config, jblacklist)
	allTabsA["blacklist"] = &blacklist_config_array

	jenv := getChildAsRMA(CurProfile, "environment")

	if jenv == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse environment values\n")
	}

	environment_config_array := populateValuesA(environment_config, jenv)
	allTabsA["environment"] = &environment_config_array


	general_config = populateValues(general_config, CurProfile)



	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	profileBox.Add(Notebook)

	for tname := range allTabs {
		tbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

		if err != nil {
			log.Fatal("Unable to create box:", err)
		}

		populate_profile_tab(tbox, *allTabs[tname])
		notebookPages[tname].Add(tbox)
	}

	profileBox.ShowAll()
}

func main() {
	reset_configs()
	loadPreferences()
	gtk.Init(nil)

	const PROFILES_DIR = "/var/lib/oz/cells.d"
	var err error
	ProfileNames, err = LoadProfilePaths(PROFILES_DIR)

	if err != nil {
		log.Fatal("Error reading contents of profiles directory:", err)
	}


	fmt.Println("profiles len = ", len(ProfileNames))
	fmt.Println("names = ", ProfileNames)
	CurProfile, err = loadProfileFile(ProfileNames[0])

	if err != nil {
		fmt.Println("XXXXXXXXXXXXXXXXXX: error")
	}
	fmt.Println("seccomp: ", reflect.TypeOf(CurProfile["seccomp"]))

	jseccomp := getChildAsRM(CurProfile, "seccomp")

	if jseccomp == nil {
		log.Fatal("Error: could not parse seccomp values")
	}

	seccomp_config = populateValues(seccomp_config, jseccomp)

	jx11 := getChildAsRM(CurProfile, "xserver")

	if jx11 == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse X11 values\n")
	}

	X11_config = populateValues(X11_config, jx11)

	jwhitelist := getChildAsRMA(CurProfile, "whitelist")

	if jwhitelist == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse whitelist values\n")
	}

	whitelist_config_array := populateValuesA(whitelist_config, jwhitelist)
	allTabsA["whitelist"] = &whitelist_config_array
/*	fmt.Println("2FINAL WHITELIST_CONFIG / len = ", len(whitelist_config_array))
	for z := 0; z < len(whitelist_config_array); z++ {
		fmt.Printf("%d: %v\n", z, whitelist_config_array[z])
	} */

	jblacklist := getChildAsRMA(CurProfile, "blacklist")

	if jblacklist == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse blacklist values\n")
	}

	blacklist_config_array := populateValuesA(blacklist_config, jblacklist)
	allTabsA["blacklist"] = &blacklist_config_array

	jenv := getChildAsRMA(CurProfile, "environment")

	if jenv == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse environment values\n")
	}

	environment_config_array := populateValuesA(environment_config, jenv)
	allTabsA["environment"] = &environment_config_array


	general_config = populateValues(general_config, CurProfile)




	mainWin, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

	if err != nil {
		log.Fatal("Unable to create window:", err)
	}

	mainWin.SetTitle("OZ Tool")

	mainWin.Connect("destroy", func() {
		fmt.Println("Shutting down...")
		savePreferences()
	        gtk.MainQuit()
	})

	mainWin.Connect("configure-event", func() {
		w, h := mainWin.GetSize()
		userPrefs.Winwidth, userPrefs.Winheight = uint(w), uint(h)
		l, t := mainWin.GetPosition()
		userPrefs.Winleft, userPrefs.Wintop = uint(l), uint(t)
	})

	mainWin.SetPosition(gtk.WIN_POS_CENTER)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	if err != nil {
		log.Fatal("Unable to create box:", err)
	}

	fmt.Println("profile box = ", profileBox)

	scrollbox, err := gtk.ScrolledWindowNew(nil, nil)

	if err != nil {
		log.Fatal("Unable to create scrolled window:", err)
	}

	mainWin.Add(scrollbox)
	scrollbox.Add(box)

	pbox := setup_profiles_list(ProfileNames)
	profileBox = box
	pbox.SetHAlign(gtk.ALIGN_START)
	pbox.SetVAlign(gtk.ALIGN_FILL)
	createMenu(box)
	box.Add(pbox)

	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	box.Add(Notebook)

	for t := 0; t < len(allTabsOrdered); t++ {
		tname := allTabsOrdered[t]
		tbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

		if err != nil {
			log.Fatal("Unable to create box:", err)
		}

		if tname == "whitelist" {
//			whitelistBox = tbox
		}

		if _, failed := allTabsA[tname]; failed {
			scrollbox, err := gtk.ScrolledWindowNew(nil, nil)

			if err != nil {
				log.Fatal("Unable to create new scrollbox:", err)
			}

			scrollbox.SetSizeRequest(600, 500)
			populate_profile_tabA(tbox, *allTabsA[tname])
			scrollbox.Add(tbox)
			notebookPages[tname].Add(scrollbox)
			continue
		}

		fmt.Println("NEXT")

		populate_profile_tab(tbox, *allTabs[tname])
		notebookPages[tname].Add(tbox)
	}

	if userPrefs.Winheight > 0 && userPrefs.Winwidth > 0 {
		mainWin.Resize(int(userPrefs.Winwidth), int(userPrefs.Winheight))
	} else {
		mainWin.SetDefaultSize(800, 700)
	}

	if userPrefs.Wintop > 0 && userPrefs.Winleft > 0 {
		mainWin.Move(int(userPrefs.Winleft), int(userPrefs.Wintop))
	}

	fmt.Println("profilebox = ", profileBox)
//	profileBox.Add(get_label("OK TEST"))
	mainWin.ShowAll()
	gtk.Main()      // GTK main loop; blocks until gtk.MainQuit() is run. 
}


func LoadProfilePaths(dir string) ([]string, error) {
        fs, err := ioutil.ReadDir(dir)
        if err != nil {
                return nil, err
        }
        ps := make([]string, 0)
        for _, f := range fs {
                if !f.IsDir() {
                        name := path.Join(dir, f.Name())
                        if strings.HasSuffix(f.Name(), ".json") {
                                _, err := loadProfileFile(name)
                                if err != nil {
                                        return nil, fmt.Errorf("error loading '%s': %v", f.Name(), err)
                                }
                                ps = append(ps, name)
                        }
                }
        }

        return ps, nil
}

func LoadProfiles(dir string) ([]map[string]*json.RawMessage, error) {
        fs, err := ioutil.ReadDir(dir)
        if err != nil {
                return nil, err
        }
        ps := make([]map[string]*json.RawMessage, len(fs))
        for _, f := range fs {
                if !f.IsDir() {
                        name := path.Join(dir, f.Name())
                        if strings.HasSuffix(f.Name(), ".json") {
                                p, err := loadProfileFile(name)
                                if err != nil {
                                        return nil, fmt.Errorf("error loading '%s': %v", f.Name(), err)
                                }
                                ps = append(ps, p)
                        }
                }
        }

        return ps, nil
}

var commentRegexp = regexp.MustCompile("^[ \t]*#")

func loadProfileFile(fpath string) (map[string]*json.RawMessage, error) {
	fmt.Println("LOADING FILE: ", fpath)
        file, err := os.Open(fpath)
        if err != nil {
                return nil, err
        }
        scanner := bufio.NewScanner(file)
        bs := ""
        for scanner.Scan() {
                line := scanner.Text()
                if !commentRegexp.MatchString(line) {
                        bs += line + "\n"
                }
        }

	objmap := make(map[string]*json.RawMessage)
	err = json.Unmarshal([]byte(bs), &objmap)

        if err != nil {
                return nil, err
        }

	return objmap, nil

/*      
        if p.Name == "" {
                p.Name = path.Base(p.Path)
        }
        if p.Networking.IpByte <= 1 || p.Networking.IpByte > 254 {
                p.Networking.IpByte = 0
        } */
}
