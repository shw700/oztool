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
)


type slPreferences struct {
	Winheight uint
	Winwidth uint
	Wintop uint
	Winleft uint
}

const (
	DataTypeString = iota
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

type ConfigOption struct {
	Flag uint
	Option interface{}
}

type configVal struct {
	Name string
	Description string
	Type int
	Value interface{}
	Possibilities []interface{}
	Option ConfigOption
}


var userPrefs slPreferences
var mainWin *gtk.Window
var globalLS *gtk.ListStore
var profileBox *gtk.Box = nil
var Notebook *gtk.Notebook = nil
var notebookPages map[string]*gtk.Box


var general_config = []configVal {
	{ "name", "Name", DataTypeString, "", nil, ConfigOption{0, nil} },
	{ "path", "Path", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "paths", "Paths", DataTypeStrArray, []string{}, nil, ConfigOption{0, nil} },
	{ "profile_path", "Profile Path", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "default_params", "Default Parameters", DataTypeStrArray, []string{}, nil, ConfigOption{0, nil} },
	{ "reject_user_args", "Reject User Arguments", DataTypeBool, false, nil, ConfigOption{0, nil} },
	{ "auto_shutdown", "Auto Shutdown", DataTypeStrMulti, "yes", []interface{}{ "no", "yes", "soft" }, ConfigOption{0, nil} },
	{ "watchdog", "Watchdog", DataTypeStrArray, []string{}, nil, ConfigOption{0, nil} },
	{ "wrapper", "Wrapper", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "multi", "One sandbox per instance", DataTypeBool, false, nil, ConfigOption{0, nil} },
	{ "no_sys_proc", "Disable sandbox mounting of /sys and /proc", DataTypeBool, false, nil, ConfigOption{0, nil} },
	{ "no_defaults", "Disable default directory mounts", DataTypeBool, false, nil, ConfigOption{0, nil} },
	{ "allow_files", "Allow bind mounting of files as args inside the sandbox", DataTypeBool, false, nil, ConfigOption{0, nil} },
	{ "allowed_groups", "Allowed Groups", DataTypeStrArray, "", nil, ConfigOption{0, nil} },
}

var X11_config = []configVal {
	{ "enabled", "Enabled", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "tray_icon", "Tray Icon", DataTypeString, "", nil, ConfigOption{ConfigOptionImage, nil} },
	{ "window_icon", "Window Icon", DataTypeString, "", nil, ConfigOption{ConfigOptionImage, nil} },
	{ "enable_tray", "Enable Tray", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "enable_notifications", "Enable Notifications", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "disable_clipboard", "Disable Clipboard", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "audio_mode", "Audio Mode", DataTypeStrMulti, "none", []interface{}{ "none", "speaker", "full", "pulseaudio"}, ConfigOption{0, nil} },
	{ "pulseaudio", "Enable PulseAudio", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "border", "Border", DataTypeBool, true, nil, ConfigOption{0, nil} },
}

var network_config = []configVal {
	{ "type", "Network Type", DataTypeStrMulti, "none", []interface{}{ "none", "host", "empty", "bridge" }, ConfigOption{0, nil} },
	{ "bridge", "Bridge", DataTypeString, "", nil, ConfigOption{0, nil} },
	{ "dns_mode", "DNS Mode", DataTypeStrMulti, "none", []interface{}{ "none", "pass", "dhcp" }, ConfigOption{0, nil} },
}

var seccomp_config = []configVal {
	{ "mode", "Mode", DataTypeStrMulti, "disabled", []interface{}{ "train", "whitelist", "blacklist", "disabled" }, ConfigOption{0, nil} },
	{ "enforce", "Enforce", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "debug", "Debug Mode", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "train", "Training Mode", DataTypeBool, true, nil, ConfigOption{0, nil} },
	{ "train_output", "Training Data Output", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "whitelist", "seccomp Syscall Whitelist", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "blacklist", "seccomp Syscall Blacklist", DataTypeString, "", nil, ConfigOption{ConfigOptionFilePicker, nil} },
	{ "extradefs", "Extra Definitions", DataTypeStrArray, []string{}, nil, ConfigOption{0, nil} },
}

var allTabs = map[string][]configVal { "general": general_config, "x11": X11_config, "network": network_config, "seccomp": seccomp_config }
var allTabsOrdered = []string{ "general", "x11", "network", "seccomp" }


var file_menu = []menuVal {
	{ "Open Profile", "Open oz profile JSON configuration file", nil, "<Ctrl>o" },
	{ "Save As", "Save current oz profile to JSON configuration file", nil, "<Ctrl>s" },
	{ "Exit", "Quit oztool", menu_Quit, "<Ctrl>q" },
}

var action_menu = []menuVal {
	{ "Run application", "Run the application in its oz sandbox", nil, "<Shift><Alt>F1" },
}

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

func clearNotebookPages(notebook *gtk.Notebook) {

	for i := notebook.GetNPages(); i >= 0; i-- {
		notebook.RemovePage(i)
	}

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
				clearNotebookPages(Notebook)
//				fillNotebookPages(Notebook)
//				clear_container(notebookPages["general"], true)
//				notebookPages["general"].Add(get_label("OK"))
//				populate_profile_container(allProfiles[lIndex], notebookPages["general"])
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
		glib.IdleAdd(func() {
			tv_click(tv, listStore)
		})
	})

/*	tv.Connect("row-activated", func() {
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
//					clear_container(profileBox)
					profileBox.Add(get_label("123"))
				//	populate_profile_container(allProfiles[lIndex], profileBox)
				}

				fmt.Println("DATAI: ", rdata.(*gtk.TreePath).String())
			}
		} else {
			fmt.Fprintf(os.Stderr, "Could not read profile selection:%v\n", err)
		}

	}) */

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

func populate_profile_tab(container *gtk.Box, valConfig []configVal) {

	for i := 0; i < len(valConfig); i++ {
fmt.Println("current one is: ", valConfig[i].Name)
		h := get_hbox()

		if valConfig[i].Type == DataTypeString {
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			h.PackStart(get_entry(valConfig[i].Value.(string)), true, true, 10)

			if valConfig[i].Option.Flag == ConfigOptionImage {
				img, err := gtk.ImageNewFromFile("baba")

				if err != nil {
					fmt.Println("Error: could not load image from file:", err)
				} else {
					h.PackStart(img, false, true, 10)

					pb, err := gdk.PixbufNewFromFileAtScale(valConfig[i].Value.(string), 64, 64, true)

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

				fcb.SetCurrentName(valConfig[i].Value.(string))
				fcb.SetCurrentFolder("/usr/bin/")
				h.PackStart(fcb, false, true, 10)
			}

		} else if valConfig[i].Type == DataTypeBool {
			h.PackStart(get_checkbox(valConfig[i].Description, valConfig[i].Value.(bool)), false, true, 10)
		} else if valConfig[i].Type == DataTypeStrMulti {
			sval := valConfig[i].Value.(string)
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			r1 := get_radiobutton(nil, valConfig[i].Possibilities[0].(string), sval==valConfig[i].Possibilities[0].(string))
			h.PackStart(r1, false, true, 10)

			for j := 1; j < len(valConfig[i].Possibilities); j++ {
				rx := get_radiobutton(r1, valConfig[i].Possibilities[j].(string), sval==valConfig[i].Possibilities[j].(string))
				h.PackStart(rx, false, true, 10)
			}

		} else if valConfig[i].Type == DataTypeStrArray {
			h.PackStart(get_label(valConfig[i].Description+":"), false, true, 10)
			h.PackStart(get_label("[Unsupported]"), false, true, 10)
		}

		container.Add(h)
	}

}

func getChildAsRM(jdata map[string]*json.RawMessage, field string) map[string]*json.RawMessage {
	var newm map[string]*json.RawMessage

	if jdata[field] == nil {
		return nil
	}

	err := json.Unmarshal(*jdata[field], &newm)

	if err != nil {
		return nil
	}

	return newm
}

func populateValues(config []configVal, jdata map[string]*json.RawMessage) []configVal {

	for c := 0; c < len(config); c++ {
		jname := config[c].Name
		fmt.Println("Attempting to merge: ", jname)

		_, ex := jdata[jname]

		if !ex {
			fmt.Println("Error: skipping over variable: ", jname)
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
		} else {
			fmt.Println("UNSUPPORTED: ", jname)
		}
//	DataTypeInt DataTypeMultiInt DataTypeUInt DataTypeStrArray 

	}

	return config
}

func main() {
	loadPreferences()
	gtk.Init(nil)

	const PROFILES_DIR = "/var/lib/oz/cells.d"
	profileNames, err := LoadProfilePaths(PROFILES_DIR)

	if err != nil {
		log.Fatal("Error reading contents of profiles directory:", err)
	}

	fmt.Println("profiles len = ", len(profileNames))
	fmt.Println("names = ", profileNames)
	xxx, err := loadProfileFile(profileNames[0])

	if err != nil {
		fmt.Println("XXXXXXXXXXXXXXXXXX: error")
	}
	fmt.Println("seccomp: ", reflect.TypeOf(xxx["seccomp"]))

	jseccomp := getChildAsRM(xxx, "seccomp")

	if jseccomp == nil {
		log.Fatal("Error: could not parse seccomp values")
	}

	seccomp_config = populateValues(seccomp_config, jseccomp)


	jx11 := getChildAsRM(xxx, "xserver")

	if jx11 == nil {
		fmt.Fprintf(os.Stderr, "Error: could not parse X11 values\n")
	}

	X11_config = populateValues(X11_config, jx11)

	general_config = populateValues(general_config, xxx)





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

	pbox := setup_profiles_list(profileNames)
	profileBox = pbox
	pbox.SetHAlign(gtk.ALIGN_START)
	pbox.SetVAlign(gtk.ALIGN_FILL)
	createMenu(box)
	box.Add(pbox)

	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	box.Add(Notebook)

//	NotebookPages["general"].Add(pbox)
//	box.Add(pbox)

//	profileBox.Add(get_label("HEH"))
//	profileBox.Add(get_label("HEH2"))


	for tname := range allTabs {
		tbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

		if err != nil {
			log.Fatal("Unable to create box:", err)
		}

		populate_profile_tab(tbox, allTabs[tname])
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
