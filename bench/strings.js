const parts = [];
for (let i = 1; i <= 200000; i++) parts.push(`item-${i}`);
const text = parts.join(',');
console.log(text.split(',').length, text.length);
