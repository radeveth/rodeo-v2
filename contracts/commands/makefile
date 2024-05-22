CHAIN_ID := 42161
RPC_URL ?= https://arb1.arbitrum.io/rpc
RPC_URL ?= $(RPC_URL_42161)

test:
	forge test -vvv --no-match-contract "(Slow)"

build:
	forge build

clean:
	rm -rf broadcast out cache

fmt:
	find *.sol | xargs -n 1 -t forge fmt

node:
	anvil --chain-id 1337 # -b 2

call:
	cast call --rpc-url=$(RPC_URL) 0x656ca10e32447fdd9502d61c6ff696b622862ab5 "expiry() returns (uint256)"

tools:
	curl -L https://foundry.paradigm.xyz | bash
	~/.foundry/bin/foundryup

deploy:
	env ETHERSCAN_API_KEY=$(RODEO_ETHERSCAN_KEY) \
		forge script Deploy.s.sol:Deploy \
		--rpc-url $(RPC_URL)  \
		--private-key $(RODEO_PRIVATE_KEY_MAINNET) \
		--etherscan-api-key $(RODEO_ETHERSCAN_KEY) \
		--broadcast --verify -vvvv

# ~/.svm/0.8.15/solc-0.8.15 --version
verify:
	env ETHERSCAN_API_KEY=$(RODEO_ETHERSCAN_KEY) \
		forge verify-contract --watch \
		--chain-id $(CHAIN_ID) \
		--etherscan-api-key $(RODEO_ETHERSCAN_KEY) \
		--num-of-optimizations 200 \
		--compiler-version 0.8.17+commit.8df45f5f \
		--constructor-args $(shell cast abi-encode "constructor(address)" "0x780D46fef77ac5f83399BD2BE363125982A78973") \
		0x768778aB1B2c4E462172136eE2584Ea7494bcB81 \
		PositionManager.sol:PositionManager
