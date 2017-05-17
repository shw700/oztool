package main

import (
	"fmt"
	"log"
	"os"
	"io/ioutil"
	"github.com/gotk3/gotk3/gtk"
)


type menuVal struct {
	Name string
	Description string
	Function interface{}
	AccelStr string
}


var file_menu = []menuVal {
	{ "_New Profile", "Create new Oz profile", menu_New, "<Ctrl>n" },
	{ "_Open Profile", "Open oz profile JSON configuration file", menu_Open, "<Ctrl>o" },
	{ "_Save As", "Save current oz profile to JSON configuration file", menu_Save, "<Ctrl>s" },
	{ "-", "", nil, "" },
	{ "E_xit", "Quit oztool", menu_Quit, "<Ctrl>q" },
}

var action_menu = []menuVal {
	{ "_Run application", "Run the application in its oz sandbox", nil, "<Shift><Alt>F1" },
}

var sandbox_menu = []menuVal {
	{ "Launch _shell", "Launch a shell inside its running oz sandbox", menu_Launch, "<Shift><Alt>S" },
	{ "Browse _filesystem", "Browse the local filesystem visible to the sandbox", menu_BrowseFS, "<Shift><Alt>F" },
	{ "View _logs", "View log files for running oz sandbox", menu_Logs, "<Shift><Alt>L" },
	{ "_Kill", "Kill running sandbox", menu_Kill, "<Shift><Alt>K" },
	{ "Relaunch _XPRA", "Relaunch XPRA for running sandbox", menu_RelaunchXPRA, "<Shift><Alt>X" },
}

var allMenus = map[string][]menuVal { "File": file_menu, "Action": action_menu, "Sandbox": sandbox_menu }
var allMenusOrdered = []string{ "File", "Action", "Sandbox" }


func menu_New() {
	fmt.Println("NEW!")
	Notebook.Destroy()
	reset_configs()

	empty_whitelist := make([][]configVal, 0)
	allTabsA["whitelist"] = &empty_whitelist

	empty_blacklist := make([][]configVal, 0)
	allTabsA["blacklist"] = &empty_blacklist

	empty_environment := make([][]configVal, 0)
	allTabsA["environment"] = &empty_environment

	empty_forwarders := make([][]configVal, 0)
        allTabsA["forwarders"] = &empty_forwarders

	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	profileBox.Add(Notebook)

	for t := 0; t < len(allTabsOrdered); t++ {
		tname := allTabsOrdered[t]
		tbox := get_vbox()

		if _, failed := allTabsA[tname]; failed {
			scrollbox := get_scrollbox()
			scrollbox.SetSizeRequest(600, 500)
			tmp := templates[tname]
			populate_profile_tabA(tbox, *allTabsA[tname], &tmp, tname, nil, 0, nil, nil)
			scrollbox.Add(tbox)
			notebookPages[tname].Add(scrollbox)
			continue
		}

		populate_profile_tab(tbox, *allTabs[tname], false)
		notebookPages[tname].Add(tbox)
	}

	profileBox.ShowAll()
}

func menu_Open() {
	ppath := pickJSONFile(false)

	if ppath != "" {
		Notebook.Destroy()
		showProfileByPath(ppath)
	}
}

func menu_Save() {
	jstr := ""

	for i := 0; i < len(allTabsOrdered); i++ {
		tname := allTabsOrdered[i]

		if _, failed := allTabsA[tname]; failed {
			fmt.Println("Trying export of unsupported section: ", tname)
			jstr += ", \"" + tname + "\": [\n"

			if len(*allTabsA[tname]) == 0 {
				jstr += "]\n"
				continue
			}

			for j := 0; j < len(*allTabsA[tname]); j++ {
				jappend, err := serializeConfigToJSON((*allTabsA[tname])[j], tname, 0, true)

				if err != nil {
					promptError(err.Error())
					return
				}

				jstr += jappend

				if j < len(*allTabsA[tname])-1 {
					jstr += ",\n"
				}

			}

			jstr += "]\n"
			continue
		}

		jappend, err := serializeConfigToJSON(*allTabs[tname], tname, 1, false)

		if err != nil {
			promptError(err.Error())
			return
		}

		jstr += jappend
	}

	jstr += "}"

	fmt.Println(jstr)

	ppath := pickJSONFile(true)

	if ppath != "" {
		err := ioutil.WriteFile(ppath, []byte(jstr+"\n"), 0644)

		if err != nil {
			promptError(err.Error())
		}

	}

}

func menu_Kill() {
	fmt.Println("KILL!")

	if promptChoice("Are you sure you want to kill this process?") {
		launchOzCmd("kill", "")
	}
}

func menu_RelaunchXPRA() {
	fmt.Println("RELAUNCH XPRA")
	launchOzCmd("relaunchxpra", "")
}

func menu_Logs() {
	fmt.Println("LOGS")
	launchOzCmd("logs", "")
}

func menu_BrowseFS() {
	fmt.Println("BROWSE FS")
	launchOzCmd("shell", "nautilus")
}

func menu_Launch() {
	fmt.Println("LAUNCH!")
	launchOzCmd("shell", "")
}

func menu_Quit() {
	fmt.Println("Quitting on user instruction.")
	gtk.MainQuit()
}

func popupContextMenu(button int, time uint32) {
	const mname = "Sandbox"
	menu, err := gtk.MenuNew()

	if err != nil {
		log.Fatal("Unable to create context menu:", err)
	}

	for i := 0; i < len(allMenus[mname]); i++ {
		mi, err := gtk.MenuItemNewWithMnemonic(allMenus[mname][i].Name)

		if err != nil {
			log.Fatal("Unable to create menu item:", err)
		}

		if allMenus[mname][i].Function != nil {
			mi.Connect("activate", allMenus[mname][i].Function)
		}

		menu.Append(mi)
	}


	menu.ShowAll()
	menu.PopupAtMouseCursor(nil, nil, button, time)
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

		mi, err := gtk.MenuItemNewWithMnemonic("_"+mname)

		if err != nil {
			log.Fatal("Unable to create menu item:", err)
		}

		mi.SetSubmenu(menu)
		var ag *gtk.AccelGroup = nil

		for i := 0; i < len(allMenus[mname]); i++ {

			if allMenus[mname][i].Name == "-" {
				mi2, err := gtk.SeparatorMenuItemNew()

				if err != nil {
					log.Fatal("Unable to create separator menu item:", err)
				}

				menu.Append(mi2)
			} else {

				mi2, err := gtk.MenuItemNewWithMnemonic(allMenus[mname][i].Name)

				if err != nil {
					log.Fatal("Unable to create menu item:", err)
				}

				if allMenus[mname][i].Function != nil {
					mi2.Connect("activate", allMenus[mname][i].Function)
				}

				menu.Append(mi2)
			}

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

		}

		menuBar.Append(mi)
	}

	box.PackStart(menuBar, false, false, 0)
}

