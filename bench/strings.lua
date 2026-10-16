local parts = {}
for i = 1, 200000 do parts[i] = 'item-' .. i end
local text = table.concat(parts, ',')
local count = 0
for _ in text:gmatch('[^,]+') do count = count + 1 end
print(count, #text)
