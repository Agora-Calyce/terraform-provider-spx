package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/test-orga/test-az/test-project/vpcs/test-id" {
			t.Errorf("Expected path /test-orga/test-az/test-project/vpcs/test-id, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "api-token test-token" {
			t.Errorf("Expected Authorization header 'api-token test-token', got %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VPC{
			ID:   "test-id",
			Name: "test-vpc",
			Az:   "test-az",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token", "test-az", "test-orga", "test-project")
	vpc, err := c.GetVPC(context.Background(), "test-orga", "test-az", "test-project", "test-id")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if vpc.ID != "test-id" {
		t.Errorf("Expected VPC ID 'test-id', got %s", vpc.ID)
	}
	if vpc.Name != "test-vpc" {
		t.Errorf("Expected VPC Name 'test-vpc', got %s", vpc.Name)
	}
}

func TestCreateVPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		var vpcReq VPCRequest
		if err := json.NewDecoder(r.Body).Decode(&vpcReq); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if vpcReq.Name != "new-vpc" {
			t.Errorf("Expected VPC name 'new-vpc', got %s", vpcReq.Name)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(VPC{
			ID:   "new-id",
			Name: "new-vpc",
			Az:   "test-az",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token", "test-az", "test-orga", "test-project")
	vpc, err := c.CreateVPC(context.Background(), "test-orga", "test-az", "test-project", VPCRequest{Name: "new-vpc"})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if vpc.ID != "new-id" {
		t.Errorf("Expected VPC ID 'new-id', got %s", vpc.ID)
	}
}

func TestGetVPC_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token", "test-az", "test-orga", "test-project")
	vpc, err := c.GetVPC(context.Background(), "test-orga", "test-az", "test-project", "non-existent")

	if err != nil {
		t.Fatalf("Expected no error for 404, got %v", err)
	}

	if vpc != nil {
		t.Errorf("Expected nil VPC for 404, got %v", vpc)
	}
}

func TestGetVPC_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token", "test-az", "test-orga", "test-project")
	vpc, err := c.GetVPC(context.Background(), "test-orga", "test-az", "test-project", "test-id")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedError := "failed to get VPC: 500 Internal Server Error - internal server error"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	if vpc != nil {
		t.Errorf("Expected nil VPC on error, got %v", vpc)
	}
}

func TestSubnet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Subnet{ID: "subnet-1", Name: "test-subnet"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Subnet{ID: "subnet-1", Name: "test-subnet"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	subnet, err := c.GetSubnet(context.Background(), "orga", "az", "proj", "subnet-1")
	if err != nil || subnet.ID != "subnet-1" {
		t.Errorf("GetSubnet failed: %v", err)
	}

	// Test Create
	subnet, err = c.CreateSubnet(context.Background(), "orga", "az", "proj", SubnetRequest{Name: "test-subnet"})
	if err != nil || subnet.ID != "subnet-1" {
		t.Errorf("CreateSubnet failed: %v", err)
	}
}

func TestDisk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Disk{ID: "disk-1", Name: "test-disk"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Disk{ID: "disk-1", Name: "test-disk"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	disk, err := c.GetDisk(context.Background(), "orga", "az", "proj", "disk-1")
	if err != nil || disk.ID != "disk-1" {
		t.Errorf("GetDisk failed: %v", err)
	}

	// Test Create
	disk, err = c.CreateDisk(context.Background(), "orga", "az", "proj", DiskRequest{Name: "test-disk"})
	if err != nil || disk.ID != "disk-1" {
		t.Errorf("CreateDisk failed: %v", err)
	}
}

func TestEip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Eip{ID: "eip-1", Name: "test-eip"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Eip{ID: "eip-1", Name: "test-eip"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	eip, err := c.GetEip(context.Background(), "orga", "az", "proj", "eip-1")
	if err != nil || eip.ID != "eip-1" {
		t.Errorf("GetEip failed: %v", err)
	}

	// Test Create
	eip, err = c.CreateEip(context.Background(), "orga", "az", "proj", EipRequest{Name: "test-eip"})
	if err != nil || eip.ID != "eip-1" {
		t.Errorf("CreateEip failed: %v", err)
	}
}

func TestLoadBalancer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(LoadBalancer{ID: "lb-1", Name: "test-lb"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(LoadBalancer{ID: "lb-1", Name: "test-lb"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	lb, err := c.GetLoadBalancer(context.Background(), "orga", "az", "proj", "lb-1")
	if err != nil || lb.ID != "lb-1" {
		t.Errorf("GetLoadBalancer failed: %v", err)
	}

	// Test Create
	lb, err = c.CreateLoadBalancer(context.Background(), "orga", "az", "proj", LoadBalancerRequest{Name: "test-lb"})
	if err != nil || lb.ID != "lb-1" {
		t.Errorf("CreateLoadBalancer failed: %v", err)
	}
}

func TestSSHKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SSHKey{ID: "key-1", Name: "test-key"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(SSHKey{ID: "key-1", Name: "test-key"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	key, err := c.GetSSHKey(context.Background(), "orga", "az", "proj", "key-1")
	if err != nil || key.ID != "key-1" {
		t.Errorf("GetSSHKey failed: %v", err)
	}

	// Test Create
	key, err = c.CreateSSHKey(context.Background(), "orga", "az", "proj", SSHKeyRequest{Name: "test-key"})
	if err != nil || key.ID != "key-1" {
		t.Errorf("CreateSSHKey failed: %v", err)
	}
}

func TestInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Instance{ID: "inst-1", Name: "test-inst"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Instance{ID: "inst-1", Name: "test-inst"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	inst, err := c.GetInstance(context.Background(), "orga", "az", "proj", "inst-1")
	if err != nil || inst.ID != "inst-1" {
		t.Errorf("GetInstance failed: %v", err)
	}

	// Test Create
	inst, err = c.CreateInstance(context.Background(), "orga", "az", "proj", InstanceRequest{Name: "test-inst"})
	if err != nil || inst.ID != "inst-1" {
		t.Errorf("CreateInstance failed: %v", err)
	}
}

func TestSecurityGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path != "/orga/az/proj/security-groups/sg-1" {
			t.Errorf("unexpected GET path %s", r.URL.Path)
		}
		if r.Method == "POST" {
			if r.URL.Path != "/orga/az/proj/security-groups" {
				t.Errorf("unexpected POST path %s", r.URL.Path)
			}
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			port := body["ingress"].([]interface{})[0].(map[string]interface{})["ports"].([]interface{})[0].(map[string]interface{})
			if port["port"] != float64(443) {
				t.Errorf("port must be sent as a number, got %#v", port["port"])
			}
			if fmt.Sprint(body["subnetIds"]) != "[subnet-a]" {
				t.Errorf("unexpected subnetIds %#v", body["subnetIds"])
			}
		}
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SecurityGroup{ID: "sg-1", Name: "test-sg"})
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(SecurityGroup{ID: "sg-1", Name: "test-sg"})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "token", "az", "orga", "proj")

	// Test Get
	fw, err := c.GetSecurityGroup(context.Background(), "orga", "az", "proj", "sg-1")
	if err != nil || fw.ID != "sg-1" {
		t.Errorf("GetSecurityGroup failed: %v", err)
	}

	// Test Create
	fw, err = c.CreateSecurityGroup(context.Background(), "orga", "az", "proj", SecurityGroupRequest{
		Name:      "test-sg",
		Ingress:   []IngressRule{{Ports: []SgPort{{Port: 443, Protocol: "TCP"}}}},
		SubnetIDs: []string{"subnet-a"},
	})
	if err != nil || fw.ID != "sg-1" {
		t.Errorf("CreateSecurityGroup failed: %v", err)
	}
}
