//go:build linux

package detector

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
)

const (
	nlmsgNoop  = 1
	nlmsgError = 2
	nlmsgDone  = 3

	nlmFRequest = 0x01
	nlmFDump    = 0x300

	rtmNewAddr  = 20
	rtmDelAddr  = 21
	rtmGetAddr  = 22
	rtmNewRoute = 24
	rtmDelRoute = 25
	rtmGetRoute = 26

	rtmgrpIPv6Ifaddr = 0x100
	rtmgrpIPv6Route  = 0x400

	afInet6 = 10

	rtaDst     = 1
	ifaAddress = 1
	ifaLocal   = 2
)

var nativeEndian = binary.NativeEndian

type netlinkDetector struct {
	cfg     config.Config
	ifindex int
	fd      int
	seq     uint32
}

func newNetlinkDetector(cfg config.Config) (Detector, error) {
	iface, err := net.InterfaceByName(cfg.Interface)
	if err != nil {
		return nil, err
	}

	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}

	groups := uint32(0)
	switch cfg.Mode {
	case config.ModePDRoute:
		groups = rtmgrpIPv6Route
	case config.ModeRAAddress:
		groups = rtmgrpIPv6Ifaddr
	}
	if err := syscall.Bind(fd, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK, Groups: groups}); err != nil {
		syscall.Close(fd)
		return nil, err
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return &netlinkDetector{cfg: cfg, ifindex: iface.Index, fd: fd}, nil
}

func (d *netlinkDetector) Snapshot(ctx context.Context) (prefix.Set, error) {
	switch d.cfg.Mode {
	case config.ModePDRoute:
		return d.snapshotRoutes(ctx)
	case config.ModeRAAddress:
		return d.snapshotAddresses(ctx)
	default:
		return nil, fmt.Errorf("unsupported mode %q", d.cfg.Mode)
	}
}

func (d *netlinkDetector) Wait(ctx context.Context) error {
	for {
		data, err := d.recv(ctx, 8192)
		if err != nil {
			return err
		}
		msgs, err := syscall.ParseNetlinkMessage(data)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			switch msg.Header.Type {
			case nlmsgNoop, nlmsgDone:
				continue
			case nlmsgError:
				return parseNetlinkError(msg.Data)
			default:
				if d.wantsEvent(msg.Header.Type) {
					return nil
				}
			}
		}
	}
}

func (d *netlinkDetector) Close() error {
	return syscall.Close(d.fd)
}

func (d *netlinkDetector) snapshotRoutes(ctx context.Context) (prefix.Set, error) {
	msgs, err := d.requestDump(ctx, rtmGetRoute, []byte{
		afInet6, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	})
	if err != nil {
		return nil, err
	}

	var items []netip.Prefix
	for _, msg := range msgs {
		if msg.Header.Type != rtmNewRoute {
			continue
		}
		c, ok := parseRouteCandidate(msg.Data)
		if !ok {
			continue
		}
		if p, ok := FilterRoute(c, d.cfg.PrefixLen); ok {
			items = append(items, p)
		}
	}
	return prefix.Normalize(items), nil
}

func (d *netlinkDetector) snapshotAddresses(ctx context.Context) (prefix.Set, error) {
	msgs, err := d.requestDump(ctx, rtmGetAddr, []byte{
		afInet6, 0, 0, 0,
		0, 0, 0, 0,
	})
	if err != nil {
		return nil, err
	}

	var items []netip.Prefix
	for _, msg := range msgs {
		if msg.Header.Type != rtmNewAddr {
			continue
		}
		for _, c := range parseAddressCandidates(msg.Data) {
			if p, ok := FilterAddress(c, d.ifindex, d.cfg.PrefixLen); ok {
				items = append(items, p)
			}
		}
	}
	return prefix.Normalize(items), nil
}

func (d *netlinkDetector) requestDump(ctx context.Context, typ uint16, payload []byte) ([]syscall.NetlinkMessage, error) {
	d.seq++
	reqLen := syscall.NLMSG_HDRLEN + len(payload)
	req := make([]byte, reqLen)
	nativeEndian.PutUint32(req[0:4], uint32(reqLen))
	nativeEndian.PutUint16(req[4:6], typ)
	nativeEndian.PutUint16(req[6:8], nlmFRequest|nlmFDump)
	nativeEndian.PutUint32(req[8:12], d.seq)
	copy(req[syscall.NLMSG_HDRLEN:], payload)

	if err := syscall.Sendto(d.fd, req, 0, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}); err != nil {
		return nil, err
	}

	var out []syscall.NetlinkMessage
	for {
		data, err := d.recv(ctx, 65536)
		if err != nil {
			return nil, err
		}
		msgs, err := syscall.ParseNetlinkMessage(data)
		if err != nil {
			return nil, err
		}
		for _, msg := range msgs {
			if msg.Header.Seq != d.seq {
				continue
			}
			switch msg.Header.Type {
			case nlmsgDone:
				return out, nil
			case nlmsgError:
				return nil, parseNetlinkError(msg.Data)
			default:
				out = append(out, msg)
			}
		}
	}
}

func (d *netlinkDetector) wantsEvent(typ uint16) bool {
	switch d.cfg.Mode {
	case config.ModePDRoute:
		return typ == rtmNewRoute || typ == rtmDelRoute
	case config.ModeRAAddress:
		return typ == rtmNewAddr || typ == rtmDelAddr
	default:
		return false
	}
}

func (d *netlinkDetector) recv(ctx context.Context, initialSize int) ([]byte, error) {
	size := initialSize
	for {
		probe := make([]byte, size)
		n, _, flags, _, err := syscall.Recvmsg(d.fd, probe, nil, syscall.MSG_PEEK|syscall.MSG_TRUNC)
		switch {
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EAGAIN):
			if err := waitReadable(ctx, d.fd); err != nil {
				return nil, err
			}
			continue
		case err != nil:
			return nil, err
		}
		if flags&syscall.MSG_TRUNC != 0 && n > size {
			size = n
			continue
		}

		buf := make([]byte, n)
		n, _, flags, _, err = syscall.Recvmsg(d.fd, buf, nil, 0)
		switch {
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EAGAIN):
			if err := waitReadable(ctx, d.fd); err != nil {
				return nil, err
			}
			continue
		case err != nil:
			return nil, err
		}
		if flags&syscall.MSG_TRUNC != 0 {
			size *= 2
			continue
		}
		return buf[:n], nil
	}
}

func waitReadable(ctx context.Context, fd int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fds := &syscall.FdSet{}
		fdSet(fd, fds)
		timeout := syscall.NsecToTimeval((200 * time.Millisecond).Nanoseconds())
		n, err := syscall.Select(fd+1, fds, nil, nil, &timeout)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
	}
}

func fdSet(fd int, set *syscall.FdSet) {
	set.Bits[fd/64] |= 1 << (uint(fd) % 64)
}

func parseRouteCandidate(data []byte) (RouteCandidate, bool) {
	if len(data) < 12 || data[0] != afInet6 {
		return RouteCandidate{}, false
	}
	dstLen := int(data[1])
	c := RouteCandidate{
		DstPrefix: netip.PrefixFrom(netip.IPv6Unspecified(), dstLen),
		Protocol:  data[3],
		Type:      data[7],
	}
	for _, attr := range parseAttrs(data[12:]) {
		if attr.typ != rtaDst {
			continue
		}
		addr, ok := addrFromBytes(attr.data)
		if !ok {
			continue
		}
		c.DstPrefix = netip.PrefixFrom(addr, dstLen)
	}
	return c, true
}

func parseAddressCandidates(data []byte) []AddressCandidate {
	if len(data) < 8 || data[0] != afInet6 {
		return nil
	}
	ifindex := int(nativeEndian.Uint32(data[4:8]))
	var out []AddressCandidate
	for _, attr := range parseAttrs(data[8:]) {
		if attr.typ != ifaAddress && attr.typ != ifaLocal {
			continue
		}
		addr, ok := addrFromBytes(attr.data)
		if !ok {
			continue
		}
		out = append(out, AddressCandidate{Address: addr, IfIndex: ifindex})
	}
	return out
}

type attr struct {
	typ  uint16
	data []byte
}

func parseAttrs(data []byte) []attr {
	var out []attr
	for len(data) >= 4 {
		l := int(nativeEndian.Uint16(data[0:2]))
		if l < 4 || l > len(data) {
			break
		}
		out = append(out, attr{
			typ:  nativeEndian.Uint16(data[2:4]),
			data: data[4:l],
		})
		next := align4(l)
		if next > len(data) {
			break
		}
		data = data[next:]
	}
	return out
}

func align4(n int) int {
	return (n + 3) &^ 3
}

func addrFromBytes(b []byte) (netip.Addr, bool) {
	if len(b) != 16 {
		return netip.Addr{}, false
	}
	var raw [16]byte
	copy(raw[:], b)
	return netip.AddrFrom16(raw), true
}

func parseNetlinkError(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("netlink error")
	}
	code := int32(nativeEndian.Uint32(data[:4]))
	if code == 0 {
		return nil
	}
	if code < 0 {
		return syscall.Errno(-code)
	}
	return syscall.Errno(code)
}
