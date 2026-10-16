words = ["alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"]
counts = {}
for i in range(1, 500001):
    w = words[i % 8]
    counts[w] = counts.get(w, 0) + 1
print(len(counts))
