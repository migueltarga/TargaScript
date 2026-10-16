const points = [];
for (let i = 1; i <= 100000; i++) points.push({x: i, y: i * 2, name: `p${i}`});
let sum = 0;
for (let r = 1; r <= 10; r++) for (const p of points) sum += p.x + p.y;
console.log(sum);
