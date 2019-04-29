module golang.zx2c4.com/wireguard/windows

require (
	github.com/Microsoft/go-winio v0.4.12
	github.com/lxn/walk v0.0.0-00010101000000-000000000000
	github.com/lxn/win v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.8.1 // indirect
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/net v0.0.0-20190424112056-4829fb13d2c6
	golang.org/x/sys v0.0.0-20190428183149-804c0c7841b5
	golang.zx2c4.com/winipcfg v0.0.0-20190425094732-ce756128240c
	golang.zx2c4.com/wireguard v0.0.0-20190429093702-4280e7ee4d8d
)

replace (
	github.com/Microsoft/go-winio => golang.zx2c4.com/wireguard/windows v0.0.0-20190429060359-b01600290cd4
	github.com/lxn/walk => golang.zx2c4.com/wireguard/windows v0.0.0-20190427141626-87d5c8c7119e
	github.com/lxn/win => golang.zx2c4.com/wireguard/windows v0.0.0-20190427131707-bdeea63ee954
)
