package java

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"mc-server-checker/internal/domain"
)

func TestClientResolvesMinecraftSRVWhenPortIsOmitted(t *testing.T) {
	t.Parallel()
	resolver := &fakeSRVResolver{
		records: []*net.SRV{{Target: "backend.example.net.", Port: 25570, Priority: 0, Weight: 10}},
	}
	client := NewClient(769)
	client.Resolver = resolver

	address, port, err := client.resolveEndpoint(context.Background(), domain.Target{
		Host: "play.example.com", Port: domain.DefaultPort, UseSRV: true,
	})
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if address != "backend.example.net:25570" || port != 25570 {
		t.Fatalf("endpoint = %q, %d", address, port)
	}
	if resolver.service != "minecraft" || resolver.proto != "tcp" || resolver.name != "play.example.com" {
		t.Fatalf("lookup = %q, %q, %q", resolver.service, resolver.proto, resolver.name)
	}
}

func TestClientCheckConnectsThroughSRVEndpoint(t *testing.T) {
	t.Parallel()
	address, stop := startFakeServerWithExpectedHost(t, true, "play.example.com")
	defer stop()
	_, port := splitTarget(t, address)

	client := NewClient(769)
	client.Resolver = &fakeSRVResolver{
		records: []*net.SRV{{Target: "127.0.0.1.", Port: port}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := client.Check(ctx, domain.Target{
		Host: "play.example.com", Port: domain.DefaultPort, UseSRV: true,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Status != domain.StatusOnline {
		t.Fatalf("result = %+v", result)
	}
}

func TestClientFallsBackToDefaultPortWithoutSRV(t *testing.T) {
	t.Parallel()
	resolver := &fakeSRVResolver{err: errors.New("record not found")}
	client := NewClient(769)
	client.Resolver = resolver

	address, port, err := client.resolveEndpoint(context.Background(), domain.Target{
		Host: "play.example.com", Port: domain.DefaultPort, UseSRV: true,
	})
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if address != "play.example.com:25565" || port != domain.DefaultPort {
		t.Fatalf("endpoint = %q, %d", address, port)
	}
}

func TestClientSkipsSRVForExplicitPort(t *testing.T) {
	t.Parallel()
	resolver := &fakeSRVResolver{records: []*net.SRV{{Target: "unexpected.example.net.", Port: 25570}}}
	client := NewClient(769)
	client.Resolver = resolver

	address, port, err := client.resolveEndpoint(context.Background(), domain.Target{
		Host: "play.example.com", Port: 25566,
	})
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if address != "play.example.com:25566" || port != 25566 || resolver.calls != 0 {
		t.Fatalf("endpoint = %q, %d; lookup calls = %d", address, port, resolver.calls)
	}
}

func TestClientCheckAgainstFakeServer(t *testing.T) {
	t.Parallel()
	address, stop := startFakeServer(t, true)
	defer stop()
	host, port := splitTarget(t, address)

	client := NewClient(769)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := client.Check(ctx, domain.Target{Host: host, Port: port})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Status != domain.StatusOnline || result.Latency == nil || result.Warning != "" {
		t.Fatalf("result status fields = %+v", result)
	}
	if result.VersionName != "Fake 1.21" || result.PlayersOnline == nil || *result.PlayersOnline != 2 || result.PlayersMax == nil || *result.PlayersMax != 10 || result.MOTD != "Fake Server" {
		t.Fatalf("result values = %+v", result)
	}
}

func TestClientKeepsOnlineWhenPongFails(t *testing.T) {
	t.Parallel()
	address, stop := startFakeServer(t, false)
	defer stop()
	host, port := splitTarget(t, address)

	client := NewClient(769)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	result, err := client.Check(ctx, domain.Target{Host: host, Port: port})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Status != domain.StatusOnline || result.Latency != nil || result.Warning == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestClientCancellationClosesActiveConnection(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()
	requestRead := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		codec := NewCodec()
		if _, err := codec.ReadPacket(conn, 0x00); err != nil {
			return
		}
		if _, err := codec.ReadPacket(conn, 0x00); err != nil {
			return
		}
		close(requestRead)
		var one [1]byte
		_, _ = conn.Read(one[:])
	}()

	host, port := splitTarget(t, listener.Addr().String())
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := NewClient(769).Check(ctx, domain.Target{Host: host, Port: port})
		result <- err
	}()
	select {
	case <-requestRead:
	case <-time.After(time.Second):
		t.Fatal("fake server did not receive request")
	}
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Check returned nil error after cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Check did not stop promptly after cancellation")
	}
	select {
	case <-serverDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("server connection remained open after cancellation")
	}
}

func startFakeServer(t *testing.T, sendPong bool) (string, func()) {
	return startFakeServerWithExpectedHost(t, sendPong, "127.0.0.1")
}

func startFakeServerWithExpectedHost(t *testing.T, sendPong bool, expectedHost string) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	_, expectedPort := splitTarget(t, listener.Addr().String())
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		codec := NewCodec()
		handshake, err := codec.ReadPacket(conn, 0x00)
		if err != nil {
			t.Errorf("read handshake: %v", err)
			return
		}
		if err := validateHandshake(codec, handshake, expectedHost, expectedPort); err != nil {
			t.Errorf("validate handshake: %v", err)
			return
		}
		request, err := codec.ReadPacket(conn, 0x00)
		if err != nil || len(request) != 0 {
			t.Errorf("read status request: payload=%v err=%v", request, err)
			return
		}
		status := `{"version":{"name":"Fake 1.21","protocol":769},"players":{"online":2,"max":10},"description":{"text":"Fake Server"}}`
		var response bytes.Buffer
		if err := codec.WriteString(&response, status); err != nil {
			t.Errorf("encode status: %v", err)
			return
		}
		if err := codec.WritePacket(conn, 0x00, response.Bytes()); err != nil {
			t.Errorf("write status: %v", err)
			return
		}
		ping, err := codec.ReadPacket(conn, 0x01)
		if err != nil {
			return
		}
		if sendPong {
			if err := codec.WritePacket(conn, 0x01, ping); err != nil {
				t.Errorf("write pong: %v", err)
			}
		}
	}()
	return listener.Addr().String(), func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("fake server did not stop")
		}
	}
}

func validateHandshake(codec Codec, payload []byte, expectedHost string, expectedPort uint16) error {
	r := bytes.NewReader(payload)
	protocol, err := DecodeVarInt(r)
	if err != nil {
		return err
	}
	host, err := codec.ReadString(r)
	if err != nil {
		return err
	}
	port, err := ReadUint16(r)
	if err != nil {
		return err
	}
	nextState, err := DecodeVarInt(r)
	if err != nil {
		return err
	}
	if protocol != 769 || host != expectedHost || port != expectedPort || nextState != 1 || r.Len() != 0 {
		return fmt.Errorf("unexpected handshake: protocol=%d host=%q port=%d next=%d trailing=%d", protocol, host, port, nextState, r.Len())
	}
	return nil
}

func splitTarget(t *testing.T, address string) (string, uint16) {
	t.Helper()
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	var port uint16
	if _, err := fmt.Sscanf(portText, "%d", &port); err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

type fakeSRVResolver struct {
	records []*net.SRV
	err     error
	calls   int
	service string
	proto   string
	name    string
}

func (r *fakeSRVResolver) LookupSRV(_ context.Context, service, proto, name string) (string, []*net.SRV, error) {
	r.calls++
	r.service = service
	r.proto = proto
	r.name = name
	return "", r.records, r.err
}
