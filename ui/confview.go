/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.zx2c4.com/wireguard/windows/conf"
	"golang.zx2c4.com/wireguard/windows/service"
)

type widgetsLine interface {
	widgets() (walk.Widget, walk.Widget)
}

type widgetsLinesView interface {
	widgetsLines() []widgetsLine
}

type labelStatusLine struct {
	label           *walk.TextLabel
	statusComposite *walk.Composite
	statusImage     *walk.ImageView
	statusLabel     *walk.LineEdit
}

type labelTextLine struct {
	label *walk.TextLabel
	text  *walk.LineEdit
}

type toggleActiveLine struct {
	composite *walk.Composite
	button    *walk.PushButton
}

type interfaceView struct {
	status       *labelStatusLine
	publicKey    *labelTextLine
	listenPort   *labelTextLine
	mtu          *labelTextLine
	addresses    *labelTextLine
	dns          *labelTextLine
	toggleActive *toggleActiveLine
	lines        []widgetsLine
}

type peerView struct {
	publicKey           *labelTextLine
	presharedKey        *labelTextLine
	allowedIPs          *labelTextLine
	endpoint            *labelTextLine
	persistentKeepalive *labelTextLine
	latestHandshake     *labelTextLine
	transfer            *labelTextLine
	lines               []widgetsLine
}

type ConfView struct {
	*walk.ScrollView
	name            *walk.GroupBox
	interfaze       *interfaceView
	peers           map[conf.Key]*peerView
	tunnelChangedCB *service.TunnelChangeCallback
	tunnel          *service.Tunnel
	updateTicker    *time.Ticker
}

func (lsl *labelStatusLine) widgets() (walk.Widget, walk.Widget) {
	return lsl.label, lsl.statusComposite
}

func (lsl *labelStatusLine) update(state service.TunnelState) {
	labelSize := lsl.label.SizeHint()
	imageRect := walk.Rectangle{0, 0, labelSize.Height, labelSize.Height}
	img, err := iconProvider.ImageForState(state, imageRect)
	if err == nil {
		lsl.statusImage.SetImage(img)
	}
	s, e := lsl.statusLabel.TextSelection()
	switch state {
	case service.TunnelStarted:
		lsl.statusLabel.SetText("Active")

	case service.TunnelStarting:
		lsl.statusLabel.SetText("Activating")

	case service.TunnelStopped:
		lsl.statusLabel.SetText("Inactive")

	case service.TunnelStopping:
		lsl.statusLabel.SetText("Deactivating")

	case service.TunnelUnknown:
		lsl.statusLabel.SetText("Unknown state")
	}
	lsl.statusLabel.SetTextSelection(s, e)
}

func newLabelStatusLine(parent walk.Container) *labelStatusLine {
	lsl := new(labelStatusLine)

	lsl.label, _ = walk.NewTextLabel(parent)
	lsl.label.SetText("Status:")
	lsl.label.SetTextAlignment(walk.AlignHFarVNear)

	lsl.statusComposite, _ = walk.NewComposite(parent)
	layout := walk.NewHBoxLayout()
	layout.SetMargins(walk.Margins{})
	layout.SetAlignment(walk.AlignHNearVNear)
	layout.SetSpacing(0)
	lsl.statusComposite.SetLayout(layout)

	lsl.statusImage, _ = walk.NewImageView(lsl.statusComposite)
	lsl.statusImage.SetMode(walk.ImageViewModeIdeal)

	lsl.statusLabel, _ = walk.NewLineEdit(lsl.statusComposite)
	win.SetWindowLong(lsl.statusLabel.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(lsl.statusLabel.Handle(), win.GWL_EXSTYLE)&^win.WS_EX_CLIENTEDGE)
	lsl.statusLabel.SetReadOnly(true)
	lsl.statusLabel.SetBackground(walk.NullBrush())
	lsl.statusLabel.FocusedChanged().Attach(func() {
		lsl.statusLabel.SetTextSelection(0, 0)
	})
	lsl.update(service.TunnelUnknown)

	return lsl
}

func (lt *labelTextLine) widgets() (walk.Widget, walk.Widget) {
	return lt.label, lt.text
}

func (lt *labelTextLine) show(text string) {
	s, e := lt.text.TextSelection()
	lt.text.SetText(text)
	lt.label.SetVisible(true)
	lt.text.SetVisible(true)
	lt.text.SetTextSelection(s, e)
}

func (lt *labelTextLine) hide() {
	lt.text.SetText("")
	lt.label.SetVisible(false)
	lt.text.SetVisible(false)
}

func newLabelTextLine(fieldName string, parent walk.Container) *labelTextLine {
	lt := new(labelTextLine)
	lt.label, _ = walk.NewTextLabel(parent)
	lt.label.SetText(fieldName + ":")
	lt.label.SetTextAlignment(walk.AlignHFarVNear)
	lt.label.SetVisible(false)

	lt.text, _ = walk.NewLineEdit(parent)
	win.SetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE)&^win.WS_EX_CLIENTEDGE)
	lt.text.SetReadOnly(true)
	lt.text.SetBackground(walk.NullBrush())
	lt.text.SetVisible(false)
	lt.text.FocusedChanged().Attach(func() {
		lt.text.SetTextSelection(0, 0)
	})
	return lt
}

func (tal *toggleActiveLine) widgets() (walk.Widget, walk.Widget) {
	return nil, tal.composite
}

func (tal *toggleActiveLine) updateGlobal(globalState service.TunnelState) {
	tal.button.SetEnabled(globalState == service.TunnelStarted || globalState == service.TunnelStopped)
}

func (tal *toggleActiveLine) update(state service.TunnelState) {
	var text string

	switch state {
	case service.TunnelStarted:
		text = "Deactivate"

	case service.TunnelStarting:
		text = "Activating..."

	case service.TunnelStopped:
		text = "Activate"

	case service.TunnelStopping:
		text = "Deactivating..."

	default:
		text = ""
	}

	tal.button.SetText(text)
	tal.button.SetVisible(state != service.TunnelUnknown)
}

func newToggleActiveLine(parent walk.Container) *toggleActiveLine {
	tal := new(toggleActiveLine)

	tal.composite, _ = walk.NewComposite(parent)
	layout := walk.NewHBoxLayout()
	layout.SetMargins(walk.Margins{0, 0, 0, 6})
	tal.composite.SetLayout(layout)

	tal.button, _ = walk.NewPushButton(tal.composite)
	walk.NewHSpacer(tal.composite)
	tal.update(service.TunnelStopped)

	return tal
}

func newInterfaceView(parent walk.Container) *interfaceView {
	iv := &interfaceView{
		newLabelStatusLine(parent),
		newLabelTextLine("Public key", parent),
		newLabelTextLine("Listen port", parent),
		newLabelTextLine("MTU", parent),
		newLabelTextLine("Addresses", parent),
		newLabelTextLine("DNS servers", parent),
		newToggleActiveLine(parent),
		nil,
	}
	iv.lines = []widgetsLine{
		iv.status,
		iv.publicKey,
		iv.listenPort,
		iv.mtu,
		iv.addresses,
		iv.dns,
		iv.toggleActive,
	}
	layoutInGrid(iv, parent.Layout().(*walk.GridLayout))
	return iv
}

func newPeerView(parent walk.Container) *peerView {
	pv := &peerView{
		newLabelTextLine("Public key", parent),
		newLabelTextLine("Preshared key", parent),
		newLabelTextLine("Allowed IPs", parent),
		newLabelTextLine("Endpoint", parent),
		newLabelTextLine("Persistent keepalive", parent),
		newLabelTextLine("Latest handshake", parent),
		newLabelTextLine("Transfer", parent),
		nil,
	}
	pv.lines = []widgetsLine{
		pv.publicKey,
		pv.presharedKey,
		pv.allowedIPs,
		pv.endpoint,
		pv.persistentKeepalive,
		pv.latestHandshake,
		pv.transfer,
	}
	layoutInGrid(pv, parent.Layout().(*walk.GridLayout))
	return pv
}

func layoutInGrid(view widgetsLinesView, layout *walk.GridLayout) {
	for i, l := range view.widgetsLines() {
		w1, w2 := l.widgets()

		if w1 != nil {
			layout.SetRange(w1, walk.Rectangle{0, i, 1, 1})
		}
		if w2 != nil {
			layout.SetRange(w2, walk.Rectangle{2, i, 1, 1})
		}
	}
}

func (iv *interfaceView) widgetsLines() []widgetsLine {
	return iv.lines
}

func (iv *interfaceView) apply(c *conf.Interface) {
	iv.publicKey.show(c.PrivateKey.Public().String())

	if c.ListenPort > 0 {
		iv.listenPort.show(strconv.Itoa(int(c.ListenPort)))
	} else {
		iv.listenPort.hide()
	}

	if c.Mtu > 0 {
		iv.mtu.show(strconv.Itoa(int(c.Mtu)))
	} else {
		iv.mtu.hide()
	}

	if len(c.Addresses) > 0 {
		addrStrings := make([]string, len(c.Addresses))
		for i, address := range c.Addresses {
			addrStrings[i] = address.String()
		}
		iv.addresses.show(strings.Join(addrStrings[:], ", "))
	} else {
		iv.addresses.hide()
	}

	if len(c.Dns) > 0 {
		addrStrings := make([]string, len(c.Dns))
		for i, address := range c.Dns {
			addrStrings[i] = address.String()
		}
		iv.dns.show(strings.Join(addrStrings[:], ", "))
	} else {
		iv.dns.hide()
	}
}

func (pv *peerView) widgetsLines() []widgetsLine {
	return pv.lines
}

func (pv *peerView) apply(c *conf.Peer) {
	pv.publicKey.show(c.PublicKey.String())

	if !c.PresharedKey.IsZero() {
		pv.presharedKey.show("enabled")
	} else {
		pv.presharedKey.hide()
	}

	if len(c.AllowedIPs) > 0 {
		addrStrings := make([]string, len(c.AllowedIPs))
		for i, address := range c.AllowedIPs {
			addrStrings[i] = address.String()
		}
		pv.allowedIPs.show(strings.Join(addrStrings[:], ", "))
	} else {
		pv.allowedIPs.hide()
	}

	if !c.Endpoint.IsEmpty() {
		pv.endpoint.show(c.Endpoint.String())
	} else {
		pv.endpoint.hide()
	}

	if c.PersistentKeepalive > 0 {
		pv.persistentKeepalive.show(strconv.Itoa(int(c.PersistentKeepalive)))
	} else {
		pv.persistentKeepalive.hide()
	}

	if !c.LastHandshakeTime.IsEmpty() {
		pv.latestHandshake.show(c.LastHandshakeTime.String())
	} else {
		pv.latestHandshake.hide()
	}

	if c.RxBytes > 0 || c.TxBytes > 0 {
		pv.transfer.show(fmt.Sprintf("%s received, %s sent", c.RxBytes.String(), c.TxBytes.String()))
	} else {
		pv.transfer.hide()
	}
}

func newPaddedGroupGrid(parent walk.Container) (group *walk.GroupBox, err error) {
	group, err = walk.NewGroupBox(parent)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			group.Dispose()
		}
	}()
	layout := walk.NewGridLayout()
	layout.SetMargins(walk.Margins{10, 5, 10, 5})
	layout.SetSpacing(0)
	err = group.SetLayout(layout)
	if err != nil {
		return nil, err
	}
	spacer, err := walk.NewSpacerWithCfg(group, &walk.SpacerCfg{walk.GrowableHorz | walk.GreedyHorz, walk.Size{10, 0}, false})
	if err != nil {
		return nil, err
	}
	layout.SetRange(spacer, walk.Rectangle{1, 0, 1, 1})
	return group, nil
}

func NewConfView(parent walk.Container) (*ConfView, error) {
	cv := new(ConfView)
	cv.ScrollView, _ = walk.NewScrollView(parent)
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 0, 5, 0})
	cv.SetLayout(vlayout)
	cv.name, _ = newPaddedGroupGrid(cv)
	cv.interfaze = newInterfaceView(cv.name)
	cv.interfaze.toggleActive.button.Clicked().Attach(cv.onToggleActiveClicked)
	cv.peers = make(map[conf.Key]*peerView)
	cv.tunnelChangedCB = service.IPCClientRegisterTunnelChange(cv.onTunnelChanged)
	cv.SetTunnel(nil)
	globalState, _ := service.IPCClientGlobalState()
	cv.interfaze.toggleActive.updateGlobal(globalState)

	if err := walk.InitWrapperWindow(cv); err != nil {
		return nil, err
	}
	cv.SetDoubleBuffering(true)
	cv.updateTicker = time.NewTicker(time.Second)
	go func() {
		for range cv.updateTicker.C {
			if cv.tunnel != nil {
				tunnel := cv.tunnel
				var state service.TunnelState
				var config conf.Config
				if state, _ = tunnel.State(); state == service.TunnelStarted {
					config, _ = tunnel.RuntimeConfig()
				}
				if config.Name == "" {
					config, _ = tunnel.StoredConfig()
				}
				cv.Synchronize(func() {
					cv.setTunnel(tunnel, &config, state)
				})
			}
		}
	}()
	return cv, nil
}

func (cv *ConfView) Dispose() {
	if cv.tunnelChangedCB != nil {
		cv.tunnelChangedCB.Unregister()
		cv.tunnelChangedCB = nil
	}
	if cv.updateTicker != nil {
		cv.updateTicker.Stop()
		cv.updateTicker = nil
	}
	cv.ScrollView.Dispose()
}

func (cv *ConfView) onToggleActiveClicked() {
	cv.interfaze.toggleActive.button.SetEnabled(false)
	go func() {
		oldState, err := cv.tunnel.Toggle()
		if err != nil {
			cv.Synchronize(func() {
				if oldState == service.TunnelUnknown {
					walk.MsgBox(cv.Form(), "Failed to determine tunnel state", err.Error(), walk.MsgBoxIconError)
				} else if oldState == service.TunnelStopped {
					walk.MsgBox(cv.Form(), "Failed to activate tunnel", err.Error(), walk.MsgBoxIconError)
				} else if oldState == service.TunnelStarted {
					walk.MsgBox(cv.Form(), "Failed to deactivate tunnel", err.Error(), walk.MsgBoxIconError)
				}
			})
		}
	}()
}

func (cv *ConfView) onTunnelChanged(tunnel *service.Tunnel, state service.TunnelState, globalState service.TunnelState, err error) {
	cv.Synchronize(func() {
		cv.interfaze.toggleActive.updateGlobal(globalState)
		if cv.tunnel != nil && cv.tunnel.Name == tunnel.Name {
			cv.interfaze.status.update(state)
			cv.interfaze.toggleActive.update(state)
		}
	})
	if cv.tunnel != nil && cv.tunnel.Name == tunnel.Name {
		var config conf.Config
		if state == service.TunnelStarted {
			config, _ = tunnel.RuntimeConfig()
		}
		if config.Name == "" {
			config, _ = tunnel.StoredConfig()
		}
		cv.Synchronize(func() {
			cv.setTunnel(tunnel, &config, state)
		})
	}
}

func (cv *ConfView) SetTunnel(tunnel *service.Tunnel) {
	cv.tunnel = tunnel //XXX: This races with the read in the updateTicker, but it's pointer-sized!

	var config conf.Config
	var state service.TunnelState
	if tunnel != nil {
		title := "Interface: " + tunnel.Name
		if title != cv.name.Title() {
			//TODO: display some sort of loading screen here!
		}
		go func() {
			if state, _ = tunnel.State(); state == service.TunnelStarted {
				config, _ = tunnel.RuntimeConfig()
			}
			if config.Name == "" {
				config, _ = tunnel.StoredConfig()
			}
			cv.Synchronize(func() {
				cv.setTunnel(tunnel, &config, state)
			})
		}()
	} else {
		cv.setTunnel(tunnel, &config, state)
	}
}

func (cv *ConfView) setTunnel(tunnel *service.Tunnel, config *conf.Config, state service.TunnelState) {
	if !(cv.tunnel == nil || tunnel == nil || tunnel.Name == cv.tunnel.Name) {
		return
	}

	cv.name.SetVisible(tunnel != nil)

	hasSuspended := false
	suspend := func() {
		if !hasSuspended {
			cv.SetSuspended(true)
			hasSuspended = true
		}
	}
	defer func() {
		if hasSuspended {
			cv.SetSuspended(false)
		}
	}()
	title := "Interface: " + config.Name
	if cv.name.Title() != title {
		cv.name.SetTitle(title)
	}
	cv.interfaze.apply(&config.Interface)
	cv.interfaze.status.update(state)
	cv.interfaze.toggleActive.update(state)
	inverse := make(map[*peerView]bool, len(cv.peers))
	for _, pv := range cv.peers {
		inverse[pv] = true
	}
	for _, peer := range config.Peers {
		if pv := cv.peers[peer.PublicKey]; pv != nil {
			pv.apply(&peer)
			inverse[pv] = false
		} else {
			suspend()
			group, _ := newPaddedGroupGrid(cv)
			group.SetTitle("Peer")
			pv := newPeerView(group)
			pv.apply(&peer)
			cv.peers[peer.PublicKey] = pv
		}
	}
	for pv, remove := range inverse {
		if !remove {
			continue
		}
		k, e := conf.NewPrivateKeyFromString(pv.publicKey.text.Text())
		if e != nil {
			continue
		}
		suspend()
		delete(cv.peers, *k)
		groupBox := pv.publicKey.label.Parent().AsContainerBase().Parent().(*walk.GroupBox)
		groupBox.Parent().Children().Remove(groupBox)
		groupBox.Dispose()
	}
}
