/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"sort"

	"github.com/urfave/cli"
)

// AddCommands adds all child commands to app
func AddCommands(app *cli.App, isFullNode bool) {
	baseCommands := []cli.Command{
		{
			Name:   "getbalance",
			Usage:  "get balance info",
			Flags:  rpcFlags(accountFlag, hashFlag, heightFlag),
			Action: rpcAction("seele", "getBalance"),
		},
		{
			Name:   "sendtx",
			Usage:  "send transaction to node",
			Flags:  rpcFlags(fromFlag, toFlag, amountFlag, priceFlag, gasLimitFlag, payloadFlag, nonceFlag),
			Action: rpcActionEx("seele", "addTx", makeTransaction, onTxAdded),
		},
		{
			Name:   "getnonce",
			Usage:  "get account nonce",
			Flags:  rpcFlags(accountFlag, hashFlag, heightFlag),
			Action: rpcAction("seele", "getAccountNonce"),
		},
		{
			Name:   "gettxcount",
			Usage:  "get account tx count",
			Flags:  rpcFlags(accountFlag, hashFlag, heightFlag),
			Action: rpcAction("seele", "getAccountTxCount"),
		},
		{
			Name:   "getblockheight",
			Usage:  "get block height",
			Flags:  rpcFlags(),
			Action: rpcAction("seele", "getBlockHeight"),
		},
		{
			Name:   "getblock",
			Usage:  "get block by height or hash",
			Flags:  rpcFlags(hashFlag, heightFlag, fulltxFlag),
			Action: rpcAction("seele", "getBlock"),
		},
		{
			Name:   "gettxpoolcontent",
			Usage:  "get transaction pool contents",
			Flags:  rpcFlags(),
			Action: rpcAction("debug", "getTxPoolContent"),
		},
		{
			Name:   "gettxpoolcount",
			Usage:  "get transaction pool transaction count",
			Flags:  rpcFlags(),
			Action: rpcAction("debug", "getTxPoolTxCount"),
		},
		{
			Name:   "getblocktxcount",
			Usage:  "get block transaction count by block height or block hash",
			Flags:  rpcFlags(hashFlag, heightFlag),
			Action: rpcAction("txpool", "getBlockTransactionCount"),
		},
		{
			Name:   "gettxinblock",
			Usage:  "get transaction by block height or block hash with index of the transaction in the block",
			Flags:  rpcFlags(hashFlag, heightFlag, indexFlag),
			Action: rpcAction("txpool", "getTransactionByBlockIndex"),
		},
		{
			Name:   "gettxbyhash",
			Usage:  "get transaction by transaction hash",
			Flags:  rpcFlags(hashFlag),
			Action: rpcAction("txpool", "getTransactionByHash"),
		},
		{
			Name:   "getreceipt",
			Usage:  "get receipt by transaction hash",
			Flags:  rpcFlags(hashFlag, abiFileFlag),
			Action: rpcAction("txpool", "getReceiptByTxHash"),
		},
		{
			Name:   "getpendingtxs",
			Usage:  "get pending transactions",
			Flags:  rpcFlags(),
			Action: rpcAction("debug", "getPendingTransactions"),
		},
		{
			Name:  "getshardnum",
			Usage: "get account shard number",
			Flags: []cli.Flag{
				accountFlag,
				privateKeyFlag,
			},
			Action: GetAccountShardNumAction,
		},
		{
			Name:  "savekey",
			Usage: "save private key to a keystore file",
			Flags: []cli.Flag{
				privateKeyFlag,
				fileNameFlag,
			},
			Action: SaveKeyAction,
		},
		{
			Name:  "sign",
			Usage: "generate a signed transaction and print it out",
			Flags: []cli.Flag{
				addressFlag,
				privateKeyFlag,
				toFlag,
				amountFlag,
				priceFlag,
				gasLimitFlag,
				payloadFlag,
				nonceFlag,
			},
			Action: SignTxAction,
		},
		{
			Name:  "key",
			Usage: "generate key with or without shard number",
			Flags: []cli.Flag{
				shardFlag,
			},
			Action: GenerateKeyAction,
		},
		{
			Name:  "payload",
			Usage: "generate the payload according to the abi file and method name and args",
			Flags: []cli.Flag{
				abiFileFlag, methodNameFlag, argsFlag,
			},
			Action: GeneratePayloadAction,
		},
		{
			Name:  "deckeyfile",
			Usage: "Decrypt key file",
			Flags: []cli.Flag{
				fileNameFlag,
			},
			Action: DecryptKeyFileAction,
		},
	}

	htlcCommands := cli.Command{
		Name:  "htlc",
		Usage: "Hash time lock contract commands",
		Subcommands: []cli.Command{
			{
				Name:   "create",
				Usage:  "create HTLC",
				Flags:  rpcFlags(fromFlag, toFlag, amountFlag, priceFlag, gasLimitFlag, nonceFlag, hashFlag, timeLockFlag),
				Action: rpcActionSystemContract("htlc", "create", handleCallResult),
			},
			{
				Name:   "withdraw",
				Usage:  "withdraw from HTLC",
				Flags:  rpcFlags(fromFlag, priceFlag, gasLimitFlag, nonceFlag, hashFlag, preimageFlag),
				Action: rpcActionSystemContract("htlc", "withdraw", handleCallResult),
			},
			{
				Name:   "refund",
				Usage:  "refund from HTLC",
				Flags:  rpcFlags(fromFlag, priceFlag, gasLimitFlag, nonceFlag, hashFlag),
				Action: rpcActionSystemContract("htlc", "refund", handleCallResult),
			},
			{
				Name:   "get",
				Usage:  "get HTLC information",
				Flags:  rpcFlags(fromFlag, hashFlag),
				Action: rpcActionSystemContract("htlc", "get", handleCallResult),
			},
			{
				Name:  "decode",
				Usage: "decode HTLC contract information",
				Flags: []cli.Flag{
					payloadFlag,
				},
				Action: decodeHTLC,
			},
			{
				Name:   "key",
				Usage:  "generate preimage key and key hash",
				Action: generateHTLCKey,
			},
			{
				Name:  "time",
				Usage: "generate unix timestamp",
				Flags: []cli.Flag{
					timeLockFlag,
				},
				Action: generateHTLCTime,
			},
		},
	}

	domainCommands := cli.Command{
		Name:  "domain",
		Usage: "system domain name commands",
		Subcommands: []cli.Command{
			{
				Name:   "register",
				Usage:  "register a domain name",
				Flags:  rpcFlags(fromFlag, priceFlag, gasLimitFlag, nameFlag, nonceFlag),
				Action: rpcActionSystemContract("domain", "create", handleCallResult),
			},
			{
				Name:   "owner",
				Usage:  "get the domain name owner",
				Flags:  rpcFlags(fromFlag, priceFlag, gasLimitFlag, nameFlag, nonceFlag),
				Action: rpcActionSystemContract("domain", "getOwner", handleCallResult),
			},
		},
	}

	subChainCommands := cli.Command{
		Name:  "subchain",
		Usage: "system sub chain commands",
		Subcommands: []cli.Command{
			{
				Name:   "getblockcreator",
				Usage:  "get block creator",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getBlockCreator"),
			},
			{
				Name:   "getbalancetreeroot",
				Usage:  "get the root of balance tree",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getBalanceTreeRoot"),
			},
			{
				Name:   "gettxtreeroot",
				Usage:  "get the root of tx tree",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getTxTreeRoot"),
			},
			{
				Name:   "getblocksignature",
				Usage:  "get block signature",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getBlockSignature"),
			},
			{
				Name:   "getblocksteminfo",
				Usage:  "get block data (creator, height, txRoot, stateRoot)",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getBlockInfoForStem"),
			},
			{
				Name:   "gettxmerkleinfo",
				Usage:  "get merkle proof and index given tx hash",
				Flags:  rpcFlags(hashFlag),
				Action: rpcAction("subchain", "getTxMerkleInfo"),
			},
			{
				Name:   "getbalancemerkleinfo",
				Usage:  "get merkle proof and index given account and block height",
				Flags:  rpcFlags(accountFlag, heightFlag),
				Action: rpcAction("subchain", "getBalanceMerkleInfo"),
			},
			{
				Name:   "getrecenttxtreeroot",
				Usage:  "get the root of recent tx tree",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getRecentTxTreeRoot"),
			},
			{
				Name:   "getrecenttxmerkleinfo",
				Usage:  "get the merkle proof and index of recent tx tree given account and block height",
				Flags:  rpcFlags(accountFlag, heightFlag),
				Action: rpcAction("subchain", "getRecentTxMerkleInfo"),
			},
			{
				Name:   "getaccounttx",
				Usage:  "get tx data between two block heights given an account",
				Flags:  rpcFlags(accountFlag, startHeightFlag, endHeightFlag),
				Action: rpcAction("subchain", "getAccountTx"),
			},
			{
				Name:   "getupdatedaccountinfo",
				Usage:  "get the updated accounts on last relay interval",
				Flags:  rpcFlags(heightFlag),
				Action: rpcAction("subchain", "getUpdatedAccountInfo"),
			},
			{
				Name:   "sendtx",
				Usage:  "send subchain transaction to node",
				Flags:  rpcFlags(fromFlag, toFlag, amountFlag, priceFlag, gasLimitFlag, payloadFlag, nonceFlag, heightFlag),
				Action: rpcActionEx("seele", "addTx", makeSubTransaction, onTxAdded),
			},
		},
	}

	p2pCommands := cli.Command{
		Name:  "p2p",
		Usage: "p2p commands",
		Subcommands: []cli.Command{
			{
				Name:   "peers",
				Usage:  "get p2p peer connections",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getPeerCount"),
			},
			{
				Name:   "peersinfo",
				Usage:  "get p2p peers information",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getPeersInfo"),
			},
			{
				Name:   "netversion",
				Usage:  "get current net version",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getNetVersion"),
			},
			{
				Name:   "networkid",
				Usage:  "get current network id",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getNetworkID"),
			},
			{
				Name:   "protocolversion",
				Usage:  "get seele protocol version",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getProtocolVersion"),
			},
		},
	}

	minerCommands := cli.Command{
		Name:  "miner",
		Usage: "miner commands",
		Subcommands: []cli.Command{
			{
				Name:   "start",
				Usage:  "start miner",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "start"),
			},
			{
				Name:   "stop",
				Usage:  "stop miner",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "stop"),
			},
			{
				Name:   "setthreads",
				Usage:  "set miner thread number",
				Flags:  rpcFlags(threadsFlag),
				Action: rpcAction("miner", "setThreads"),
			},
			{
				Name:   "setcoinbase",
				Usage:  "set miner coinbase",
				Flags:  rpcFlags(coinbaseFlag),
				Action: rpcAction("miner", "setCoinbase"),
			},
			{
				Name:   "getcoinbase",
				Usage:  "get miner coinbase",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "getCoinbase"),
			},
			{
				Name:   "status",
				Usage:  "get miner status",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "status"),
			},
			{
				Name:   "hashrate",
				Usage:  "get hashrate",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "getHashrate"),
			},
			{
				Name:   "threads",
				Usage:  "get thread number",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "getThreads"),
			},
		},
	}

	// add full node support api
	if isFullNode {
		baseCommands = append(baseCommands, []cli.Command{
			{
				Name:   "getinfo",
				Usage:  "get node info",
				Flags:  rpcFlags(),
				Action: rpcAction("seele", "getInfo"),
			},
			{
				Name:   "height",
				Usage:  "get current block height",
				Flags:  rpcFlags(),
				Action: rpcAction("seele", "getHeight"),
			},
			{
				Name:   "getdebts",
				Usage:  "get pending debts",
				Flags:  rpcFlags(),
				Action: rpcAction("debug", "getPendingDebts"),
			},
			{
				Name:   "dumpheap",
				Usage:  "dump heap for profiling, return the file path",
				Flags:  rpcFlags(dumpFileFlag, gcBeforeDumpFlag),
				Action: rpcAction("debug", "dumpHeap"),
			},
			{
				Name:   "call",
				Usage:  "call contract",
				Flags:  rpcFlags(toFlag, payloadFlag, heightFlag),
				Action: rpcAction("seele", "call"),
			},
			{
				Name:   "getlogs",
				Usage:  "get logs",
				Flags:  rpcFlags(heightFlag, contractFlag, abiFileFlag, eventNameFlag),
				Action: rpcAction("seele", "getLogs"),
			},
			{
				Name:   "getdebtbyhash",
				Usage:  "get debt by debt hash",
				Flags:  rpcFlags(hashFlag),
				Action: rpcAction("txpool", "getDebtByHash"),
			},
		}...)

		baseCommands = append(baseCommands,
			htlcCommands,
			domainCommands,
			subChainCommands,
			minerCommands)
	}

	baseCommands = append(baseCommands, p2pCommands)

	app.Commands = baseCommands

	// sort commands and flags by name
	sortCommands(app.Commands)
}

func sortCommands(commands []cli.Command) {
	sort.Sort(cli.CommandsByName(commands))

	for _, command := range commands {
		if len(command.Subcommands) > 0 {
			sortCommands(command.Subcommands)
		}
	}
}
