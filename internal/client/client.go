package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Endpoint   string
	ApiToken   string
	Az         string
	OrgaId     string
	ProjectId  string
	HTTPClient *http.Client
}

type VPC struct {
	ID           string        `json:"id,omitempty"`
	Name         string        `json:"name"`
	Az           string        `json:"az"`
	StaticRoutes []StaticRoute `json:"static_routes"`
}

type StaticRoute struct {
	Policy     string `json:"policy"`
	CIDR       string `json:"cidr"`
	NextHopIP  string `json:"next_hop_ip"`
	RouteTable string `json:"route_table"`
}

type VPCRequest struct {
	Name string `json:"name"`
	// StaticRoutes: nil = existing routes kept by the proxy, [] = all removed
	StaticRoutes *[]StaticRoute `json:"static_routes,omitempty"`
}

type Subnet struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	VPCID        string `json:"vpc_id"`
	IPv4         string `json:"ipv4"`
	IPv6         string `json:"ipv6,omitempty"`
	Private      bool   `json:"private"`
	Protocol     string `json:"protocol"`
	DNSV4        string `json:"dns_v4,omitempty"`
	DNSV6        string `json:"dns_v6,omitempty"`
	NatGateway   bool   `json:"nat_gateway"`
	NatGatewayIp string `json:"nat_gateway_ip,omitempty"`
}

type SubnetRequest struct {
	Name       string `json:"name"`
	VPCID      string `json:"vpc_id"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6,omitempty"`
	Private    bool   `json:"private"`
	Protocol   string `json:"protocol"`
	DNSV4      string `json:"dns_v4,omitempty"`
	DNSV6      string `json:"dns_v6,omitempty"`
	NatGateway bool   `json:"nat_gateway"`
}

type Disk struct {
	ID           string      `json:"id,omitempty"`
	Name         string      `json:"name"`
	Az           string      `json:"az"`
	Size         int         `json:"size"`
	StorageClass string      `json:"storage_class"`
	Source       *DiskSource `json:"source,omitempty"`
	Labels       []string    `json:"labels"`
}

type DiskSource struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Clone    string `json:"clone,omitempty"`
	Snapshot string `json:"snapshot,omitempty"`
}

type DiskRequest struct {
	Name         string      `json:"name"`
	Size         int         `json:"size"`
	StorageClass string      `json:"storage_class"`
	Source       *DiskSource `json:"source,omitempty"`
	Labels       []string    `json:"labels"`
}

type Eip struct {
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name"`
	Az         string    `json:"az"`
	SubnetId   string    `json:"subnet_id"`
	InternalIP string    `json:"internal_ip"`
	ExternalIP string    `json:"external_ip"`
	SNAT       []string  `json:"snat,omitempty"`
	DNAT       []EipDNAT `json:"dnat,omitempty"`
	Nat        string    `json:"nat,omitempty"`
}

type EipDNAT struct {
	ExternalPort string `json:"external_port"`
	InternalIP   string `json:"internal_ip"`
	InternalPort string `json:"internal_port"`
	Protocol     string `json:"protocol"`
}

type EipRequest struct {
	Name       string    `json:"name"`
	SubnetId   string    `json:"subnet_id"`
	InternalIP string    `json:"internal_ip,omitempty"`
	SNAT       []string  `json:"snat,omitempty"`
	DNAT       []EipDNAT `json:"dnat,omitempty"`
}

type LoadBalancer struct {
	ID        string             `json:"id,omitempty"`
	Name      string             `json:"name"`
	Az        string             `json:"az"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

type LBSelector struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LoadBalancerPort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

type LoadBalancerRequest struct {
	Name      string             `json:"name"`
	VIP       string             `json:"vip"`
	Selectors []LBSelector       `json:"selectors,omitempty"`
	Endpoints []string           `json:"endpoints,omitempty"`
	Ports     []LoadBalancerPort `json:"ports"`
}

type SSHKey struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type SSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type Instance struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name"`
	RunStrategy    string            `json:"run_strategy"`
	VMType         string            `json:"vm_type"`
	CPU            int               `json:"cpu"`
	Memory         int               `json:"memory"`
	Network        []InstanceNetwork `json:"network"`
	Disks          []InstanceDisk    `json:"disks"`
	CloudInit      InstanceCloudInit `json:"cloud_init"`
	SSHKeys        []string          `json:"ssh_keys"`
	Labels         []string          `json:"labels"`
	Ready          bool              `json:"ready"`
	IPAddress      string            `json:"ip_address"`
	Architecture   string            `json:"architecture"`
	ContainerDisks []string          `json:"container_disks"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

// InstanceAdvanced holds the advanced options (TPM / EFI). A nil field = inherited from the vm_type.
type InstanceAdvanced struct {
	TPM InstanceTPM `json:"tpm"`
	EFI InstanceEFI `json:"efi"`
}

type InstanceTPM struct {
	Enabled    *bool `json:"enabled,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

type InstanceEFI struct {
	Enabled    *bool `json:"enabled,omitempty"`
	SecureBoot *bool `json:"secure_boot,omitempty"`
	Persistent *bool `json:"persistent,omitempty"`
}

type InstanceNetwork struct {
	Order      int    `json:"order"`
	SubnetID   string `json:"subnet_id"`
	Model      string `json:"model"`
	Enabled    *bool  `json:"enabled,omitempty"`
	IPv4       string `json:"ipv4,omitempty"`
	IPv6       string `json:"ipv6,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}

type InstanceDisk struct {
	Order  int    `json:"order"`
	Bus    string `json:"bus"`
	DiskID string `json:"disk_id,omitempty"`
	Cdrom  bool   `json:"cdrom"`
}

type InstanceCloudInit struct {
	Custom bool   `json:"custom"`
	Bus    string `json:"bus"`
	Config string `json:"config,omitempty"`
}

type InstanceRequest struct {
	Name        string            `json:"name"`
	RunStrategy string            `json:"run_strategy"`
	VMType      string            `json:"vm_type"`
	CPU         int               `json:"cpu"`
	Memory      int               `json:"memory"`
	Network     []InstanceNetwork `json:"network"`
	Disks       []InstanceDisk    `json:"disks"`
	CloudInit   InstanceCloudInit `json:"cloud_init"`
	SSHKeys     []string          `json:"ssh_keys"`
	Labels      []string          `json:"labels"`
	// ContainerDisks: nil = recommended set (create) / existing mounts kept (update)
	ContainerDisks *[]string         `json:"container_disks,omitempty"`
	Advanced       *InstanceAdvanced `json:"advanced,omitempty"`
}

type SecurityGroup struct {
	ID          string        `json:"id,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	SubnetIDs   []string      `json:"subnetIds"`
}

type SecurityGroupRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      LabelSelector `json:"target"`
	Ingress     []IngressRule `json:"ingress"`
	Egress      []EgressRule  `json:"egress"`
	SubnetIDs   []string      `json:"subnetIds"`
}

type LabelSelector struct {
	MatchLabels      []string          `json:"matchLabels"`
	MatchExpressions []MatchExpression `json:"matchExpressions"`
}

type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type IngressRule struct {
	Ports    []SgPort `json:"ports"`
	From     []Peer   `json:"from"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

type EgressRule struct {
	Ports    []SgPort `json:"ports"`
	To       []Peer   `json:"to"`
	AllowAll bool     `json:"allowAll"`
	DenyAll  bool     `json:"denyAll"`
}

type SgPort struct {
	Port     int32  `json:"port"`
	EndPort  int32  `json:"endPort,omitempty"`
	Protocol string `json:"protocol"`
}

type Peer struct {
	PodSelector *LabelSelector `json:"podSelector,omitempty"`
	IPBlock     *IPBlock       `json:"IPBlock,omitempty"`
}

type IPBlock struct {
	CIDR   string   `json:"CIDR"`
	Except []string `json:"Except"`
}

func NewClient(endpoint, apiToken, az, orgaId, projectId string) *Client {
	return &Client{
		Endpoint:  endpoint,
		ApiToken:  apiToken,
		Az:        az,
		OrgaId:    orgaId,
		ProjectId: projectId,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) GetVPC(ctx context.Context, orgaId, az, projectId, id string) (*VPC, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/vpcs/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get VPC: %s - %s", resp.Status, string(body))
	}

	var vpc VPC
	if err := json.NewDecoder(resp.Body).Decode(&vpc); err != nil {
		return nil, err
	}

	return &vpc, nil
}

func (c *Client) CreateVPC(ctx context.Context, orgaId, az, projectId string, vpcReq VPCRequest) (*VPC, error) {
	data, err := json.Marshal(vpcReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/vpcs", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create VPC: %s - %s", resp.Status, string(body))
	}

	var vpc VPC
	if err := json.NewDecoder(resp.Body).Decode(&vpc); err != nil {
		return nil, err
	}

	return &vpc, nil
}

func (c *Client) UpdateVPC(ctx context.Context, orgaId, az, projectId, id string, vpcReq VPCRequest) (*VPC, error) {
	data, err := json.Marshal(vpcReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/vpcs/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update VPC: %s - %s", resp.Status, string(body))
	}

	var vpc VPC
	if err := json.NewDecoder(resp.Body).Decode(&vpc); err != nil {
		return nil, err
	}

	return &vpc, nil
}

func (c *Client) DeleteVPC(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/vpcs/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete VPC: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetSubnet(ctx context.Context, orgaId, az, projectId, id string) (*Subnet, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/subnets/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get Subnet: %s - %s", resp.Status, string(body))
	}

	var subnet Subnet
	if err := json.NewDecoder(resp.Body).Decode(&subnet); err != nil {
		return nil, err
	}

	return &subnet, nil
}

func (c *Client) CreateSubnet(ctx context.Context, orgaId, az, projectId string, subnetReq SubnetRequest) (*Subnet, error) {
	data, err := json.Marshal(subnetReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/subnets", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create Subnet: %s - %s", resp.Status, string(body))
	}

	var subnet Subnet
	if err := json.NewDecoder(resp.Body).Decode(&subnet); err != nil {
		return nil, err
	}

	return &subnet, nil
}

func (c *Client) UpdateSubnet(ctx context.Context, orgaId, az, projectId, id string, subnetReq SubnetRequest) (*Subnet, error) {
	data, err := json.Marshal(subnetReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/subnets/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update Subnet: %s - %s", resp.Status, string(body))
	}

	var subnet Subnet
	if err := json.NewDecoder(resp.Body).Decode(&subnet); err != nil {
		return nil, err
	}

	return &subnet, nil
}

func (c *Client) DeleteSubnet(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/subnets/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete Subnet: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetDisk(ctx context.Context, orgaId, az, projectId, id string) (*Disk, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/disks/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get Disk: %s - %s", resp.Status, string(body))
	}

	var disk Disk
	if err := json.NewDecoder(resp.Body).Decode(&disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

func (c *Client) CreateDisk(ctx context.Context, orgaId, az, projectId string, diskReq DiskRequest) (*Disk, error) {
	data, err := json.Marshal(diskReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/disks", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create Disk: %s - %s", resp.Status, string(body))
	}

	var disk Disk
	if err := json.NewDecoder(resp.Body).Decode(&disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

func (c *Client) UpdateDisk(ctx context.Context, orgaId, az, projectId, id string, diskReq DiskRequest) (*Disk, error) {
	data, err := json.Marshal(diskReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/disks/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update Disk: %s - %s", resp.Status, string(body))
	}

	var disk Disk
	if err := json.NewDecoder(resp.Body).Decode(&disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

func (c *Client) DeleteDisk(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/disks/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete Disk: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) ListStorageClasses(ctx context.Context, orgaId, az, projectId string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/storage-classes", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list StorageClasses: %s - %s", resp.Status, string(body))
	}

	var storageClasses []string
	if err := json.NewDecoder(resp.Body).Decode(&storageClasses); err != nil {
		return nil, err
	}

	return storageClasses, nil
}

func (c *Client) ListVMTypes(ctx context.Context, orgaId, az, projectId string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s/vm-types", c.Endpoint, orgaId, az, projectId), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, string(body))
	}

	var vmTypes []string
	if err := json.NewDecoder(res.Body).Decode(&vmTypes); err != nil {
		return nil, err
	}

	return vmTypes, nil
}

func (c *Client) GetEip(ctx context.Context, orgaId, az, projectId, id string) (*Eip, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/eips/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get EIP: %s - %s", resp.Status, string(body))
	}

	var eip Eip
	if err := json.NewDecoder(resp.Body).Decode(&eip); err != nil {
		return nil, err
	}

	return &eip, nil
}

func (c *Client) CreateEip(ctx context.Context, orgaId, az, projectId string, eipReq EipRequest) (*Eip, error) {
	data, err := json.Marshal(eipReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/eips", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create EIP: %s - %s", resp.Status, string(body))
	}

	var eip Eip
	if err := json.NewDecoder(resp.Body).Decode(&eip); err != nil {
		return nil, err
	}

	return &eip, nil
}

func (c *Client) UpdateEip(ctx context.Context, orgaId, az, projectId, id string, eipReq EipRequest) (*Eip, error) {
	data, err := json.Marshal(eipReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/eips/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update EIP: %s - %s", resp.Status, string(body))
	}

	var eip Eip
	if err := json.NewDecoder(resp.Body).Decode(&eip); err != nil {
		return nil, err
	}

	return &eip, nil
}

func (c *Client) DeleteEip(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/eips/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete EIP: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetLoadBalancer(ctx context.Context, orgaId, az, projectId, id string) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/load-balancers/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get Load Balancer: %s - %s", resp.Status, string(body))
	}

	var lb LoadBalancer
	if err := json.NewDecoder(resp.Body).Decode(&lb); err != nil {
		return nil, err
	}

	return &lb, nil
}

func (c *Client) CreateLoadBalancer(ctx context.Context, orgaId, az, projectId string, lbReq LoadBalancerRequest) (*LoadBalancer, error) {
	data, err := json.Marshal(lbReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/load-balancers", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create Load Balancer: %s - %s", resp.Status, string(body))
	}

	var lb LoadBalancer
	if err := json.NewDecoder(resp.Body).Decode(&lb); err != nil {
		return nil, err
	}

	return &lb, nil
}

func (c *Client) UpdateLoadBalancer(ctx context.Context, orgaId, az, projectId, id string, lbReq LoadBalancerRequest) (*LoadBalancer, error) {
	data, err := json.Marshal(lbReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/load-balancers/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update Load Balancer: %s - %s", resp.Status, string(body))
	}

	var lb LoadBalancer
	if err := json.NewDecoder(resp.Body).Decode(&lb); err != nil {
		return nil, err
	}

	return &lb, nil
}

func (c *Client) DeleteLoadBalancer(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/load-balancers/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete Load Balancer: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetSSHKey(ctx context.Context, orgaId, az, projectId, id string) (*SSHKey, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/ssh-keys/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get SSH Key: %s - %s", resp.Status, string(body))
	}

	var sshKey SSHKey
	if err := json.NewDecoder(resp.Body).Decode(&sshKey); err != nil {
		return nil, err
	}

	return &sshKey, nil
}

func (c *Client) CreateSSHKey(ctx context.Context, orgaId, az, projectId string, sshKeyReq SSHKeyRequest) (*SSHKey, error) {
	data, err := json.Marshal(sshKeyReq)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/%s/%s/ssh-keys", c.Endpoint, orgaId, az, projectId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create SSH Key: %s - %s", resp.Status, string(body))
	}

	var sshKey SSHKey
	if err := json.NewDecoder(resp.Body).Decode(&sshKey); err != nil {
		return nil, err
	}

	return &sshKey, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, orgaId, az, projectId, id string) error {
	url := fmt.Sprintf("%s/%s/%s/%s/ssh-keys/%s", c.Endpoint, orgaId, az, projectId, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete SSH Key: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) ListInstances(ctx context.Context, orgaId, az, projectId string) ([]Instance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s/instances", c.Endpoint, orgaId, az, projectId), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var instances []Instance
	err = json.NewDecoder(resp.Body).Decode(&instances)
	return instances, err
}

func (c *Client) CreateInstance(ctx context.Context, orgaId, az, projectId string, instanceReq InstanceRequest) (*Instance, error) {
	data, err := json.Marshal(instanceReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/%s/%s/%s/instances", c.Endpoint, orgaId, az, projectId), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
	}

	var instance Instance
	err = json.NewDecoder(resp.Body).Decode(&instance)
	return &instance, err
}

func (c *Client) GetInstance(ctx context.Context, orgaId, az, projectId, id string) (*Instance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s/instances/%s", c.Endpoint, orgaId, az, projectId, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var instance Instance
	err = json.NewDecoder(resp.Body).Decode(&instance)
	return &instance, err
}

func (c *Client) UpdateInstance(ctx context.Context, orgaId, az, projectId, id string, instanceReq InstanceRequest) (*Instance, error) {
	data, err := json.Marshal(instanceReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/%s/%s/%s/instances/%s", c.Endpoint, orgaId, az, projectId, id), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
	}

	var instance Instance
	err = json.NewDecoder(resp.Body).Decode(&instance)
	return &instance, err
}

func (c *Client) DeleteInstance(ctx context.Context, orgaId, az, projectId, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/%s/%s/%s/instances/%s", c.Endpoint, orgaId, az, projectId, id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) ListSecurityGroups(ctx context.Context, orgaId, az, projectId string) ([]SecurityGroup, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s/security-groups", c.Endpoint, orgaId, az, projectId), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var securityGroups []SecurityGroup
	err = json.NewDecoder(resp.Body).Decode(&securityGroups)
	return securityGroups, err
}

func (c *Client) GetSecurityGroup(ctx context.Context, orgaId, az, projectId, id string) (*SecurityGroup, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s/security-groups/%s", c.Endpoint, orgaId, az, projectId, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var securityGroup SecurityGroup
	err = json.NewDecoder(resp.Body).Decode(&securityGroup)
	return &securityGroup, err
}

func (c *Client) CreateSecurityGroup(ctx context.Context, orgaId, az, projectId string, sgReq SecurityGroupRequest) (*SecurityGroup, error) {
	data, err := json.Marshal(sgReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/%s/%s/%s/security-groups", c.Endpoint, orgaId, az, projectId), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
	}

	var securityGroup SecurityGroup
	err = json.NewDecoder(resp.Body).Decode(&securityGroup)
	return &securityGroup, err
}

func (c *Client) UpdateSecurityGroup(ctx context.Context, orgaId, az, projectId, id string, sgReq SecurityGroupRequest) (*SecurityGroup, error) {
	data, err := json.Marshal(sgReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/%s/%s/%s/security-groups/%s", c.Endpoint, orgaId, az, projectId, id), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
	}

	var securityGroup SecurityGroup
	err = json.NewDecoder(resp.Body).Decode(&securityGroup)
	return &securityGroup, err
}

func (c *Client) DeleteSecurityGroup(ctx context.Context, orgaId, az, projectId, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/%s/%s/%s/security-groups/%s", c.Endpoint, orgaId, az, projectId, id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("api-token %s", c.ApiToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}

	return nil
}
