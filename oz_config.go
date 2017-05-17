package main


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
