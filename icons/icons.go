package icons

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var (
	BackIcon      *widget.Icon = must(widget.NewIcon(icons.NavigationArrowBack))
	ForwardIcon   *widget.Icon = must(widget.NewIcon(icons.NavigationArrowForward))
	ContentEdit   *widget.Icon = must(widget.NewIcon(icons.ContentCreate))
	ContentDelete *widget.Icon = must(widget.NewIcon(icons.ContentDeleteSweep))
	ContentAdd    *widget.Icon = must(widget.NewIcon(icons.ContentAdd))
	Configuration *widget.Icon = must(widget.NewIcon(icons.ActionSettings))
)

func must(icon *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return icon
}
