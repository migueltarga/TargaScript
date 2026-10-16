local items = {}
for i = 1, 200000 do items[i] = {id = i, score = (i * 7919) % 1000} end
local sorted = {}
for i = 1, #items do sorted[i] = items[i] end
table.sort(sorted, function(a, b) return a.score < b.score end)
local sum = 0
for i = 1, #items do sum = sum + items[i].score end
print(sorted[1].score, sum)
