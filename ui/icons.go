/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"image/color"
)

var (
	logoIcon,
	connectedLogoIcon,
	connectingLogoIcon,
	disconnectedLogoIcon,
	connectedIcon,
	connectingIcon,
	addTunnelIcon,
	fromFileIcon,
	fromScratchIcon,
	exportTunnelsIcon,
	editConfigIcon,
	saveConfigIcon,
	deleteConfigIcon,
	_ *walk.Icon
)

func loadSystemIcon(dll string, index uint) (*walk.Icon, error) {
	//TODO: Replace with GetSystemDirectory when https://go-review.googlesource.com/c/sys/+/165759 is merged.
	filename := fmt.Sprintf("C:\\Windows\\System32\\%s.dll", dll)
	hicon := win.ExtractIcon(win.GetModuleHandle(nil), windows.StringToUTF16Ptr(filename), int32(index))
	if hicon <= 1 {
		return nil, fmt.Errorf("Unable to find icon %d of %s", index, dll)
	}
	return walk.NewIconFromHICON(hicon)
}

//TODO: This combination business looks awful. Replace with real composed icons.
func combineIcons(icon1, icon2 *walk.Icon) (*walk.Icon, error) {
	bitmap1, err := walk.NewBitmapFromIcon(icon1, walk.Size{64, 64})
	if err != nil {
		return nil, err
	}
	bitmap2, err := walk.NewBitmapFromIcon(icon2, walk.Size{64, 64})
	if err != nil {
		return nil, err
	}
	image1, err := bitmap1.ToImage()
	if err != nil {
		return nil, err
	}
	image2, err := bitmap2.ToImage()
	if err != nil {
		return nil, err
	}
	bitmap1.Dispose()
	bitmap2.Dispose()
	icon1.Dispose()
	// We don't dispose icon2.

	black := color.RGBA{0, 0, 0, 255}
	for x := 0; x < image2.Rect.Max.X; x++ {
		for y := 0; y < image2.Rect.Max.Y; y++ {
			if image2.RGBAAt(x, y) == black {
				image2.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			} else if image1.At(x, y) != black {
				image2.SetRGBA(x, y, image1.RGBAAt(x, y))
			}
		}
	}

	return walk.NewIconFromImage(image2)
}

func loadIcons() (err error) {
	logoIcon, err = walk.NewIconFromResourceId(1)
	if err != nil {
		return
	}

	connectedIcon, err = loadSystemIcon("imageres", 227)
	if err != nil {
		return
	}
	connectingIcon, err = loadSystemIcon("imageres", 228)
	if err != nil {
		return
	}

	disconnectedLogoIcon, err = loadSystemIcon("imageres", 92)
	if err != nil {
		return
	}
	connectedLogoIcon, err = loadSystemIcon("imageres", 148)
	if err != nil {
		return
	}
	connectingLogoIcon, err = loadSystemIcon("imageres", 219)
	if err != nil {
		return
	}

	addTunnelIcon, err = loadSystemIcon("netcenter", 11)
	if err != nil {
		return
	}
	exportTunnelsIcon, err = loadSystemIcon("imageres", 266)
	if err != nil {
		return
	}

	fromFileIcon, err = loadSystemIcon("shell32", 275)
	if err != nil {
		return
	}
	fromScratchIcon, err = loadSystemIcon("shell32", 69)
	if err != nil {
		return
	}

	editConfigIcon, err = loadSystemIcon("shell32", 269)
	if err != nil {
		return
	}
	saveConfigIcon, err = loadSystemIcon("shell32", 258)
	if err != nil {
		return
	}
	deleteConfigIcon, err = loadSystemIcon("shell32", 152)
	if err != nil {
		return
	}

	//TODO: This combination business looks awful. Replace with real composed icons.
	disconnectedLogoIcon, err = combineIcons(disconnectedLogoIcon, logoIcon)
	if err != nil {
		return
	}
	connectedLogoIcon, err = combineIcons(connectedLogoIcon, logoIcon)
	if err != nil {
		return
	}
	connectingLogoIcon, err = combineIcons(connectingLogoIcon, logoIcon)
	if err != nil {
		return
	}

	return
}
