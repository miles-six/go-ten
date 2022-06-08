package network

import (
	"fmt"

	"github.com/obscuronet/obscuro-playground/go/ethclient/erc20contractlib"
	"github.com/obscuronet/obscuro-playground/go/ethclient/mgmtcontractlib"
	"github.com/obscuronet/obscuro-playground/go/obscuronode/wallet"
	"github.com/obscuronet/obscuro-playground/integration/gethnetwork"

	"github.com/obscuronet/obscuro-playground/go/obscuronode/obscuroclient"

	"github.com/obscuronet/obscuro-playground/go/ethclient"

	"github.com/obscuronet/obscuro-playground/integration/simulation/params"

	"github.com/obscuronet/obscuro-playground/integration/simulation/stats"
)

// creates Obscuro nodes with their own enclave servers that communicate with peers via sockets, wires them up, and populates the network objects
type networkOfSocketNodes struct {
	obscuroClients   []obscuroclient.Client
	enclaveAddresses []string

	// geth
	gethNetwork  *gethnetwork.GethNetwork
	gethClients  []ethclient.EthClient
	wallets      []wallet.Wallet
	contracts    []string
	workerWallet wallet.Wallet
}

func NewNetworkOfSocketNodes(wallets []wallet.Wallet, workerWallet wallet.Wallet, contracts []string) Network {
	return &networkOfSocketNodes{
		wallets:      wallets,
		contracts:    contracts,
		workerWallet: workerWallet,
	}
}

func (n *networkOfSocketNodes) Create(params *params.SimParams, stats *stats.Stats) ([]ethclient.EthClient, []obscuroclient.Client, []string, error) {
	// kickoff the network with the prefunded wallet addresses
	params.MgmtContractAddr, params.StableTokenContractAddr, n.gethClients, n.gethNetwork = SetUpGethNetwork(
		n.wallets,
		n.workerWallet,
		params.StartPort,
		params.NumberOfNodes,
		int(params.AvgBlockDuration.Seconds()),
	)

	params.MgmtContractLib = mgmtcontractlib.NewMgmtContractLib(params.MgmtContractAddr)
	params.ERC20ContractLib = erc20contractlib.NewERC20ContractLib(params.MgmtContractAddr, params.StableTokenContractAddr)

	// Start the enclaves
	startRemoteEnclaveServers(0, params, stats)

	n.enclaveAddresses = make([]string, params.NumberOfNodes)
	for i := 0; i < params.NumberOfNodes; i++ {
		n.enclaveAddresses[i] = fmt.Sprintf("%s:%d", Localhost, params.StartPort+DefaultEnclaveOffset+i)
	}

	obscuroClients, nodeP2pAddrs := startStandaloneObscuroNodes(params, stats, n.gethClients, n.enclaveAddresses)
	n.obscuroClients = obscuroClients

	return n.gethClients, n.obscuroClients, nodeP2pAddrs, nil
}

func (n *networkOfSocketNodes) TearDown() {
	// First stop the obscuro nodes
	StopObscuroNodes(n.obscuroClients)
	StopGethNetwork(n.gethClients, n.gethNetwork)

	// stop the enclaves
	// todo
}
