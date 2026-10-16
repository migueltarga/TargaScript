local items = {}
for i = 1, 300000 do items[i] = i end
local total = 0
for r = 1, 5 do
  local doubled = {}
  for i = 1, #items do doubled[i] = items[i] * 2 end
  local filtered = {}
  for i = 1, #doubled do
    if doubled[i] % 3 == 0 then filtered[#filtered + 1] = doubled[i] end
  end
  for i = 1, #filtered do total = total + filtered[i] end
end
print(total)
