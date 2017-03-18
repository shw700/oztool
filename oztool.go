package main

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/pango"
	"log"
	"fmt"
	"strings"
	"os"
	"io/ioutil"
	"encoding/json"
	"os/user"
	"strconv"
	"reflect"
)


type slPreferences struct {
	Winheight uint
	Winwidth uint
	Wintop uint
	Winleft uint
}


var userPrefs slPreferences
var mainWin *gtk.Window
var globalLS *gtk.ListStore
var profileBox *gtk.Box = nil
var allProfiles Profiles = nil


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

func add_all_unique_meta_fields(mmap []string, data map[string]string) []string {

	for i := range data {
		var j = 0

		for j = 0; j < len(mmap); j++ {

			if strings.ToLower(mmap[j]) == strings.ToLower(i) {
				break
			}

		}

		if j == len(mmap) {
			fmt.Println("YYY appending: metadata name = ", i)
			mmap = append(mmap, i)
		}

	}

	return mmap
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

func createMenu(box*gtk.Box) {
	menu, err := gtk.MenuNew()

	if err != nil {
		log.Fatal("Unable to create menu:", err)
	}

	mi, err := gtk.MenuItemNewWithLabel("File")

	if err != nil {
		log.Fatal("Unable to create menu item:", err)
	}

	mi.SetSubmenu(menu)

	mi2, err := gtk.MenuItemNewWithLabel("Exit")

	if err != nil {
		log.Fatal("Unable to create menu item:", err)
	}

	menu.Append(mi2)

	menuBar, err := gtk.MenuBarNew()

	if err != nil {
		log.Fatal("Unable to create menu bar:", err)
	}

	menuBar.Append(mi)
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
				clear_container(profileBox)
				profileBox.Add(get_label("123"))
				populate_profile_container(allProfiles[lIndex], profileBox)
			}

			fmt.Println("DATAI: ", rdata.(*gtk.TreePath).String())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Could not read profile selection:%v\n", err)
	}

}

func setup_profiles_list(profiles Profiles) *gtk.Box {
	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	if err != nil {
		log.Fatal("Unable to create settings box:", err)
	}

	scrollbox, err := gtk.ScrolledWindowNew(nil, nil)

	if err != nil {
		log.Fatal("Unable to create settings scrolled window:", err)
	}

	box.Add(scrollbox)
	scrollbox.SetSizeRequest(300, 200)

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

	for n := 0; n < len(profiles); n++ {
		addRow(listStore, profiles[n].Name, "XXX")
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

func clear_container(container *gtk.Box) {
	children := container.GetChildren()

	fmt.Println("RETURNED CHILDREN: ", children.Length())

	i := 0

	children.Foreach(func (item interface{}) {
		i++

		if i > 1 {
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

func populate_profile_container(profile *Profile, container *gtk.Box) {
	fmt.Println("Populating.")
	fmt.Println("1 PROFILE BOX = ", profileBox)

	h := get_hbox()
	h.PackStart(get_label("Name:"), true, true, 0)
	h.PackStart(get_entry(profile.Name), true, true, 0)
	container.Add(h)

	h = get_hbox()
	h.PackStart(get_label("Path:"), true, true, 0)
	h.PackStart(get_entry(profile.Path), true, true, 0)

	fcb, err := gtk.FileChooserButtonNew("Select an application executable", gtk.FILE_CHOOSER_ACTION_OPEN)

	if err != nil {
		log.Fatal("Unable to create file choose button:", err)
	}

	fcb.SetCurrentName(profile.Path)
	fcb.SetCurrentFolder("/usr/bin/")
	h.Add(fcb)
	container.Add(h)

	h = get_hbox()
	h.PackStart(get_label("JSON Path:"), true, true, 0)
	h.PackStart(get_entry(profile.ProfilePath), true, true, 0)
	container.Add(h)

	if len(profile.Paths) == 0 {
		container.Add(get_label("Matching paths: [none]"))
	} else {
		container.Add(get_label("Matching paths:"))

		for i := 0; i < len(profile.Paths); i++ {
			container.Add(get_label("   - " + profile.Paths[i]))
		}
	}

	if len(profile.DefaultParams) == 0 {
		container.Add(get_label("Default parameters: [none]"))
	} else {
		container.Add(get_label("Default parameters:"))

		for i := 0; i < len(profile.DefaultParams); i++ {
			container.Add(get_label("   " + profile.DefaultParams[i]))
		}
	}

	container.Add(get_checkbox("Reject User Arguments", profile.RejectUserArgs))

	if profile.Wrapper == "" {
		container.Add(get_label("Optional binary wrapper: [none]"))
	} else {
		container.Add(get_label("Optional binary wrapper: " + profile.Wrapper))
	}

	container.Add(get_checkbox("One sandbox per instance", profile.Multi))
	container.Add(get_checkbox("Disable sandbox mounting of /sys and /proc ", profile.NoSysProc))
	container.Add(get_checkbox("Disable default directory mounts", profile.NoDefaults))
	container.Add(get_checkbox("Allow bind mounting of files as args inside the sandbox", profile.AllowFiles))
}

func main() {
	loadPreferences()
	gtk.Init(nil)

	const PROFILES_DIR = "/var/lib/oz/cells.d"
	profiles, err := LoadProfiles(PROFILES_DIR)

	if err != nil {
		log.Fatal("Unable to load oz profiles from default directory:", err)
	}

	if len(profiles) == 0 {
		log.Fatal("Unable to load any oz profiles from default directory")
	}

	allProfiles = profiles
	fmt.Println("profile num = ", len(profiles))

/*      
        RejectUserArgs bool `json:"reject_user_args"`
        AutoShutdown ShutdownMode `json:"auto_shutdown"`
        // Optional list of executable names to watch for exit in case initial command spawns and exit
        Watchdog []string
        // Optional wrapper binary to use when launching command (ex: tsocks)
        Wrapper string
        // If true launch one sandbox per instance, otherwise run all instances in same sandbox
        Multi bool
        // Disable mounting of sys and proc inside the sandbox
        NoSysProc bool
        // Disable bind mounting of default directories (etc,usr,bin,lib,lib64)
        // Also disables default blacklist items (/sbin, /usr/sbin, /usr/bin/sudo)
        // Normally not used
        NoDefaults bool
        // Allow bind mounting of files passed as arguments inside the sandbox
        AllowFiles    bool     `json:"allow_files"`
        AllowedGroups []string `json:"allowed_groups"`
        // List of paths to bind mount inside jail
        Whitelist []WhitelistItem
        // List of paths to blacklist inside jail
        Blacklist []BlacklistItem
        // Optional XServer config
        XServer XServerConf
        // List of environment variables
        Environment []EnvVar
        // Networking
        Networking NetworkProfile
        // Seccomp
        Seccomp SeccompConf
        // External Forwarders
        ExternalForwarders []ExternalForwarder `json:"external_forwarders"` */



	// Create a new toplevel window, set its title, and connect it to the "destroy" signal to exit the GTK main loop when it is destroyed.
	mainWin, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

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

/*	vbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	createMenu(vbox) */


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

	pbox := setup_profiles_list(profiles)
	pbox.SetHAlign(gtk.ALIGN_START)
	pbox.SetVAlign(gtk.ALIGN_FILL)
	createMenu(box)
	box.Add(pbox)

	profileBox = pbox
	populate_profile_container(profiles[0], profileBox)
	profileBox.Add(get_label("test label"))


	if userPrefs.Winheight > 0 && userPrefs.Winwidth > 0 {
		mainWin.Resize(int(userPrefs.Winwidth), int(userPrefs.Winheight))
	} else {
		mainWin.SetDefaultSize(800, 600)
	}

	if userPrefs.Wintop > 0 && userPrefs.Winleft > 0 {
		mainWin.Move(int(userPrefs.Winleft), int(userPrefs.Wintop))
	}

	mainWin.ShowAll()
	gtk.Main()      // GTK main loop; blocks until gtk.MainQuit() is run. 
}
