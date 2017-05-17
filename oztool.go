/*
 * Might require gtkAction implementation in order to support shortcuts in menu items:
 * http://www.kksou.com/php-gtk2/sample-codes/set-up-menu-using-GtkAction-Part-3-add-accelerators-with-labels.php
 *
 * Determine serialization behavior for values left to default
 * Loading of tab contents requires a single code path, not two duplicate ones
 * Need to handle special vars like ${HOME} and ${XDG_DOWNLOAD_DIR} etc.
 *
 */

package main

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
//	"github.com/gotk3/gotk3/pango"
	"log"
	"fmt"
	"os"
	"encoding/json"
	"os/exec"
	"strconv"
	"reflect"
	"strings"
	"path/filepath"
	"errors"
	"time"
	"io"
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

const (
	ConfigOptionNone = iota
	ConfigOptionImage
	ConfigOptionFilePicker
	ConfigOptionDirPicker
)

const (
	ConfigVerifierNone = 0
	ConfigVerifierFileExists = 1
	ConfigVerifierFileReadable = 2
	ConfigVerifierFileExec = 4
	ConfigVerifierFileCanBeNull = 8
	ConfigVerifierStrNoBlank = 16
	ConfigVerifierArrayNoBlank = 32
)

type PrefStateEnabled struct {
	State bool
}

var PrefEnabled = PrefStateEnabled{true}
var PrefDisabled = PrefStateEnabled{false}

type PrefState struct {
	PrefName string
	Value interface{}
}


type ConfigOption struct {
	Flag uint
	Option interface{}
	Verification uint
	Tooltips interface{}
	ChangeSet map[interface{}][]PrefState
}

type configVal struct {
	Name string
	Label string
	Description string
	Type int
	Value interface{}
	WidgetAssoc interface{}
	Possibilities []interface{}
	Option ConfigOption
}

type objSelected struct {
	Frame *gtk.Frame
	Configs [][]configVal
	SelIndex int
}

var InitSelect = objSelected{ nil, nil, -1 }

type settingsTab struct {
	JName string
	TabName string
	Tooltip string
	SelVal objSelected
}


var userPrefs slPreferences
var mainWin *gtk.Window
var globalLS *gtk.ListStore
var profileBox *gtk.Box = nil
var Notebook *gtk.Notebook = nil
var notebookPages map[string]*gtk.Box
var CurProfile map[string]*json.RawMessage
var ProfileNames []string
var alertProvider *gtk.CssProvider = nil
var selectProvider *gtk.CssProvider = nil
var LastOzList = ""
var monitorTV *gtk.TreeView = nil
var monitorLS *gtk.ListStore = nil

var configOptNone ConfigOption = ConfigOption{ 0, nil, 0, nil, nil }


var allTabs = map[string]*[]configVal { "general": &general_config, "x11": &X11_config, "network": &network_config, "seccomp": &seccomp_config, "whitelist": &whitelist_config, "blacklist": &blacklist_config, "forwarders": &forwarders_config }
var allTabsA = map[string]*[][]configVal { "whitelist": nil, "blacklist": nil, "environment": nil, "forwarders": nil }

var allTabsOrdered = []string{ "general", "x11", "network", "whitelist", "blacklist", "seccomp", "environment", "forwarders" }

var templates = map[string][]configVal { "whitelist": whitelist_config_template, "blacklist": blacklist_config_template, "environment": envvar_config_template }


var allTabInfo = map[string]settingsTab {
	"general": { "", "General", "General application settings", InitSelect },
	"x11": { "xserver", "X11", "X11 settings", InitSelect },
	"network": { "networking", "Network", "Network settings", InitSelect },
	"whitelist": { "whitelist", "Whitelist", "Host filesystem paths mounted into sandbox", InitSelect },
	"blacklist": { "blacklist", "Blacklist", "Restricted paths from host filesystem", InitSelect },
	"seccomp": { "seccomp", "seccomp", "System call filtering rules", InitSelect },
	"environment": { "environment", "Environment", "Application-specific environment variables", InitSelect },
	"forwarders": { "forwarders", "Forwarders", "Connection listeners forwarding into sandbox", InitSelect },
}

var general_config, X11_config, network_config, seccomp_config, whitelist_config, blacklist_config, environment_config, forwarders_config []configVal


func applyConfigValChange(config configVal, configs []configVal, nval interface{}) {
//	fmt.Println("*** in applyConfigValChange()")

	if config.Option.ChangeSet == nil {
		return
	}

//	fmt.Println("AAA / checking for val = ", nval, " / n changeset = ", len(cset))

	if _, ok := config.Option.ChangeSet[nval]; !ok {
		return
	}

	cset := config.Option.ChangeSet[nval]

	var ps = PrefStateEnabled{true}

	for i := 0; i < len(cset); i++ {
//fmt.Println("TRYING: ", cset[i].PrefName)

		for j := 0; j < len(configs); j++ {

			if cset[i].PrefName == configs[j].Name {
//fmt.Println("Setting a value : ", cset[i].PrefName, " -> ", cset[i].Value)

				setEnabled := false

				if reflect.TypeOf(cset[i].Value) == reflect.TypeOf(ps) {
					setEnabled = true
				}

				if configs[j].Type == DataTypeString {

					if setEnabled {
//			fmt.Println("--- SETTING STATE -> ", cset[i].Value.(PrefStateEnabled).State)
						configs[j].WidgetAssoc.(*gtk.Entry).SetSensitive(cset[i].Value.(PrefStateEnabled).State)
					}

				} else if configs[j].Type == DataTypeBool {

					if setEnabled {
//			fmt.Println("/// SETTING STATE -> ", cset[i].Value.(PrefStateEnabled).State)
						configs[j].WidgetAssoc.(*gtk.CheckButton).SetSensitive(cset[i].Value.(PrefStateEnabled).State)
					}

				} else if configs[j].Type == DataTypeStrMulti {

					if setEnabled {
						buttons := configs[j].WidgetAssoc.([]*gtk.RadioButton)
//			fmt.Println("~~~ SETTING STATE -> ", cset[i].Value.(PrefStateEnabled).State)

						for b := 0; b < len(buttons); b++ {
							buttons[b].SetSensitive(cset[i].Value.(PrefStateEnabled).State)
						}

					}

				}
/*	DataTypeStrArray DataTypeStructArray */
			}

		}

	}


//type configVal struct { Name string Label string Description string Type int Value interface{} WidgetAssoc interface{} Possibilities []interface{} Option ConfigOption
}

func launchOzCmd(subcmd string, input string) bool {
	sid, err := getSelectedSandboxID()

	if err != nil {
		log.Fatal("Unable to get selected sandbox ID:", err)
	} else if sid == -1 {
		promptError("Could not determine currently selected Oz sandbox ID")
		return false
	}

	fmt.Println("Launching oz command on sandbox ID: ", sid)

	cmdstr := "xterm"
	cmdargs := []string{ "-e", "oz", subcmd, strconv.Itoa(sid) }

	if subcmd == "logs" {
//		cmdargs = []string{ "-e", "watch", "oz", "logs", strconv.Itoa(sid) }
		shcmd := fmt.Sprintf("while `true`; do clear; oz logs %d; read; done", sid)
		cmdargs = []string{ "-e", "/bin/bash", "-c", shcmd }
	} else if subcmd == "shell" && input == "nautilus" {
		cmdstr = "oz"
		cmdargs = []string{ "shell", strconv.Itoa(sid) }
	}

	fmt.Println("launching process -> ", cmdstr, strings.Join(cmdargs, " "))
	ozCmd := exec.Command(cmdstr, cmdargs...)

	var stdin io.WriteCloser = nil
	var oreader io.ReadCloser = nil
	var ereader io.ReadCloser = nil

	if input != "" {
		stdin, err = ozCmd.StdinPipe()

		if err != nil {
			log.Fatal("Unable to get stdin handle for new process:", err)
		}

		oreader, err = ozCmd.StdoutPipe()

		if err != nil {
			log.Fatal("Unable to get stdout handle for new process:", err)
		}

		ereader, err = ozCmd.StderrPipe()

		if err != nil {
			log.Fatal("Unable to get stdout handle for new process:", err)
		}

	}

	go func() {
		err = ozCmd.Start()

		if err != nil {
			fmt.Println("Error launching application: ", err)
		}

		if oreader != nil {
			buf := make([]byte, 1024)
			nread, err := oreader.Read(buf)

			if err != nil {
				fmt.Println("Error reading output from oz shell: ", err)
			} else {
				fmt.Println("Read ", nread, " bytes after termination of oz process")
//				fmt.Println(string(buf))
			}

		}

		if stdin != nil {
			input += "; exit 0; \r\n\r\n"
			io.WriteString(stdin, input)
		}

		if ereader != nil {
			buf := make([]byte, 1024)
			nread, err := ereader.Read(buf)

			if err != nil {
				fmt.Println("Error reading stderr from oz shell: ", err)
			} else {
				fmt.Println("Read", nread, "stderr bytes after termination of oz process")
				fmt.Println(string(buf))
			}

		}

		err = ozCmd.Wait()
		fmt.Println("Application returned.")

		if err != nil {
			fmt.Println("Error waiting on application: ", err)
		}


		if stdin != nil {
			stdin.Close()
		}

	}()

	return true
}

func getSelectedSandboxID() (int, error) {
	sel, err := monitorTV.GetSelection()

	if err != nil {
		return -1, err
	}

	rows := sel.GetSelectedRows(monitorLS)

	if rows.Length() > 0 {
		rdata := rows.NthData(0)
		tm, err := monitorTV.GetModel()

		if err != nil {
			return -1, err
		}

		iter, err := tm.GetIter(rdata.(*gtk.TreePath))

		if err != nil {
			return -1, err
		}

		val, err := tm.GetValue(iter, 1)

		if err != nil {
			return -1, err
		}

		gval, err := val.GoValue()

		if err != nil {
			return -1, err
		}

		gival, err := strconv.Atoi(gval.(string))

		if err != nil {
			return -1, nil
		}

		return gival, nil
	}

	return -1, nil
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

	} else if (vflags & ConfigVerifierStrNoBlank == ConfigVerifierStrNoBlank) && strings.TrimSpace(param) == "" {
		return errors.New("Field is not permitted to be empty")
	}

	return nil
}

func tv_click(tv *gtk.TreeView, listStore *gtk.ListStore) {
	sel, err := tv.GetSelection()

	if err == nil {
		rows := sel.GetSelectedRows(listStore)

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

func setup_oz_monitor_list() (*gtk.Box, *gtk.ListStore) {
	box := get_vbox()
	scrollbox := get_scrollbox()
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

	tv.SetHeadersClickable(true)

	col1 := createColumn("Oz Profile", 0)
	col2 := createColumn("Sandbox #", 1)
	col1.SetResizable(true)
	col2.SetResizable(true)
	col1.SetSortColumnID(0)
	col2.SetSortColumnID(1)
	tv.AppendColumn(col1)
	tv.AppendColumn(col2)

	listStore := createListStore(2)
	tv.SetModel(listStore)

//	addRow(listStore, plist[n], pname, ppath)

	monitorTV = tv
	monitorLS = listStore

	tv.Connect("row-activated", func() {
		fmt.Println("GOT THIS CLICK")
	})

	tv.Connect("button-press-event",  func(tv *gtk.TreeView, event *gdk.Event) {
		if event == nil {
			return
		}

		eb := &gdk.EventButton{ Event: event }

		if eb.Type() == gdk.EVENT_BUTTON_PRESS && eb.Button() == 3 {
//			fmt.Printf("x = %v, y = %v, button = %v, state = %v, buttonval = %v\n", eb.X(), eb.Y(), eb.Button(), eb.State(), eb.ButtonVal())

			sel, err := monitorTV.GetSelection()

			if err == nil {
				rows := sel.GetSelectedRows(listStore)

				if rows.Length() > 0 {
					popupContextMenu(int(eb.Button()), eb.Time())
				}

			}

		}

	})

	return box, listStore
}

func setup_profiles_list(plist []string) *gtk.Box {
	box := get_vbox()
	scrollbox := get_scrollbox()
	box.Add(scrollbox)
	scrollbox.SetSizeRequest(650, 200)

	tv, err := gtk.TreeViewNew()

	if err != nil {
		log.Fatal("Unable to create treeview:", err)
	}

	scrollbox.Add(tv)

	sel, err := tv.GetSelection()

	if err == nil {
		sel.SetMode(gtk.SELECTION_SINGLE)
	}

	tv.SetHeadersClickable(true)

	col1 := createColumn("App Path", 0)
	col2 := createColumn("Description", 1)
	col3 := createColumn("Profile Path", 2)
	col1.SetResizable(true)
	col2.SetResizable(true)
	col3.SetResizable(true)
	col1.SetSortColumnID(0)
	col2.SetSortColumnID(1)
	col3.SetSortColumnID(2)
	tv.AppendColumn(col1)
	tv.AppendColumn(col2)
	tv.AppendColumn(col3)

	listStore := createListStore(2)
	globalLS = listStore

	tv.SetModel(listStore)

	for n := 0; n < len(plist); n++ {
		pname, ppath := "---", "---"
		prof, err := loadProfileFile(plist[n])

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading profile data:", err)
		} else {
			tmp_config := make([]configVal, len(general_config_template))
			copy(tmp_config, general_config_template)
			tmp_config = populateValues(tmp_config, prof)
			pname = tmp_config[0].Value.(string)
			ppath = tmp_config[1].Value.(string)
		}

		addRow(listStore, plist[n], pname, ppath)
	}

	tv.Connect("row-activated", func() {
		tv_click(tv, listStore)
	})

	return box
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

func rebalanceMap(emap map[int]int, ind int) map[int]int {

	for key := range emap {

		if emap[key] >= ind {
			(emap[key])--
		}

	}

	return emap
}

func populate_profile_tabA(container *gtk.Box, valConfigs [][]configVal, template *[]configVal, jname string, dbutton *gtk.Button, selbase int, emap *map[int]int, bpool *[]*gtk.Button) {
	ctrlbox := get_hbox()
	ctrlbox.SetMarginTop(10)
fmt.Printf("XXX: populatingA on section %s with len = %d\n", jname, len(valConfigs))

	var delButton *gtk.Button = nil

	if dbutton != nil {
		delButton = dbutton
	}

	entryMap := emap

	if entryMap == nil {
		tmpMap := make(map[int]int)
		entryMap = &tmpMap
	}

	selButtonPool := bpool

	if selButtonPool == nil {
		tmpPool :=  make([]*gtk.Button, 0)
		selButtonPool = &tmpPool
	}

	if template != nil {
		b := get_button("New")

		b.Connect("clicked", func() {
			fmt.Println("NEWWWW")
			new_entry := make([]configVal, len(*template))
			copy(new_entry, *template)
			new_entryA := [][]configVal{ new_entry }
			populate_profile_tabA(container, new_entryA, nil, jname, delButton, len(valConfigs), entryMap, selButtonPool)
			container.ShowAll()
//fmt.Println("XXX: old configval = ")
//fmt.Println(valConfigs)
			valConfigs = append(valConfigs, new_entry)
			allTabsA[jname] = &valConfigs
		})

		ctrlbox.PackStart(b, false, true, 5)
		delButton = get_button("Delete")
		delButton.SetSensitive(false)

		delButton.Connect("clicked", func() {
			fmt.Println("DELETED")
			tab := allTabInfo[jname]

	fmt.Println("XXX: selindex = ", tab.SelVal.SelIndex, "jname = ", jname, " / frame = ", tab.SelVal.Frame)

			if tab.SelVal.Frame != nil {
				tab.SelVal.Frame.Destroy()
				tab.SelVal.Frame = nil
				allTabInfo[jname] = tab

				this_map := *(entryMap)
				vi := this_map[tab.SelVal.SelIndex]
				// XXX: theoretically, this could fail...
				fmt.Println("vi = ", vi)
				valConfigs = append(valConfigs[:vi], valConfigs[vi+1:]...)
				allTabsA[jname] = &valConfigs
				*entryMap = rebalanceMap(*entryMap, 1)
			}



			delButton.SetSensitive(false)
//                        tab.SelVal.Configs = valConfigs
//                        tab.SelVal.SelIndex = saved_i
//fmt.Println("XXX: final selbutton pool size = ", len(*selButtonPool))
		})

		ctrlbox.PackStart(delButton, false, true, 5)
		container.Add(ctrlbox)
	}

	for i := 0; i < len(valConfigs); i++ {
		saved_i := i + selbase

/*		if emap != nil {
			saved_i += len(*emap)
		} */

		unique := getUniqueWidgetID()
		frame, err := gtk.FrameNew("")

		if err != nil {
			log.Fatal("Unable to create new frame:", err)
		}

		frame.SetBorderWidth(8)
		frame.SetShadowType(gtk.SHADOW_ETCHED_IN)

		v := get_vbox()
		v.SetMarginTop(10)
		v.SetMarginBottom(10)
		h := get_hbox()
		sbutton := get_button("Select")
		sbutton.SetMarginStart(5)
		*selButtonPool = append(*selButtonPool, sbutton)

		sbutton.Connect("clicked", func() {
//			fmt.Println("SELECTED: ", jname)
			tab := allTabInfo[jname]
			tab.SelVal.Frame = frame
			tab.SelVal.Configs = valConfigs
			tab.SelVal.SelIndex = unique
//			this_map := *(entryMap)
//fmt.Println("XXX: setting selindex = ", tab.SelVal.SelIndex, " -> ", this_map[tab.SelVal.SelIndex])
			allTabInfo[jname] = tab
			delButton.SetSensitive(true)

			for s := 0; s < len(*selButtonPool); s++ {
				unsetSelected((*selButtonPool)[s])
			}

			setSelected(sbutton)
		})

		h.Add(sbutton)
		v.Add(h)
		populate_profile_tab(v, valConfigs[i], true)
		frame.Add(v)
		container.PackStart(frame, false, true, 10)
		(*entryMap)[unique] = saved_i
	}

}

func populate_profile_tab(container *gtk.Box, valConfig []configVal, tight bool) {
	var last_hbox *gtk.Box = nil

	for i := 0; i < len(valConfig); i++ {
//fmt.Println("XXX: current one is: ", valConfig[i].Name)
		h := get_hbox()
		h.SetMarginTop(5)
		container.Add(h)

		if valConfig[i].Type == DataTypeString {
			h.PackStart(get_label(valConfig[i].Label+":"), false, true, 10)
			val := get_entry(valConfig[i].Value.(string))

			if valConfig[i].Description != "" {
				val.SetTooltipText(valConfig[i].Description)
			}

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

					fcb, err := gtk.FileChooserButtonNew("Select a file", gtk.FILE_CHOOSER_ACTION_OPEN)

					if err != nil {
						log.Fatal("Unable to create file choose button:", err)
					}

					fcb.Connect("file-set", func() {
						val.SetText(fcb.GetFilename())

						pb, err := gdk.PixbufNewFromFileAtScale(fcb.GetFilename(), 48, 48, true)

						if err != nil {
							fmt.Println("Error: could not load pixel buf from file:", err)
						} else {
							img.SetFromPixbuf(pb)
						}

					})

//					fcb.SetCurrentName(valConfig[i].Value.(string))
					fcb.SetCurrentFolder(filepath.Dir(valConfig[i].Value.(string)))

					img_filters := make(map[string][]string)
					img_filters["PNG files (*.png)"] = []string{ "*.png" }
					img_filters["SVG files (*.svg)"] = []string{ "*.svg" }
					img_filters["JPEG files (*.jpg, *.jpeg))"] = []string{ "*.jpg", "*.jpeg" }

					for fname := range img_filters {

						ff, err := gtk.FileFilterNew()

						if err != nil {
							log.Fatal("Unable to create file filter:", err)
						}

						ff.SetName(fname)

						for g := 0; g < len(img_filters[fname]); g++ {
							ff.AddPattern(img_filters[fname][g])
						}

						fcb.AddFilter(ff)
					}

					h.PackStart(fcb, false, true, 10)
				}

			} else if valConfig[i].Option.Flag == ConfigOptionFilePicker || valConfig[i].Option.Flag == ConfigOptionDirPicker {
				cflag := gtk.FILE_CHOOSER_ACTION_OPEN

				if valConfig[i].Option.Flag == ConfigOptionDirPicker {
					cflag = gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER
				}

				fcb, err := gtk.FileChooserButtonNew("Select a file", cflag)

				if err != nil {
					log.Fatal("Unable to create file choose button:", err)
				}

				fcb.Connect("file-set", func() {
					val.SetText(fcb.GetFilename())
				})

//				fcb.SetCurrentName(valConfig[i].Value.(string))
				fcb.SetCurrentFolder(filepath.Dir(valConfig[i].Value.(string)))

				if cflag == gtk.FILE_CHOOSER_ACTION_OPEN && valConfig[i].Option.Option != nil {
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

			if tight {
				wcheck := get_checkbox(valConfig[i].Label, valConfig[i].Value.(bool))

				if valConfig[i].Description != "" {
					wcheck.SetTooltipText(valConfig[i].Description)
				}

				valConfig[i].WidgetAssoc = wcheck

				if (i > 0) && (valConfig[i-1].Type == DataTypeBool) && (last_hbox != nil) {
//					fmt.Println("last_hval = ", last_hbox)
					h.Destroy()
					last_hbox.PackStart(wcheck, false, true, 10)
					continue
				} else {
					h.PackStart(wcheck, false, true, 10)
				}
			} else {
				wcheck := get_checkbox(valConfig[i].Label, valConfig[i].Value.(bool))

				saved_i := i

				if valConfig[i].Option.ChangeSet != nil {
					wcheck.Connect("clicked", func() {
						applyConfigValChange(valConfig[saved_i], valConfig, wcheck.GetActive())
					})
				}

				if valConfig[i].Description != "" {
					wcheck.SetTooltipText(valConfig[i].Description)
				}

				valConfig[i].WidgetAssoc = wcheck
				h.PackStart(wcheck, false, true, 10)
			}

		} else if valConfig[i].Type == DataTypeStrMulti {
			radios := make([]*gtk.RadioButton, 0)
			sval := valConfig[i].Value.(string)
			rlabel := get_label(valConfig[i].Label+":")

			if valConfig[i].Description != "" {
				rlabel.SetTooltipText(valConfig[i].Description)
			}

			h.PackStart(rlabel, false, true, 10)

			tooltips := []string{}

			if valConfig[i].Option.Tooltips != nil {
				tooltips = valConfig[i].Option.Tooltips.([]string)
			}

			r1 := get_radiobutton(nil, valConfig[i].Possibilities[0].(string), sval==valConfig[i].Possibilities[0].(string))

			saved_i := i

			if valConfig[i].Option.ChangeSet != nil {
				r1.Connect("clicked", func() {

					if !r1.GetActive() {
						return
					}

					applyConfigValChange(valConfig[saved_i], valConfig, valConfig[saved_i].Possibilities[0].(string))
				})
			}

			if len(tooltips) > 0 && tooltips[0] != "" {
				r1.SetTooltipText(tooltips[0])
			}

			radios = append(radios, r1)
			h.PackStart(r1, false, true, 10)

			for j := 1; j < len(valConfig[i].Possibilities); j++ {
				saved_j := j
				rx := get_radiobutton(r1, valConfig[i].Possibilities[j].(string), sval==valConfig[i].Possibilities[j].(string))

				if valConfig[i].Option.ChangeSet != nil {
					rx.Connect("clicked", func() {

						if !rx.GetActive() {
							return
						}

						applyConfigValChange(valConfig[saved_i], valConfig, valConfig[saved_i].Possibilities[saved_j].(string))
					})
				}

				if len(tooltips) > j && tooltips[j-1] != "" {
					rx.SetTooltipText(tooltips[j])
				}

				radios = append(radios, rx)
				h.PackStart(rx, false, true, 10)
			}

			valConfig[i].WidgetAssoc = radios
		} else if valConfig[i].Type == DataTypeStrArray {
			h.PackStart(get_label(valConfig[i].Label+":"), false, true, 10)
			ebutton := get_narrow_button("Edit")
			ebutton.SetTooltipText(valConfig[i].Description)
			h.PackStart(ebutton, false, true, 0)

			saved_i := i
			this_label := get_label("[empty]")
			this_label.SetTooltipText(valConfig[i].Description)
			ebutton.Connect("clicked", func() {
				valConfig[saved_i].Value = editStrArray(valConfig[saved_i].Value.([]string), 0)

				if len(valConfig[saved_i].Value.([]string)) == 0 {
					this_label.SetText("[empty]")
				} else {
					this_label.SetText(strings.Join(valConfig[saved_i].Value.([]string), "\n"))
				}

			})

			if valConfig[i].Value != nil && len(valConfig[i].Value.([]string)) != 0 {
				this_label.SetText(strings.Join(valConfig[i].Value.([]string), "\n"))
			}

			h.PackStart(this_label, false, true, 10)
		} else if valConfig[i].Type == DataTypeStructArray {
			fmt.Println("!!!! struct array")
			fmt.Println("typeof = ", reflect.TypeOf(valConfig[i].Value))
			fmt.Println("val = ", valConfig[i].Value)
		} else {
			fmt.Println("***** UNSUPPORTED -> " + valConfig[i].Name + " / " + valConfig[i].Label)
		}

		last_hbox = h
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
		} else if config[c].Type == DataTypeStrArray {
			sval := []string{}
			err := json.Unmarshal(*jdata[jname], &sval)

			if err != nil {
				fmt.Println("Error reading in JSON data as string array:", err)
			}

			config[c].Value = sval
		} else if config[c].Type == DataTypeStructArray {
			fmt.Println("\n\n\n\nUNSUPPORTEDMULTI!!")
		} else {
			fmt.Println("UNSUPPORTED: ", jname)
		}
//	DataTypeInt DataTypeMultiInt DataTypeUInt DataTypeStrArray 

	}

	return config
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

	jforwarders := getChildAsRMA(CurProfile, "external_forwarder")

        if jforwarders == nil {
                fmt.Fprintf(os.Stderr, "Error: could not parse forwarders values\n")
        }

        forwarders_config_array := populateValuesA(forwarders_config, jforwarders)
        allTabsA["forwarders"] = &forwarders_config_array



	general_config = populateValues(general_config, CurProfile)



	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	profileBox.Add(Notebook)

/*	for tname := range allTabs {
		tbox := get_vbox()
		populate_profile_tab(tbox, *allTabs[tname], false)
		notebookPages[tname].Add(tbox)
	} */



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

	jforwarders := getChildAsRMA(CurProfile, "external_forwarder")

        if jforwarders == nil {
                fmt.Fprintf(os.Stderr, "Error: could not parse forwarders values\n")
        }

        forwarders_config_array := populateValuesA(forwarders_config, jforwarders)
        allTabsA["forwarders"] = &forwarders_config_array


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

	box := get_vbox()

	if err != nil {
		log.Fatal("Unable to create box:", err)
	}

//	scrollbox := get_scrollbox()
//	mainWin.Add(scrollbox)
//	scrollbox.Add(box)
	mainWin.Add(box)

	treebox := get_hbox()
	pbox := setup_profiles_list(ProfileNames)
	monitor, mls := setup_oz_monitor_list()
	fmt.Println("monitor = ", monitor)
	profileBox = box
	pbox.SetHAlign(gtk.ALIGN_START)
	pbox.SetVAlign(gtk.ALIGN_FILL)
	monitor.SetHAlign(gtk.ALIGN_END)
	monitor.SetVAlign(gtk.ALIGN_FILL)
	createMenu(box)
	treebox.Add(pbox)
	spacing := get_vbox()
	spacing.Add(get_label(" "))
	treebox.Add(spacing)
	treebox.Add(monitor)
	box.Add(treebox)



	ticker := time.NewTicker(2 * time.Second)
	quit := make(chan struct{})

	go func() {
		for {
			select {
				case <- ticker.C:
//				fmt.Println("TIMER!!!")

				cmd := exec.Command("/usr/bin/oz", "list")
				outbytes, err := cmd.Output()

				if err != nil {
					log.Fatal("Error retrieving output of external command:", err)
					return
				}

				outstr := string(outbytes)

				if LastOzList == outstr {
//					fmt.Println("Skipping update.... same")
					continue
				}

				LastOzList = outstr

				olines := strings.Split(outstr, "\n")
				mls.Clear()

				for o := 0; o < len(olines); o++ {
					olines[o] = strings.TrimSpace(olines[o])

					if olines[o] == "" {
						continue
					}

					toks := strings.Split(olines[o], ")")

					if len(toks) != 2 {
						fmt.Fprintln(os.Stderr, "Output from oz tool does not match expected data format!")
						continue
					}

					toks[0] = strings.TrimSpace(toks[0])
					toks[1] = strings.TrimSpace(toks[1])

//					fmt.Println("TIMER: ", olines[o])
					iter := mls.Append()
					colVals := []interface{}{ toks[1], toks[0] }
					colNums := []int{ 0, 1 }
					err = mls.Set(iter, colNums, colVals)

					if err != nil {
						log.Fatal("Unable to add row:", err)
					}

				}

			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()


	notebookPages = make(map[string]*gtk.Box)
	Notebook = createNotebook()
	box.Add(Notebook)

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
