const words = ['alpha', 'beta', 'gamma', 'delta', 'epsilon', 'zeta', 'eta', 'theta'];
const counts = {};
for (let i = 1; i <= 500000; i++) {
  const w = words[i % 8];
  counts[w] = (counts[w] ?? 0) + 1;
}
console.log(Object.keys(counts).length);
