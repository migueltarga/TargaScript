items = []
for i in range(1, 200001):
    items.append({"id": i, "score": (i * 7919) % 1000})
s = sorted(items, key=lambda x: x["score"])
print(s[0]["score"], sum(x["score"] for x in items))
