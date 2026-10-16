class Counter:
    def __init__(self):
        self.count = 0

    def add(self, n):
        self.count += n


c = Counter()
for i in range(1, 1000001):
    c.add(i)
print(c.count)
