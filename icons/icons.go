package icons

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var BackIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationArrowBack)
	return icon
}()

var ForwardIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationArrowForward)
	return icon
}()

var ClearIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentClear)
	return icon
}()

var ReplyIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentReply)
	return icon
}()

var CancelReplyIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationCancel)
	return icon
}()

var SendReplyIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentSend)
	return icon
}()

var ContentAdd *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentAdd)
	return icon
}()

var CopyIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentContentCopy)
	return icon
}()

var PasteIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentContentPaste)
	return icon
}()

var FilterIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentFilterList)
	return icon
}()

var MenuIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationMenu)
	return icon
}()

var ServerIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionDNS)
	return icon
}()

var SettingsIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSettings)
	return icon
}()

var ChatIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.CommunicationChat)
	return icon
}()

var IdentityIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionPermIdentity)
	return icon
}()

var ContentEdit *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentCreate)
	return icon
}()

var ContentDelete *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentDeleteSweep)
	return icon
}()
