package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/api"
	"google.golang.org/grpc"
)

func main() {
	l := logrus.New()

	conn, err := grpc.Dial("localhost:3021",
		grpc.WithInsecure(),
		// grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		// 	return net.DialTimeout("unix", "/tmp/nebula.sock", 1*time.Second)
		// }),
	)
	if err != nil {
		l.WithError(err).Fatal()
	}
	defer conn.Close()

	client := api.NewNebulaControlClient(conn)

	flag.Parse()
	args := flag.Args()

	switch args[0] {
	case "list-hostmap", "list", "l":
		err = listHostmap(client, args[1:])
	case "hostinfo":
		err = hostinfo(client, args[1:])
	case "set-remote":
		err = setRemote(client, args[1:])
	}

	if err != nil {
		l.WithError(err).Fatal()
	}

}

func hostinfo(client api.NebulaControlClient, args []string) error {
	// TODO flags

	r, err := client.GetHostInfo(context.Background(), &api.GetHostInfoParams{
		VpnIP: args[0],
	})
	if err != nil {
		return err
	}

	j, err := json.Marshal(r)
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	return nil
}

func setRemote(client api.NebulaControlClient, args []string) error {
	// TODO flags

	r, err := client.SetRemote(context.Background(), &api.SetRemoteParams{
		VpnIP:   args[0],
		UdpAddr: args[1],
	})
	if err != nil {
		return err
	}

	j, err := json.Marshal(r)
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	return nil
}

func listHostmap(client api.NebulaControlClient, args []string) error {
	// TODO flags

	r, err := client.ListHostmap(context.Background(), &api.ListHostmapParams{})
	if err != nil {
		return err
	}

	for {
		hostInfo, err := r.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		j, err := json.Marshal(hostInfo)
		if err != nil {
			return err
		}

		fmt.Println(string(j))
	}

	return nil
}
