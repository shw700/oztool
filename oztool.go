/*
 * Might require gtkAction implementation in order to support shortcuts in menu items:
 * http://www.kksou.com/php-gtk2/sample-codes/set-up-menu-using-GtkAction-Part-3-add-accelerators-with-labels.php
 *
 * Determine serialization behavior for values left to default
 * Loading of tab contents requires a single code path, not two duplicate ones
 *
 * Directory picker option
 */

package main

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
//	"github.com/gotk3/gotk3/pango"
	"log"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"os/user"
	"os/exec"
	"strconv"
	"reflect"
	"strings"
	"path"
	"bufio"
	"regexp"
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


var extforwarder_config_template = []configVal {
	{ "name", "Name", "", DataTypeString, "", nil, nil, configOptNone },
	{ "dynamic", "Dynamic", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "multi", "Multi", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "ext_proto", "External Protocol", "", DataTypeString, "", nil, nil, configOptNone },
	{ "proto", "Protocol", "", DataTypeString, "", nil, nil, configOptNone },
	{ "addr", "Address", "", DataTypeString, "", nil, nil, configOptNone },
	{ "target_host", "Target Host", "", DataTypeString, "", nil, nil, configOptNone },
	{ "target_port", "Target Port", "", DataTypeString, "", nil, nil, configOptNone },
	{ "socket_owner", "Socket Owner", "", DataTypeString, "", nil, nil, configOptNone },
}

var whitelist_config_template = []configVal {
	{ "path", "Path", "Path to be included in sandbox", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists, nil, nil} },
	{ "target", "Target", "Target path inside sandbox, if different", DataTypeString, "", nil, nil, configOptNone },
	{ "read_only", "Read Only", "Mount specified file as read-only", DataTypeBool, true, nil, nil, configOptNone },
	{ "can_create", "Can Create", "Create the specified file in the sandbox if it doesn't already exist", DataTypeBool, false, nil, nil, configOptNone },
	{ "ignore", "Ignore", "Ignore this file entry if it doesn't exist", DataTypeBool, false, nil, nil, configOptNone },
	{ "force", "Force", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "no_follow", "No Follow", "Do not follow symlinks in mounting process", DataTypeBool, true, nil, nil, configOptNone },
	{ "allow_suid", "Allow Setuid", "Allow setuid files to be mounted in sandbox", DataTypeBool, false, nil, nil, configOptNone },
}

var blacklist_config_template = []configVal {
	{ "path", "Path", "", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists, nil, nil} },
	{ "no_follow", "No Follow", "", DataTypeBool, true, nil, nil, configOptNone },
}

var envvar_config_template = []configVal {
	{ "name", "Name", "", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionNone, nil, ConfigVerifierStrNoBlank, nil, nil} },
	{ "value", "Value", "", DataTypeString, "", nil, nil, configOptNone },
}

var general_config_template = []configVal {
	{ "name", "Name", "Application sandbox name", DataTypeString, "", nil, nil, configOptNone },
	{ "path", "Path", "Pathname to application executable", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists, nil, nil} },
	{ "paths", "Paths", "Additional list of path to binaries matching this sandbox", DataTypeStrArray, []string{}, nil, nil, configOptNone },
	{ "profile_path", "Profile Path", "", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "OZ Profile Configs (*.json)": {"*.json"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
	{ "default_params", "Default Parameters", "Default command line parameters to be passed to the application", DataTypeStrArray, []string{}, nil, nil, configOptNone },
	{ "reject_user_args", "Reject User Arguments", "Discard any custom parameters passed to the application on the command line", DataTypeBool, false, nil, nil, configOptNone },
	{ "auto_shutdown", "Auto Shutdown", "Automatically close sandbox after the application terminates", DataTypeStrMulti, "yes", nil, []interface{}{ "no", "yes", "soft" }, configOptNone },
	{ "watchdog", "Watchdog", "Name of watchdog process(es) e.g. 'python'", DataTypeStrArray, []string{}, nil, nil, configOptNone },
	{ "wrapper", "Wrapper", "", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
	{ "multi", "Create separate sandboxes on each application launch", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "no_sys_proc", "Disable sandbox mounting of /sys and /proc", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "no_defaults", "Disable default directory mounts", "", DataTypeBool, false, nil, nil, configOptNone },
	{ "allow_files", "Allow bind mounting of files as args inside the sandbox", "Command line arguments that are file paths will automatically be mounted into the sandboxed filesystem", DataTypeBool, false, nil, nil, configOptNone },
	{ "allowed_groups", "Allowed Groups", "Additional names of groups whose gids the process will run under", DataTypeStrArray, []string{}, nil, nil, configOptNone },
}

var X11_config_template = []configVal {
	{ "enabled", "Enabled", "Start X11 server inside sandbox", DataTypeBool, true, nil, nil, configOptNone },
	{ "tray_icon", "Tray Icon", "A pathname to an image file to be used as the application's Xpra tray icon", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionImage, nil, 0, nil, nil} },
	{ "window_icon", "Window Icon", "", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionImage, nil, 0, nil, nil} },
	{ "enable_tray", "Enable Tray", "Enable Xpra utility tray for this application", DataTypeBool, true, nil, nil, configOptNone },
	{ "enable_notifications", "Enable Notifications", "Allow notifications on host desktop", DataTypeBool, true, nil, nil, configOptNone },
	{ "disable_clipboard", "Disable Clipboard", "Disable copying and pasting for this application", DataTypeBool, true, nil, nil, configOptNone },
	{ "audio_mode", "Audio Mode", "Audio settings", DataTypeStrMulti, "none", nil, []interface{}{ "none", "speaker", "full", "pulseaudio"}, ConfigOption{0, nil, 0, []string{ "No audio support", "Speaker support only", "Full audio support", "Pass Pulseaudio socket through to sandbox" }, nil } },
	{ "pulseaudio", "Enable PulseAudio", "Use host audio", DataTypeBool, true, nil, nil, configOptNone },
	{ "border", "Border", "Draw border around application", DataTypeBool, true, nil, nil, configOptNone },
}

var StatesNetworkTypeNotBridge = []PrefState{ {"bridge", ""}, {"bridge", PrefDisabled}, {"vpn", PrefDisabled}, {"configpath", false}, {"configpath", PrefDisabled}, {"authfile", false}, {"authfile", PrefDisabled} }
var StatesNetworkTypeBridge = []PrefState{ {"bridge", PrefEnabled}, {"vpn", true}, {"vpn", PrefEnabled}, {"configpath", true}, {"configpath", PrefEnabled}, {"authfile", true}, {"authfile", PrefEnabled } }

var NetworkTypeChangeSet = map[interface{}][]PrefState{ "none": StatesNetworkTypeNotBridge, "host": StatesNetworkTypeNotBridge, "empty": StatesNetworkTypeNotBridge, "bridge": StatesNetworkTypeBridge }

var network_config_template = []configVal {
	{ "type", "Network Type", "Networking options for sandbox", DataTypeStrMulti, "none", nil, []interface{}{ "none", "host", "empty", "bridge" }, ConfigOption{0, nil, 0, []string{ "No loopback interface", "No networking sandbox (sandbox shares network stack with host)", "Loopback device only", "Connect sandbox virtual ethernet interface to bridge" }, NetworkTypeChangeSet } },
	{ "bridge", "Bridge", "", DataTypeString, "", nil, nil, configOptNone },
//	{ "dns_mode", "DNS Mode", "", DataTypeStrMulti, "none", nil, []interface{}{ "none", "pass", "dhcp" }, configOptNone },
	{ "vpn", "VPN", "", DataTypeStrMulti, "openvpn", nil, []interface{}{ "openvpn" }, configOptNone },
	{ "configpath", "Config Path", "Path to VPN configuration file", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "OpenVPN Configurations (*.ovpn)": {"*.ovpn"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
	{ "authfile", "Auth File", "Path to VPN authorization file", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
}

var seccomp_config_template = []configVal {
	{ "mode", "Mode", "Enforcement mode", DataTypeStrMulti, "disabled", nil, []interface{}{ "train", "whitelist", "blacklist", "disabled" }, configOptNone },
	{ "enforce", "Enforce", "", DataTypeBool, true, nil, nil, configOptNone },
	{ "debug", "Debug Mode", "Display full strace-style system call output (only when enforcement is set to false)", DataTypeBool, true, nil, nil, configOptNone },
	{ "train", "Training Mode", "seccomp bpf training mode", DataTypeBool, true, nil, nil, configOptNone },
	{ "train_output", "Training Data Output", "Path to generated training policy", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, nil, 0, nil, nil} },
	{ "whitelist", "seccomp Syscall Whitelist", "Path to seccomp bpf whitelist", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "Seccomp Configs (*.seccomp)": {"*.seccomp"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
	{ "blacklist", "seccomp Syscall Blacklist", "Path to seccomp bpf blacklist", DataTypeString, "", nil, nil, ConfigOption{ConfigOptionFilePicker, map[string][]string{ "Seccomp Configs (*.seccomp)": {"*.seccomp"} }, ConfigVerifierFileExists|ConfigVerifierFileCanBeNull, nil, nil} },
	{ "extradefs", "Extra Definitions", "seccomp bpf policy includes file, for adding variable definitions, macros etc.", DataTypeStrArray, []string{}, nil, nil, configOptNone },
}

/*var whitelist_config_template = []configVal {
	{ "", "Whitelist Entry", DataTypeStructArray, nil, nil, nil, configOptNone },
} */

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

var general_config, X11_config, network_config, seccomp_config, whitelist_config, blacklist_config, environment_config, forwarders_config []configVal

var allMenus = map[string][]menuVal { "File": file_menu, "Action": action_menu, "Sandbox": sandbox_menu }
var allMenusOrdered = []string{ "File", "Action", "Sandbox" }


func getConfigPath() string {
	usr, err := user.Current()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine location of user preferences file:", err, "\n");
		return ""
	}

	prefPath := usr.HomeDir + "/.oztool.json"
	return prefPath
}

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

func unsetSelected(widget *gtk.Button) {

	if selectProvider == nil {
		return
	}

	selectContext, err := widget.GetStyleContext()

	if err != nil {
		log.Fatal("Unable to get select context:", err)
	}

	selectContext.RemoveProvider(selectProvider)
}

func setSelected(widget *gtk.Button) {

	if selectProvider == nil {
		var err error
		selectProvider, err = gtk.CssProviderNew()

		if err != nil {
			log.Fatal("Unable to create CSS provider:", err)
		}

		selectProvider.LoadFromData("button { border-bottom-color: green; border-top-color: green; border-left-color: green; border-right-color: green; background-color: green; color: green; } button:hover { color: green; }")
	}

	selectContext, err := widget.GetStyleContext()

	if err != nil {
		log.Fatal("Unable to get select context:", err)
	}

	selectContext.AddProvider(selectProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
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

func serializeConfigToJSON(config []configVal, secname string, fmtlevel int, inner bool) (string, error) {
	result := ""
	first := true
	padding := 0

	if inner {
		result += ""
	} else {
		result += "{\n"
	}

	if !inner && secname != "general" {
		result = ", \"" + secname + "\": {\n"
	} else if inner {
		result += "     {"
	}

	if fmtlevel > 0 {

		for i := 0; i < len(config); i++ {
			if len(config[i].Name) > padding {
				padding = len(config[i].Name)
			}
		}

	}

	for i := 0; i < len(config); i++ {

		if !inner {
			if secname == "general" {
				result += " "
			} else {
				result += "     "
			}
		}

		if !first {
			result += ","

			if inner {
				result += " "
			}

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
fmt.Println("XXX: verify -> ", config[i].Name)
			err = verifyConfig(config[i].Option.Verification, estr)

			if err != nil {
				rgb := gdk.NewRGBA()
				rgb.Parse("#0000ff")
				setAlerted(config[i].WidgetAssoc.(*gtk.Entry))
//				config[i].WidgetAssoc.(*gtk.Entry).GrabFocus()
				errstr := "Could not verify config field \"" + config[i].Name + "\" in section " + secname + ": " + err.Error()
				return "", errors.New(errstr)
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
			stra := config[i].Value.([]string)

			if len(stra) == 0 {
				result += "[]"
			} else {
				result += "[ "

				for s := 0; s < len(stra); s++ {
					result += "\"" + stra[s] + "\""

					if s < len(stra)-1 {
						result += ", "
					}

				}

				result += " ]"
			}

		} else {
			result += "\"unsupported\""
		}

		if !inner {
			result += "\n"
		}

	}

	if !inner && secname == "general" {
		result += "\n"
	} else {
		if inner {
			result += " }"
		} else {
			result += "}\n"
		}
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

func get_label_tt(text, tooltip string) *gtk.Label {
	label, err := gtk.LabelNew(text)

	if err != nil {
		log.Fatal("Unable to create label in GUI:", err)
		return nil
	}

	label.SetTooltipText(tooltip)
	return label
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

	for n := 0; n < len(allTabsOrdered); n++ {

		box := get_vbox()
		notebook.AppendPage(box, get_label_tt(allTabInfo[allTabsOrdered[n]].TabName, allTabInfo[allTabsOrdered[n]].Tooltip))
		notebookPages[allTabsOrdered[n]] = box
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
	fmt.Println("OPEN!")
}

func menu_Save() {
	fmt.Println("SAVE!")

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

func editStrArray(input []string, fvalidate int) []string {
        dialog := gtk.MessageDialogNew(mainWin, 0, gtk.MESSAGE_INFO, gtk.BUTTONS_OK_CANCEL, "Edit strings list:")

        tv, err := gtk.TextViewNew()

        if err != nil {
                log.Fatal("Unable to create TextView:", err)
        }

        tvbuf, err := tv.GetBuffer()

        if err != nil {
                log.Fatal("Unable to get buffer:", err)
        }

	buftext := strings.Join(input, "\n")
        tvbuf.SetText(buftext)
        tv.SetEditable(true)
        tv.SetWrapMode(gtk.WRAP_WORD)

        scrollbox := get_scrollbox()
        scrollbox.Add(tv)
        scrollbox.SetSizeRequest(500, 200)

        box, err := dialog.GetContentArea()

        if err != nil {
                log.Fatal("Unable to get content area of dialog:", err)
        }

        box.Add(scrollbox)
        dialog.ShowAll()
        choice := dialog.Run()

	bstr, err := tvbuf.GetText(tvbuf.GetStartIter(), tvbuf.GetEndIter(), false)

	if err != nil {
		log.Fatal("Unable to get buffer from text editor:", err)
	}

        dialog.Destroy()

	if choice != int(gtk.RESPONSE_OK) {
		return input
	}

	stra := strings.Split(bstr, "\n")

	if len(stra) == 1 && stra[0] == "" {
		return []string{}
	}

	return stra
}

func promptChoice(msg string) bool {
	dialog := gtk.MessageDialogNew(mainWin, 0, gtk.MESSAGE_ERROR, gtk.BUTTONS_YES_NO, msg)
	result := dialog.Run()
	dialog.Destroy()
	return result == int(gtk.RESPONSE_YES)
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

	} else if (vflags & ConfigVerifierStrNoBlank == ConfigVerifierStrNoBlank) && strings.TrimSpace(param) == "" {
		return errors.New("Field is not permitted to be empty")
	}

	return nil
}

func addRow(listStore *gtk.ListStore, name, data, path string) {
	iter := listStore.Append()

	colVals := make([]interface{}, 3)
	colVals[0] = path
	colVals[1] = data
	colVals[2] = name

	colNums := make([]int, 3)

	for n := 0; n < 3; n++ {
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

func get_scrollbox() *gtk.ScrolledWindow {
	scrollbox, err := gtk.ScrolledWindowNew(nil, nil)

	if err != nil {
		log.Fatal("Unable to create scrolled window:", err)
	}

	return scrollbox
}

func get_hbox() *gtk.Box {
	hbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	if err != nil {
		log.Fatal("Unable to create horizontal box:", err)
	}

	return hbox
}

func get_vbox() *gtk.Box {
	vbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	if err != nil {
		log.Fatal("Unable to create vertical box:", err)
	}

	return vbox
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

func get_button(label string) *gtk.Button {
	button, err := gtk.ButtonNewWithLabel(label)

	if err != nil {
		log.Fatal("Unable to create new button:", err)
	}

	return button
}

func get_narrow_button(label string) *gtk.Button {
	button, err := gtk.ButtonNewWithLabel(label)

	if err != nil {
		log.Fatal("Unable to create new button:", err)
	}

/*	allocation := button.GetAllocation()
	w, h := allocation.GetWidth(), allocation.GetHeight()


	w -=20
	h -= 20
	button.SetSizeRequest(w, h) */

	return button
}

var widget_counter = 31336

func getUniqueWidgetID() int {
	widget_counter++

	if widget_counter == 0 {
		widget_counter++
	}

	return widget_counter
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

func reset_configs() {
	general_config = make([]configVal, len(general_config_template))
	X11_config = make([]configVal, len(X11_config_template))
	network_config = make([]configVal, len(network_config_template))
	seccomp_config = make([]configVal, len(seccomp_config_template))
	whitelist_config = make([]configVal, len(whitelist_config_template))
	blacklist_config = make([]configVal, len(blacklist_config_template))
	environment_config = make([]configVal, len(envvar_config_template))
	forwarders_config = make([]configVal, len(extforwarder_config_template))
	copy(general_config, general_config_template)
	copy(X11_config, X11_config_template)
	copy(network_config, network_config_template)
	copy(seccomp_config, seccomp_config_template)
	copy(whitelist_config, whitelist_config_template)
	copy(blacklist_config, blacklist_config_template)
	copy(environment_config, envvar_config_template)
	copy(forwarders_config, extforwarder_config_template)
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
