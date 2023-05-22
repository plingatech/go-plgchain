#!/bin/sh

set -e

GO_PLGCHAIN_BIN=./go-plgchain
CHAIN_CUSTOM_OPTIONS=$(tr "\n" " " << EOL
--block-gas-limit 10000000
--epoch-size 10
--chain-id 51001
--name go-plgchain-docker
--premine 0x228466F2C715CbEC05dEAbfAc040ce3619d7CF0B:0xD3C21BCECCEDA1000000
--premine 0xca48694ebcB2548dF5030372BE4dAad694ef174e:0xD3C21BCECCEDA1000000
EOL
)

case "$1" in

   "init")
      case "$2" in 
         "ibft")
         if [ -f "$GENESIS_PATH" ]; then
              echo "Secrets have already been generated."
         else
              echo "Generating secrets..."
              secrets=$("$GO_PLGCHAIN_BIN" secrets init --insecure --num 4 --data-dir /data/data- --json)
              echo "Secrets have been successfully generated"
              echo "Generating IBFT Genesis file..."
              cd /data && /go-plgchain/go-plgchain genesis $CHAIN_CUSTOM_OPTIONS \
                --dir genesis.json \
                --consensus ibft \
                --ibft-validators-prefix-path data- \
                --validator-set-size=4 \
                --bootnode "/dns4/node-1/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[0] | .node_id')" \
                --bootnode "/dns4/node-2/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[1] | .node_id')"
         fi
              ;;
          "plgbft")
              echo "Generating PlgBFT secrets..."
              secrets=$("$GO_PLGCHAIN_BIN" plgbft-secrets init --insecure --num 4 --data-dir /data/data- --json)
              echo "Secrets have been successfully generated"

              echo "Generating manifest..."
              "$GO_PLGCHAIN_BIN" manifest --path /data/manifest.json --validators-path /data --validators-prefix data-

              echo "Generating PlgBFT Genesis file..."
              "$GO_PLGCHAIN_BIN" genesis $CHAIN_CUSTOM_OPTIONS \
                --dir /data/genesis.json \
                --consensus plgbft \
                --manifest /data/manifest.json \
                --validator-set-size=4 \
                --bootnode "/dns4/node-1/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[0] | .node_id')" \
                --bootnode "/dns4/node-2/tcp/1478/p2p/$(echo "$secrets" | jq -r '.[1] | .node_id')"
              ;;
      esac
      ;;

   *)
      echo "Executing go-plgchain..."
      exec "$GO_PLGCHAIN_BIN" "$@"
      ;;

esac
