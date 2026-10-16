import functools

items = list(range(1, 300001))
total = 0
for r in range(5):
    doubled = list(filter(lambda x: x % 3 == 0, map(lambda x: x * 2, items)))
    total += functools.reduce(lambda a, b: a + b, doubled, 0)
print(total)
