local points = {}
for i = 1, 100000 do points[i] = {x = i, y = i * 2, name = 'p' .. i} end
local sum = 0
for r = 1, 10 do
  for _, p in ipairs(points) do sum = sum + p.x + p.y end
end
print(sum)
