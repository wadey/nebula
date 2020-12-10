package nebula

import (
	"context"
	"errors"
	"net"

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
