package nebula

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes"
	"github.com/slackhq/nebula/api"
	"google.golang.org/grpc"
)

type NebulaAPI struct {
	api.UnimplementedNebulaControlServer
	Control *Control
}

func (n *NebulaAPI) GetVersion(context.Context, *api.GetVersionParams) (*api.Version, error) {
	return &api.Version{
		Version: n.Control.f.version,
	}, nil
}

func (n *NebulaAPI) GetHostInfo(ctx context.Context, p *api.GetHostInfoParams) (*api.HostInfo, error) {
	vpnIP := p.GetVpnIP()
	ip := net.ParseIP(vpnIP)
	if ip == nil {
		return nil, errors.New("invalid vpnIP")
	}
	hostInfo := n.Control.GetHostInfoByVpnIP(ip2int(ip), false)
	if hostInfo == nil {
		return &api.HostInfo{}, nil
	}

	return apiHostInfo(hostInfo), nil
}

func (n *NebulaAPI) SetRemote(ctx context.Context, p *api.SetRemoteParams) (*api.HostInfo, error) {
	vpnIP := p.GetVpnIP()
	ip := net.ParseIP(vpnIP)
	if ip == nil {
		return nil, errors.New("invalid vpnIP")
	}
	udpAddr := NewUDPAddrFromString(p.UdpAddr)
	hostInfo := n.Control.SetRemoteForTunnel(ip2int(ip), *udpAddr)

	return apiHostInfo(hostInfo), nil
}

func (n *NebulaAPI) ListHostmap(p *api.ListHostmapParams, s api.NebulaControl_ListHostmapServer) error {
	hm := n.Control.ListHostmap(false)
	for i := range hm {
		err := s.Send(apiHostInfo(&hm[i]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *NebulaAPI) Ping(p *api.PingParams, s api.NebulaControl_PingServer) error {
	ifce := n.Control.f

	parsedIp := net.ParseIP(p.VpnIP)
	if parsedIp == nil {
		return fmt.Errorf("The provided vpn ip could not be parsed: %s", p.VpnIP)
	}

	vpnIp := ip2int(parsedIp)
	if vpnIp == 0 {
		return fmt.Errorf("The provided vpn ip could not be parsed: %s", p.VpnIP)
	}

	c := make(chan *api.DebugResult, 16)

	hostInfo, _ := ifce.hostMap.QueryVpnIP(uint32(vpnIp))
	if hostInfo != nil {
		hostInfo.debug = c
		hostInfo.debugMsg("tunnel already exists")
	} else {
		hostInfo, _ = ifce.handshakeManager.pendingHostMap.QueryVpnIP(uint32(vpnIp))
		if hostInfo != nil {
			hostInfo.debug = c
			hostInfo.debugMsg("tunnel already handshaking")
		} else {
			hostInfo = ifce.getOrHandshake(vpnIp)
			hostInfo.debug = c
			hostInfo.debugMsg("starting new handshake")
		}
	}

	// TODO allow to manually set the remote udpAddr
	// var addr *udpAddr
	// if flags.Address != "" {
	// 	addr = NewUDPAddrFromString(flags.Address)
	// 	if addr == nil {
	// 		return w.WriteLine("Address could not be parsed")
	// 	}
	// }

	// hostInfo = ifce.handshakeManager.AddVpnIP(vpnIp)
	// if addr != nil {
	// 	hostInfo.SetRemote(*addr)
	// }

	hi := ifce.getOrHandshake(vpnIp)
	if hi != hostInfo {
		return fmt.Errorf("hostInfo changed while we were starting")
	}
	defer func() {
		hi.debug = nil
	}()

	done := s.Context().Done()

	ifce.SendMessageToVpnIp(test, testRequest, vpnIp, []byte(""), make([]byte, 12, 12), make([]byte, mtu))
	hi.debugMsg("test packet sent")

	for {
		select {
		case result, ok := <-c:
			if !ok {
				return nil
			}
			if result.Timestamp == nil {
				result.Timestamp = ptypes.TimestampNow()
			}
			s.Send(result)

			// TODO let the client determine this
			if result.Message == "test packet received" {
				return nil
			}
		case <-done:
			err := s.Context().Err()
			if err == context.Canceled {
				return nil
			}
			return err
		}
	}
}

func apiHostInfo(hostInfo *ControlHostInfo) *api.HostInfo {
	remoteAddrs := make([]string, len(hostInfo.RemoteAddrs))
	for i, a := range hostInfo.RemoteAddrs {
		remoteAddrs[i] = a.String()
	}
	return &api.HostInfo{
		VpnIP:         hostInfo.VpnIP.String(),
		LocalIndex:    hostInfo.LocalIndex,
		RemoteIndex:   hostInfo.RemoteIndex,
		RemoteAddrs:   remoteAddrs,
		Cert:          hostInfo.Cert.Raw(),
		CurrentRemote: hostInfo.CurrentRemote.String(),
	}
}

func (n *NebulaAPI) Run() {
	// lis, err := net.Listen("unix", "/tmp/nebula.sock")
	lis, err := net.Listen("tcp", "localhost:3021")
	if err != nil {
		l.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	api.RegisterNebulaControlServer(grpcServer, n)
	grpcServer.Serve(lis)
}
