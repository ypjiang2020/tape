# tape

# Build
`go build ./cmd/tape`

# Run
1. Link to crypto materials: `ln -sf $YOUR_PROJECT/organizations`
2. End-to-End Run     
    Set `e2e: true` in config.yaml. 
    ```bash
    # if(ACCOUNTS not exists):
    #   create ACCOUNTS  
    # else:
    #   send transactions (i.e. transfer money from A to B) 
    # end

    # typically 
    rm ACCOUNTS # clean old accounts
    ./tape -c config.yaml -n 1000  # create 1000 accounts according config.yaml
    ./tape -c config.yaml -n 10000  # send 10000 transactions using ACCOUNTS
    ```
3. Breakdown      
    Set `e2e: false` in config.yaml. 
    ```bash
    # if(EDNORSEMENT not exists):
    #   start phase1 to create ENDORSEMENT: send proposals to endorsements 
    # else:
    #   start phase2 to broadcast transactions to orderer (read from ENDORSEMENT)
    # end

    # typically 
    rm ENDORSEMENT # clean old accounts
    ./tape -c config.yaml -n 10000  # create 10000 endorsements
    ./tape -c config.yaml -n 10000  # broadcast 10000 transactions 
    ```
# Result
Save output to file for analysis: `./tape -c config.yaml -n 10000  > log.transactions `

## latency breakdown
```
cat log.transactions | python3 scripts/latency.py > latency.log  
```

**Output format:** txid [#1, #2, #2]
1. endorseement: clients sends proposal => client receives enough endorsement
2. local_process: clients generate signed transaction based on endorsements
3. consensus & commit: clients send signed transaction => clients receive response (including consensus, validation, and commit)


## conflict rate
```bash
bash scripts/conflict.sh
```

# TODO
1. zipfan distribution workload
2. add prometheus 
