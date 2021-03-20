import numpy as np 
import sys
import matplotlib.pyplot as plt

times = {}
log = {}
ids = []

fig = {}
for i in range(3):
    fig[i] = []

for line in sys.stdin:
    temp = line.split()
    if "start: " in line or "proposal: " in line or "sent: " in line or "end: " in line:
        if '_' in temp[2]:
            idx = int(temp[2].split('_')[0])
        else:
            idx = int(temp[2], 16)

    if "start: " in line:
        times[idx] = int(temp[1])
        log[idx] = []
    if "proposal: " in line:
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[0].append((float(temp[1]) - times[idx])/1e6)
        # print(temp[2], "endorsement_breakdown", (float(temp[1]) - times[temp[2]])/1e6)
        times[idx] = int(temp[1])
    if "sent: " in line:
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[1].append((float(temp[1]) - times[idx])/1e6)
        # print(temp[2], "before_order", (float(temp[1]) - times[temp[2]])/1e6)
        times[idx] = int(temp[1])
    if "end: " in line:
        ids.append(idx)
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[2].append((float(temp[1]) - times[idx])/1e6)
        # print(temp[2], "consensus_&_commit", (float(temp[1]) - times[temp[2]])/1e6)

for tx in ids:
    print(tx, log[tx])

for i in range(3):
    plt.plot(fig[i], label = str(i))
plt.xlabel("time")
plt.ylabel("latency (ms)")
plt.title(" latency")
plt.legend()
plt.savefig("latency_ori_fifo.pdf")