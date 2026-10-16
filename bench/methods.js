function makeCounter() { return {count: 0, add(n) { this.count += n; }}; }
const c = makeCounter();
for (let i = 1; i <= 1000000; i++) c.add(i);
console.log(c.count);
