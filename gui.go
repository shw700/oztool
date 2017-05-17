package main

import (
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"log"
	"fmt"
	"reflect"
)

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

