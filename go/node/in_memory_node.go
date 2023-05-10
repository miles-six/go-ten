package node

import (
	"fmt"

	"github.com/sanity-io/litter"

	enclavecontainer "github.com/obscuronet/go-obscuro/go/enclave/container"
	hostcontainer "github.com/obscuronet/go-obscuro/go/host/container"
)

type InMemNode struct {
	cfg     *Config
	enclave *enclavecontainer.EnclaveContainer
	host    *hostcontainer.HostContainer
}

func NewInMemNode(cfg *Config) *InMemNode {
	return &InMemNode{
		cfg: cfg,
	}
}

func (d *InMemNode) Start() error {
	// TODO this should probably be removed in the future
	fmt.Printf("Starting Node %s with config: \n%s\n\n", d.cfg.nodeName, litter.Sdump(*d.cfg))

	err := d.startEnclave()
	if err != nil {
		return err
	}

	err = d.startHost()
	if err != nil {
		return err
	}

	return nil
}

func (d *InMemNode) Stop() error {
	fmt.Println("Stopping existing host and enclave")
	if err := d.host.Stop(); err != nil {
		return err
	}

	return d.enclave.Stop()
}

func (d *InMemNode) Upgrade(networkCfg *NetworkConfig) error {
	// TODO this should probably be removed in the future
	fmt.Printf("Upgrading node %s with config: %+v\n", d.cfg.nodeName, d.cfg)

	err := d.Stop()
	if err != nil {
		return err
	}

	// update network configs
	d.cfg.UpdateNodeConfig(
		WithManagementContractAddress(networkCfg.ManagementContractAddress),
		WithManagementContractAddress(networkCfg.MessageBusAddress),
		WithL1Start(networkCfg.L1StartHash),
	)

	fmt.Println("Starting upgraded host and enclave")
	err = d.startEnclave()
	if err != nil {
		return err
	}

	err = d.startHost()
	if err != nil {
		return err
	}

	return nil
}

func (d *InMemNode) startHost() error {
	hostConfig := d.cfg.ToHostConfig()
	d.host = hostcontainer.NewHostContainerFromConfig(hostConfig)
	return d.host.Start()
}

func (d *InMemNode) startEnclave() error {
	enclaveCfg := d.cfg.ToEnclaveConfig()
	d.enclave = enclavecontainer.NewEnclaveContainerFromConfig(*enclaveCfg)
	return d.enclave.Start()
}