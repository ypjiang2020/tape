import numpy as np
import matplotlib.pyplot as plt


ss = [x/10 for x in range(0, 21, 2)] 
accounts=[1000, 10000]


for x in accounts:
    ax = [y for y in range(x)]
    for s in ss:
        sump = 0
        res = []
        for i in range(x):
            p = (1+i)**(-s)
            sump += p
            res.append(p)
        for i in range(x):
            res[i] /= sump
        y = []
        for i in range(x):
            y.append(np.sum(res[:i+1]))
        plt.plot(ax, y, label= "s=%.2f" % s )
        
    plt.legend()
    plt.savefig("zipf-%d.pdf"%x)
    plt.cla()


# x = [x for x in range(1000)]
# y = [func(i, 2, 2) for i in x]
# plt.plot(x, y)
