points = []
for i in range(1, 100001):
    points.append({"x": i, "y": i * 2, "name": f"p{i}"})
s = 0
for r in range(10):
    for p in points:
        s += p["x"] + p["y"]
print(s)
