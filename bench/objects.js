const items = [];
for (let i = 1; i <= 200000; i++) items.push({id: i, score: (i * 7919) % 1000});
const sorted = [...items].sort((a, b) => a.score - b.score);
console.log(sorted[0].score, items.map(x => x.score).reduce((a, b) => a + b, 0));
