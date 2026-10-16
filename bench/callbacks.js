const items = Array.from({length: 300000}, (_, i) => i + 1);
let total = 0;
for (let r = 1; r <= 5; r++) {
  const doubled = items.map(x => x * 2).filter(x => x % 3 === 0);
  total += doubled.reduce((a, b) => a + b, 0);
}
console.log(total);
