// List widget with selection bar based on the termui List widget
package selection_list

// TODO
// - Don't use padding to do scrolling, just store the current scroll offset
//   instead and start the text from that point.
// - Implement wrapping (Overflow: wrap)
// - Higlight the line all the way across by explicitly setting the relevant
//   line on the screen using buf.Set() instead of relying on default fg/bg.
//   Then we can override and not have to deal with the green.

import (
	ui "github.com/gizak/termui"
)

type SelectionList struct {
	ui.List
	SelectedItem        int
	EnableSelection     bool
	SelectedItemBgColor ui.Attribute
	SelectedItemFgColor ui.Attribute
}

func NewSelectionList() *SelectionList {
	l := &SelectionList{List: *ui.NewList()}
	// We don't have theme items for these, so just pick some values here by
	// default
	l.SelectedItemFgColor = ui.ColorBlack
	l.SelectedItemBgColor = ui.ColorCyan
	l.EnableSelection = true
	return l
}

func (l *SelectionList) Scroll(x, y int, absolute bool) {
	if absolute {
		l.PaddingLeft = -x
		l.PaddingTop = -y
	} else {
		l.PaddingLeft -= x
		l.PaddingTop -= y
	}
	if l.PaddingTop < (l.Height - len(l.Items)) {
		l.PaddingTop = (l.Height - len(l.Items))
	}
	if l.PaddingTop > 0 {
		l.PaddingTop = 0
	}
	if l.PaddingLeft > 0 {
		l.PaddingLeft = 0
	}
}

func (l *SelectionList) SelectItem(count int, absolute bool) {
	if !l.EnableSelection {
		// Selection mode is turned off, so just scroll instead
		l.Scroll(0, count, absolute)
	}
	// Select a new item, with bounds checking
	if absolute {
		l.SelectedItem = count
	} else {
		l.SelectedItem += count
	}
	if l.SelectedItem >= len(l.Items) {
		l.SelectedItem = len(l.Items) - 1
	}
	if l.SelectedItem < 0 {
		l.SelectedItem = 0
	}
	// Scroll so that the selected item is always in view
	if l.PaddingTop < 0-l.SelectedItem {
		l.PaddingTop = 0 - l.SelectedItem
	}
	if l.PaddingTop > l.Height-1-l.SelectedItem {
		l.PaddingTop = l.Height - 1 - l.SelectedItem
	}
}

func (l *SelectionList) Buffer() ui.Buffer {
	buf := l.Block.Buffer()

	trimItems := l.Items
	if len(trimItems) > l.Block.InnerHeight() {
		trimItems = trimItems[:l.Block.InnerHeight()]
	}
	for i, v := range trimItems {
		fg := l.ItemFgColor
		bg := l.ItemBgColor
		if i == l.SelectedItem && l.EnableSelection {
			fg = l.SelectedItemFgColor
			bg = l.SelectedItemBgColor
		}
		cs := ui.DTrimTxCls(ui.DefaultTxBuilder.Build(v, fg, bg),
			l.Block.InnerWidth())
		j := 0
		for _, vv := range cs {
			w := vv.Width()
			buf.Set(l.Block.InnerX()+j, l.Block.InnerY()+i, vv)
			j += w
		}
	}
	return buf
}
