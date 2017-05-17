package main

import (
	"strings"
	"log"
	"github.com/gotk3/gotk3/gtk"
)


func pickJSONFile(save bool) string {
	title := "Save data to profile"
	action := gtk.FILE_CHOOSER_ACTION_SAVE
	btext := "Save"
	retval := ""

	if !save {
		title = "Load data from profile"
		action = gtk.FILE_CHOOSER_ACTION_OPEN
		btext = "Open"
	}

	dialog, err := gtk.FileChooserDialogNewWith1Button(title, mainWin, action, btext, gtk.RESPONSE_OK)

	if err != nil {
		log.Fatal("Unable to create file choice dialog:", err)
	}

	ff, err := gtk.FileFilterNew()

	if err != nil {
		log.Fatal("Unable to create profile file filter:", err)
	}

	dialog.SetCurrentFolder("/var/lib/oz/cells.d")
	ff.SetName("Oz profiles (*.json")
	ff.AddPattern("*.json")
	dialog.AddFilter(ff)

	choice := dialog.Run()

	if choice == int(gtk.RESPONSE_OK) {
		retval = dialog.GetFilename()
	}

	dialog.Destroy()
	return retval
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
