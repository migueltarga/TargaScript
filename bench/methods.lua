local function makeCounter()
  return {count = 0, add = function(self, n) self.count = self.count + n end}
end
local c = makeCounter()
for i = 1, 1000000 do c:add(i) end
print(c.count)
