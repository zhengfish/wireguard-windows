/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package service

import (
	"bytes"
	"encoding/gob"
	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows/svc"
	"golang.zx2c4.com/wireguard/windows/conf"
	"io/ioutil"
	"net/rpc"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var managerServices = make(map[*ManagerService]bool)
var managerServicesLock sync.RWMutex
var haveQuit uint32
var quitManagersChan = make(chan struct{}, 1)

type ManagerService struct {
	events *os.File
}

func (s *ManagerService) StoredConfig(tunnelName string, config *conf.Config) error {
	c, err := conf.LoadFromName(tunnelName)
	if err != nil {
		return err
	}
	*config = *c
	return nil
}

func (s *ManagerService) RuntimeConfig(tunnelName string, config *conf.Config) error {
	storedConfig, err := conf.LoadFromName(tunnelName)
	if err != nil {
		return err
	}
	pipePath, err := PipePathOfTunnel(storedConfig.Name)
	if err != nil {
		return err
	}
	pipe, err := winio.DialPipe(pipePath, nil)
	if err != nil {
		return err
	}
	pipe.SetWriteDeadline(time.Now().Add(time.Second * 2))
	_, err = pipe.Write([]byte("get=1\n\n"))
	if err != nil {
		return err
	}
	pipe.SetReadDeadline(time.Now().Add(time.Second * 2))
	resp, err := ioutil.ReadAll(pipe)
	if err != nil {
		return err
	}
	pipe.Close()
	runtimeConfig, err := conf.FromUAPI(string(resp), storedConfig)
	if err != nil {
		return err
	}
	*config = *runtimeConfig
	return nil
}

func (s *ManagerService) Start(tunnelName string, unused *uintptr) error {
	trackedTunnelsLock.Lock()
	tt := make([]string, 0, len(trackedTunnels))
	for t := range trackedTunnels {
		tt = append(tt, t)
	}
	trackedTunnelsLock.Unlock()
	for _, t := range tt {
		s.Stop(t, unused)
		s.WaitForStop(t, unused)
	}

	// After that process is started -- it's somewhat asynchronous -- we install the new one.
	c, err := conf.LoadFromName(tunnelName)
	if err != nil {
		return err
	}
	path, err := c.Path()
	if err != nil {
		return err
	}
	return InstallTunnel(path)
}

func (s *ManagerService) Stop(tunnelName string, unused *uintptr) error {
	err := UninstallTunnel(tunnelName)
	if err == syscall.Errno(serviceDOES_NOT_EXIST) {
		_, notExistsError := conf.LoadFromName(tunnelName)
		if notExistsError == nil {
			return nil
		}
	}
	return err
}

func (s *ManagerService) WaitForStop(tunnelName string, unused *uintptr) error {
	serviceName, err := ServiceNameOfTunnel(tunnelName)
	if err != nil {
		return err
	}
	m, err := serviceManager()
	if err != nil {
		return err
	}
	for {
		service, err := m.OpenService(serviceName)
		if err == nil || err == syscall.Errno(serviceMARKED_FOR_DELETE) {
			service.Close()
			time.Sleep(time.Second / 3)
		} else {
			return nil
		}
	}
}

func (s *ManagerService) Delete(tunnelName string, unused *uintptr) error {
	err := s.Stop(tunnelName, nil)
	if err != nil {
		return err
	}
	return conf.DeleteName(tunnelName)
}

func (s *ManagerService) State(tunnelName string, state *TunnelState) error {
	serviceName, err := ServiceNameOfTunnel(tunnelName)
	if err != nil {
		return err
	}
	m, err := serviceManager()
	if err != nil {
		return err
	}
	service, err := m.OpenService(serviceName)
	if err != nil {
		*state = TunnelStopped
		return nil
	}
	defer service.Close()
	status, err := service.Query()
	if err != nil {
		*state = TunnelUnknown
		return err
	}
	switch status.State {
	case svc.Stopped:
		*state = TunnelStopped
	case svc.StopPending:
		*state = TunnelStopping
	case svc.Running:
		*state = TunnelStarted
	case svc.StartPending:
		*state = TunnelStarting
	default:
		*state = TunnelUnknown
	}
	return nil
}

func (s *ManagerService) GlobalState(unused uintptr, state *TunnelState) error {
	*state = trackedTunnelsGlobalState()
	return nil
}

func (s *ManagerService) Create(tunnelConfig conf.Config, tunnel *Tunnel) error {
	err := tunnelConfig.Save()
	if err != nil {
		return err
	}
	*tunnel = Tunnel{tunnelConfig.Name}
	return nil
	//TODO: handle already existing situation
	//TODO: handle already running and existing situation
}

func (s *ManagerService) Tunnels(unused uintptr, tunnels *[]Tunnel) error {
	names, err := conf.ListConfigNames()
	if err != nil {
		return err
	}
	*tunnels = make([]Tunnel, len(names))
	for i := 0; i < len(*tunnels); i++ {
		(*tunnels)[i].Name = names[i]
	}
	return nil
	//TODO: account for running ones that aren't in the configuration store somehow
}

func (s *ManagerService) Quit(stopTunnelsOnQuit bool, alreadyQuit *bool) error {
	if !atomic.CompareAndSwapUint32(&haveQuit, 0, 1) {
		*alreadyQuit = true
		return nil
	}
	*alreadyQuit = false

	// Work around potential race condition of delivering messages to the wrong process by removing from notifications.
	managerServicesLock.Lock()
	delete(managerServices, s)
	managerServicesLock.Unlock()

	if stopTunnelsOnQuit {
		names, err := conf.ListConfigNames()
		if err != nil {
			return err
		}
		for _, name := range names {
			UninstallTunnel(name)
		}
	}

	quitManagersChan <- struct{}{}
	return nil
}

func IPCServerListen(reader *os.File, writer *os.File, events *os.File) error {
	service := &ManagerService{events: events}

	server := rpc.NewServer()
	err := server.Register(service)
	if err != nil {
		return err
	}

	go func() {
		managerServicesLock.Lock()
		managerServices[service] = true
		managerServicesLock.Unlock()
		server.ServeConn(&pipeRWC{reader, writer})
		managerServicesLock.Lock()
		delete(managerServices, service)
		managerServicesLock.Unlock()

	}()
	return nil
}

func notifyAll(notificationType NotificationType, ifaces ...interface{}) {
	if len(managerServices) == 0 {
		return
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(notificationType)
	if err != nil {
		return
	}
	for _, iface := range ifaces {
		err = encoder.Encode(iface)
		if err != nil {
			return
		}
	}

	managerServicesLock.RLock()
	for m := range managerServices {
		m.events.SetWriteDeadline(time.Now().Add(time.Second))
		m.events.Write(buf.Bytes())
	}
	managerServicesLock.RUnlock()
}

func IPCServerNotifyTunnelChange(name string, state TunnelState, err error) {
	if err == nil {
		notifyAll(TunnelChangeNotificationType, name, state, "")
	} else {
		notifyAll(TunnelChangeNotificationType, name, state, err.Error())
	}
}

func IPCServerNotifyTunnelsChange() {
	notifyAll(TunnelsChangeNotificationType)
}
