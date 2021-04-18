import numpy as np 
import sys
import matplotlib.pyplot as plt

times = {}
log = {}
ids = []

fig = {}
for i in range(3):
    fig[i] = []

labels = ['endorsement', 'assemble_endorsement', 'consensus&commit']
for line in sys.stdin:
    temp = line.split()
    if "start: " in line or "proposal: " in line or "sent: " in line or "end: " in line:
        if '_' in temp[2]:
            idx = int(temp[2].split('_')[0])
        else: 
            idx = int(temp[2], 16)

    if "start: " in line:
        ids.append(idx)
        times[idx] = int(temp[1])
        log[idx] = [int(temp[3]), int(temp[4])] # clientid, connectionid
    if "proposal: " in line:
        # breakdown_endorsement
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[0].append((float(temp[1]) - times[idx])/1e6)
        times[idx] = int(temp[1])
    if "sent: " in line:
        # breakdown_asemble_endorsment(not important)
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[1].append((float(temp[1]) - times[idx])/1e6)
        times[idx] = int(temp[1])
    if "end: " in line:
        # breakdown_consensus&commit
        log[idx].append((float(temp[1]) - times[idx])/1e6)
        fig[2].append((float(temp[1]) - times[idx])/1e6)

# for tx in ids:
    # print(tx, log[tx])

# for i in range(3):
#     plt.plot(fig[i], label = labels[i])
# plt.xlabel("time")
# plt.ylabel("latency (ms)")
# plt.title("breakdown")
# plt.legend()
# plt.savefig("latency_breakdown.pdf")

for i in range(3):
    print(labels[i])
    fig[i].sort()
    print("\tmean", np.mean(fig[i]))
    print("\t99_tail", fig[i][int(len(fig[i]) * 0.99)])
    print("\t95_tail", fig[i][int(len(fig[i]) * 0.95)])
    print("\t90_tail", fig[i][int(len(fig[i]) * 0.90)])
    print("\t50_tail", fig[i][int(len(fig[i]) * 0.50)])
