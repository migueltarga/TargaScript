parts = []
for i in range(1, 200001):
    parts.append(f"item-{i}")
text = ",".join(parts)
print(len(text.split(",")), len(text))
