package main

import (
	"fmt"
	"strings"
	"path"
	"io/ioutil"
	"regexp"
	"log"
	"os"
	"bufio"
	"encoding/json"
	"errors"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
)


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
