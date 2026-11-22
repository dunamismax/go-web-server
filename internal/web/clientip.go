package web

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ipResolver works out the client's address. By default that is the TCP
// peer. X-Forwarded-For is consulted only when the peer is a configured
// trusted proxy, because any client can send that header.
type ipResolver struct {
	trusted []netip.Prefix
}

func (res ipResolver) resolve(r *http.Request) string {
	peer := peerAddr(r.RemoteAddr)
	if !peer.IsValid() {
		return r.RemoteAddr
	}
	if !res.isTrusted(peer) {
		return peer.String()
	}

	// Walk the chain from the nearest hop outwards. Each trusted proxy
	// vouches for the entry before it; the first address that is not one of
	// ours is the client.
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	client := peer
	for i := len(hops) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		client = addr.Unmap()
		if !res.isTrusted(client) {
			break
		}
	}
	return client.String()
}

func (res ipResolver) isTrusted(addr netip.Addr) bool {
	for _, prefix := range res.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func peerAddr(remoteAddr string) netip.Addr {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}
