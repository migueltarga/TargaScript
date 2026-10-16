local words = {'alpha', 'beta', 'gamma', 'delta', 'epsilon', 'zeta', 'eta', 'theta'}
local counts = {}
for i = 1, 500000 do
  local w = words[(i % 8) + 1]
  counts[w] = (counts[w] or 0) + 1
end
local count = 0
for _ in pairs(counts) do count = count + 1 end
print(count)
