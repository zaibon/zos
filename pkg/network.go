package pkg

import (
	"net"

	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/versioned"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module network -version 0.0.1 -name network -package stubs github.com/threefoldtech/zos/pkg+Networker stubs/network_stub.go

// Member holds information about a join operation
type Member struct {
	Namespace string
	IPv6      net.IP
	IPv4      net.IP
}

//Networker is the interface for the network module
type Networker interface {
	// Create a new network resource
	CreateNR(Network) (string, error)
	// Delete a network resource
	DeleteNR(Network) error

	// Join a network (with network id) will create a new isolated namespace
	// that is hooked to the network bridge with a veth pair, and assign it a
	// new IP from the network resource range. The method return the new namespace
	// name.
	// The member name specifies the name of the member, and must be unique
	// The NetID is the network id to join
	Join(networkdID NetID, containerID string, addrs []string) (join Member, err error)
	// Leave delete a container nameapce created by Join
	Leave(networkdID NetID, containerID string) (err error)

	// ZDBPrepare creates a network namespace with a macvlan interface into it
	// to allow the 0-db container to be publicly accessible
	// it retusn the name of the network namespace created
	// hw is an optional hardware address that will be set on the new interface
	ZDBPrepare(hw net.HardwareAddr) (string, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	Addrs(iface string, netns string) ([]net.IP, error)
}

// Network represent the description if a user private network
type Network struct {
	Name string `json:"name"`
	//unique id inside the reservation is an autoincrement (USE AS NET_ID)
	NetID NetID `json:"net_id"`
	// IP range of the network, must be an IPv4 /16
	IPRange types.IPNet `json:"ip_range"`

	NetResources []NetResource `json:"net_resources"`

	AccessPoints []AccessPoint `json:"access_points,omitempty"`
}

// NetResource is the description of a part of a network local to a specific node
type NetResource struct {
	NodeID string `json:"node_id"`
	// Public endpoints
	PubEndpoints []net.IP `json:"pub_endpoints"`
	// IPV4 subnet from network IPRange
	Subnet types.IPNet `json:"subnet"`

	WGPrivateKey string `json:"wg_private_key"`
	WGPublicKey  string `json:"wg_public_key"`
	WGListenPort uint16 `json:"wg_listen_port"`

	Peers []Peer `json:"peers"`
}

// Peer is the description of a peer of a NetResource
type Peer struct {
	// IPV4 subnet of the network resource of the peer
	Subnet types.IPNet `json:"subnet"`

	WGPublicKey string        `json:"wg_public_key"`
	AllowedIPs  []types.IPNet `json:"allowed_ips"`
	Endpoint    string        `json:"endpoint,omitempty"`
}

// NetID is a type defining the ID of a network
type NetID string

var (
	// NetworkSchemaV1 network object schema version 1.0.0
	NetworkSchemaV1 = versioned.MustParse("1.0.0")
	// NetworkSchemaLatestVersion network object latest version
	NetworkSchemaLatestVersion = NetworkSchemaV1
)

// AccessPoint info for a network, defining a node which will act as the AP, and
// the subnet which will be routed through it
type AccessPoint struct {
	// NodeID of the access point in the network
	NodeID string `json:"node_id"`
	// Subnet to be routed through this access point
	Subnet      types.IPNet `json:"subnet"`
	WGPublicKey string      `json:"wg_public_key"`
}
