package java

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"mc-server-checker/internal/domain"
)

const pingFailureWarning = "Statusは取得できましたが、Pingの計測に失敗しました。"

var (
	ErrTrailingPacketData = errors.New("packet contains trailing data")
	ErrPongNonceMismatch  = errors.New("pong nonce does not match ping")
)

type Client struct {
	ProtocolVersion int32
	Codec           Codec
	Dialer          *net.Dialer
	Resolver        SRVResolver
	Now             func() time.Time
}

type SRVResolver interface {
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}

func NewClient(protocolVersion int32) *Client {
	return &Client{
		ProtocolVersion: protocolVersion,
		Codec:           NewCodec(),
		Dialer:          &net.Dialer{Timeout: 3 * time.Second},
		Resolver:        net.DefaultResolver,
		Now:             time.Now,
	}
}

func (c *Client) Check(ctx context.Context, target domain.Target) (domain.Result, error) {
	dialer := c.Dialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 3 * time.Second}
	}
	endpoint, resolvedBySRV, err := c.resolveEndpoint(ctx, target)
	if err != nil {
		return domain.Result{}, err
	}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint.Address())
	if err != nil {
		return domain.Result{}, err
	}
	defer conn.Close()
	stopCancelClose := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopCancelClose()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return domain.Result{}, err
		}
	}

	codec := c.Codec
	handshakeTarget := target
	handshakeTarget.Port = endpoint.Port
	if err := c.writeHandshake(codec, conn, handshakeTarget); err != nil {
		return domain.Result{}, fmt.Errorf("write handshake: %w", err)
	}
	if err := codec.WritePacket(conn, 0x00, nil); err != nil {
		return domain.Result{}, fmt.Errorf("write status request: %w", err)
	}
	statusPacket, err := codec.ReadPacket(conn, 0x00)
	if err != nil {
		return domain.Result{}, fmt.Errorf("read status response: %w", err)
	}
	statusReader := bytes.NewReader(statusPacket)
	statusJSON, err := codec.ReadString(statusReader)
	if err != nil {
		return domain.Result{}, fmt.Errorf("read status json: %w", err)
	}
	if statusReader.Len() != 0 {
		return domain.Result{}, ErrTrailingPacketData
	}
	payload, motd, modInfo, err := DecodeStatus([]byte(statusJSON), NewMOTDNormalizer())
	if err != nil {
		return domain.Result{}, err
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}
	result := domain.Result{
		Target:          target,
		Status:          domain.StatusOnline,
		VersionName:     payload.Version.Name,
		Protocol:        payload.Version.Protocol,
		PlayersOnline:   payload.Players.Online,
		PlayersMax:      payload.Players.Max,
		MOTD:            motd,
		CheckedAt:       now(),
		ModInfoDetected: modInfo.Detected,
		ModLoader:       modInfo.Loader,
		ModCount:        modInfo.Count,
		ModInfoWarning:  modInfo.Warning,
	}
	if resolvedBySRV {
		result.ResolvedTarget = &endpoint
	}

	latency, err := c.ping(codec, conn, now)
	if err != nil {
		result.Warning = pingFailureWarning
		return result, nil
	}
	result.Latency = &latency
	return result, nil
}

func (c *Client) resolveEndpoint(ctx context.Context, target domain.Target) (domain.Target, bool, error) {
	if !target.UseSRV {
		return domain.Target{Host: target.Host, Port: target.Port}, false, nil
	}

	resolver := c.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	_, records, _ := resolver.LookupSRV(ctx, "minecraft", "tcp", strings.TrimSuffix(target.Host, "."))
	if len(records) > 0 {
		record := records[0]
		host := strings.TrimSuffix(record.Target, ".")
		if host != "" && record.Port != 0 {
			return domain.Target{Host: host, Port: record.Port}, true, nil
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return domain.Target{}, false, ctxErr
	}
	return domain.Target{Host: target.Host, Port: target.Port}, false, nil
}

func (c *Client) writeHandshake(codec Codec, conn net.Conn, target domain.Target) error {
	var payload bytes.Buffer
	payload.Write(EncodeVarInt(c.ProtocolVersion))
	if err := codec.WriteString(&payload, target.Host); err != nil {
		return err
	}
	if err := WriteUint16(&payload, target.Port); err != nil {
		return err
	}
	payload.Write(EncodeVarInt(1))
	return codec.WritePacket(conn, 0x00, payload.Bytes())
}

func (c *Client) ping(codec Codec, conn net.Conn, now func() time.Time) (time.Duration, error) {
	var payload bytes.Buffer
	nonce := now().UnixNano()
	if err := WriteInt64(&payload, nonce); err != nil {
		return 0, err
	}
	started := now()
	if err := codec.WritePacket(conn, 0x01, payload.Bytes()); err != nil {
		return 0, err
	}
	pong, err := codec.ReadPacket(conn, 0x01)
	if err != nil {
		return 0, err
	}
	if len(pong) != 8 {
		return 0, ErrTrailingPacketData
	}
	gotNonce, err := ReadInt64(bytes.NewReader(pong))
	if err != nil {
		return 0, err
	}
	if gotNonce != nonce {
		return 0, ErrPongNonceMismatch
	}
	latency := now().Sub(started)
	if latency < 0 {
		latency = 0
	}
	return latency, nil
}
