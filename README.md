# tape

# Build
- local: `go build ./cmd/tape`       
- docker: `docker build -f Dockerfile -t tape .`

# Run
1. Link to crypto materials: `ln -sf $YOUR_PROJECT/organizations`
2. End-to-End Run     
    ```bash
    # create 2000 accounts
    ./tape -c config.yaml --txtype put --endorser_group 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --orderer_client 5 --num_of_conn 4 --client_per_conn 4 
    # start 2000 payment transactions
    ./tape -c config.yaml --txtype conflict --endorser_group 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --orderer_client 5 --num_of_conn 4 --client_per_conn 4 
    ```
3. Breakdown      
    ```bash
    # phase1
    ./tape --no-e2e -c config.yaml --txtype put --endorser_group 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --orderer_client 5 --num_of_conn 4 --client_per_conn 4 
    # phase2
    ./tape --no-e2e -c config.yaml --txtype put --endorser_group 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --orderer_client 5 --num_of_conn 4 --client_per_conn 4 
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
